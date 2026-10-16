package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"my-ple-backend/model"
)

var ErrAIPlanStale = errors.New("AI draft source changed")
var ErrAIPlanConflict = errors.New("AI draft or review conflict")
var ErrAIPlanInvalid = errors.New("invalid AI draft")

type AISelection struct {
	Actionable int `json:"actionable"`
	Selected   int `json:"selected"`
	Deferred   int `json:"deferred"`
}
type AIRecommendation struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	SourceTaskIDs      []string `json:"sourceTaskIds"`
	RecommendationType string   `json:"recommendationType"`
	SuggestedDueDate   string   `json:"suggestedDueDate"`
	Reason             string   `json:"reason"`
	AssigneeIDs        []string `json:"assigneeIds,omitempty"`
}
type AIDraft struct {
	SchemaVersion     string             `json:"schemaVersion"`
	ID                string             `json:"id"`
	WorkspaceID       int                `json:"workspaceId"`
	PlanDate          string             `json:"planDate"`
	Status            string             `json:"status"`
	EvidenceStatus    string             `json:"evidenceStatus"`
	GeneratedAt       time.Time          `json:"generatedAt"`
	InputSnapshotHash string             `json:"inputSnapshotHash"`
	PromptVersion     string             `json:"promptVersion"`
	PromptHash        string             `json:"promptHash"`
	ModelMode         string             `json:"modelMode"`
	ModelProfile      string             `json:"modelProfile"`
	Selection         *AISelection       `json:"selection"`
	Recommendations   []AIRecommendation `json:"recommendations"`
}
type AIReview struct {
	Decision     string    `json:"decision"`
	Note         string    `json:"note"`
	ReviewerID   string    `json:"reviewerId"`
	ReviewerName string    `json:"reviewerName"`
	ReviewedAt   time.Time `json:"reviewedAt"`
	Version      int       `json:"version"`
}
type AIPlanRecommendation struct {
	AIRecommendation
	SourceEvidence []model.AITask `json:"sourceEvidence"`
	Review         AIReview       `json:"review"`
	History        []AIReview     `json:"history"`
}
type AIPlan struct {
	Draft           AIDraft                `json:"draft"`
	Recommendations []AIPlanRecommendation `json:"recommendations"`
	Stale           bool                   `json:"stale"`
	Reviewed        int                    `json:"reviewed"`
}
type AIPlanSummary struct {
	ID          string    `json:"id"`
	PlanDate    string    `json:"planDate"`
	GeneratedAt time.Time `json:"generatedAt"`
	ModelMode   string    `json:"modelMode"`
	Stale       bool      `json:"stale"`
}

func AISnapshotHash(s model.AISnapshot) (string, error) {
	tasks := append([]model.AITask{}, s.Tasks...)
	for i := range tasks {
		tasks[i].AssigneeIDs = append([]string(nil), tasks[i].AssigneeIDs...)
		sort.Strings(tasks[i].AssigneeIDs)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	assignees := append([]string(nil), s.AllowedAssigneeIDs...)
	sort.Strings(assignees)
	raw, err := json.Marshal(struct {
		Tasks     []model.AITask
		Assignees []string
	}{tasks, assignees})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// SHARE locks cover existing rows and inserts, including writes outside Workspace handlers.
func lockAISources(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `LOCK TABLE jobs, work_items IN SHARE MODE`)
	return err
}

func (db *Database) ImportAIPlan(ctx context.Context, d AIDraft) error {
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockAISources(ctx, tx); err != nil {
		return err
	}
	snap, err := aiSnapshotTx(ctx, tx, d.WorkspaceID)
	if err != nil {
		return err
	}
	hash, err := AISnapshotHash(snap)
	if err != nil {
		return err
	}
	if hash != d.InputSnapshotHash {
		return ErrAIPlanStale
	}
	actionable := []model.AITask{}
	for _, t := range snap.Tasks {
		if t.StartDate == "" || t.StartDate <= d.PlanDate {
			actionable = append(actionable, t)
		}
	}
	if d.Selection == nil || d.Selection.Actionable != len(actionable) || d.Selection.Selected != min(20, len(actionable)) || d.Selection.Deferred != len(actionable)-d.Selection.Selected {
		return ErrAIPlanInvalid
	}
	approaching, _ := time.Parse("2006-01-02", d.PlanDate)
	near := approaching.AddDate(0, 0, 3).Format("2006-01-02")
	score := func(t model.AITask) int {
		n := map[string]int{"URGENT": 80, "HIGH": 40, "MEDIUM": 15}[t.Priority]
		if t.DueDate != "" && t.DueDate < d.PlanDate {
			n += 100
		} else if t.DueDate != "" && t.DueDate <= near {
			n += 30
		}
		if t.RequiresQA || t.Status == "NEED_REVISION" || t.Status == "REVISION" {
			n += 35
		}
		if t.Status == "IN_PROGRESS" || t.Status == "NEED_REVISION" || t.Status == "REVISION" {
			n += 10
		}
		return n
	}
	sort.Slice(actionable, func(i, j int) bool {
		a, b := actionable[i], actionable[j]
		if score(a) != score(b) {
			return score(a) > score(b)
		}
		if a.DueDate != b.DueDate {
			return a.DueDate != "" && (b.DueDate == "" || a.DueDate < b.DueDate)
		}
		return a.ID < b.ID
	})
	candidates := map[string]bool{}
	for i := 0; i < min(20, len(actionable)); i++ {
		candidates[actionable[i].ID] = true
	}
	evidence := map[string]model.AITask{}
	for _, t := range snap.Tasks {
		evidence[t.ID] = t
	}
	signatures := map[string]bool{}
	for _, r := range d.Recommendations {
		if len(r.SourceTaskIDs) != 1 || !candidates[r.SourceTaskIDs[0]] || signatures[r.RecommendationType+"|"+r.SourceTaskIDs[0]] {
			return ErrAIPlanInvalid
		}
		signatures[r.RecommendationType+"|"+r.SourceTaskIDs[0]] = true
		t := evidence[r.SourceTaskIDs[0]]
		if r.Priority != t.Priority || r.RecommendationType == "ADDRESS_OVERDUE" && (t.DueDate == "" || t.DueDate >= d.PlanDate) || r.RecommendationType == "QA_OR_REVISION" && !(t.RequiresQA || t.Status == "NEED_REVISION" || t.Status == "REVISION") || len(r.AssigneeIDs) > 0 {
			return ErrAIPlanInvalid
		}
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return err
	}
	var identical bool
	err = tx.QueryRowContext(ctx, `SELECT draft=$2::jsonb FROM ai_plan_drafts WHERE id=$1 FOR UPDATE`, d.ID, raw).Scan(&identical)
	if err == nil {
		if !identical {
			return ErrAIPlanConflict
		}
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO ai_plan_drafts(id,workspace_id,plan_date,snapshot_hash,draft) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, d.ID, d.WorkspaceID, d.PlanDate, hash, raw)
	if err != nil {
		return err
	}
	// Concurrent import may have won the unique constraint; compare its immutable payload.
	if err = tx.QueryRowContext(ctx, `SELECT draft=$2::jsonb FROM ai_plan_drafts WHERE id=$1`, d.ID, raw).Scan(&identical); err != nil {
		return err
	}
	if !identical {
		return ErrAIPlanConflict
	}
	for _, r := range d.Recommendations {
		refs := make([]model.AITask, 0, len(r.SourceTaskIDs))
		for _, id := range r.SourceTaskIDs {
			refs = append(refs, evidence[id])
		}
		item, _ := json.Marshal(r)
		source, _ := json.Marshal(refs)
		if _, err = tx.ExecContext(ctx, `INSERT INTO ai_plan_recommendations(draft_id,id,data,source_evidence) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, d.ID, r.ID, item, source); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *Database) AIPlanHistory(ctx context.Context, workspaceID int, limit, offset int) ([]AIPlanSummary, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT id,plan_date::text,(draft->>'generatedAt')::timestamptz,draft->>'modelMode',snapshot_hash FROM ai_plan_drafts WHERE workspace_id=$1 ORDER BY plan_date DESC,imported_at DESC,id LIMIT $2 OFFSET $3`, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []AIPlanSummary{}
	storedHashes := []string{}
	for rows.Next() {
		var p AIPlanSummary
		var stored string
		if err = rows.Scan(&p.ID, &p.PlanDate, &p.GeneratedAt, &p.ModelMode, &stored); err != nil {
			return nil, err
		}
		result = append(result, p)
		storedHashes = append(storedHashes, stored)
	}
	if err = rows.Err(); err != nil || len(result) == 0 {
		return result, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	snap, err := db.AISnapshot(ctx, workspaceID)
	if err != nil && !errors.Is(err, ErrAISnapshotTooLarge) {
		return nil, err
	}
	hash := ""
	if err == nil {
		hash, err = AISnapshotHash(snap)
		if err != nil {
			return nil, err
		}
	}
	for i := range result {
		result[i].Stale = storedHashes[i] != hash
	}
	return result, nil
}

func (db *Database) AIPlanIDByDate(ctx context.Context, workspaceID int, date string) (string, error) {
	var id string
	err := db.Conn.QueryRowContext(ctx, `SELECT id FROM ai_plan_drafts WHERE workspace_id=$1 AND plan_date=$2 ORDER BY imported_at DESC,id DESC LIMIT 1`, workspaceID, date).Scan(&id)
	return id, err
}

func (db *Database) AIPlanDetail(ctx context.Context, workspaceID int, id string) (AIPlan, error) {
	var p AIPlan
	var raw []byte
	var hash string
	err := db.Conn.QueryRowContext(ctx, `SELECT draft,snapshot_hash FROM ai_plan_drafts WHERE workspace_id=$1 AND id=$2`, workspaceID, id).Scan(&raw, &hash)
	if err != nil {
		return p, err
	}
	if err = json.Unmarshal(raw, &p.Draft); err != nil {
		return p, err
	}
	snap, err := db.AISnapshot(ctx, workspaceID)
	if err != nil && !errors.Is(err, ErrAISnapshotTooLarge) {
		return p, err
	}
	current := ""
	if err == nil {
		current, err = AISnapshotHash(snap)
		if err != nil {
			return p, err
		}
	}
	p.Stale = current != hash
	rows, err := db.Conn.QueryContext(ctx, `SELECT r.data,r.source_evidence,r.decision,r.note,COALESCE(NULLIF(r.reviewer_identity,''),r.reviewer_id::text,''),CASE WHEN r.version=0 THEN '' ELSE COALESCE(NULLIF(r.reviewer_name,''),u.name,'Pengguna dihapus') END,r.reviewed_at,r.version FROM ai_plan_recommendations r LEFT JOIN users u ON u.id=r.reviewer_id WHERE r.draft_id=$1 ORDER BY r.id`, id)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	p.Recommendations = []AIPlanRecommendation{}
	for rows.Next() {
		var item AIPlanRecommendation
		var data, evidence []byte
		var at sql.NullTime
		if err = rows.Scan(&data, &evidence, &item.Review.Decision, &item.Review.Note, &item.Review.ReviewerID, &item.Review.ReviewerName, &at, &item.Review.Version); err != nil {
			return p, err
		}
		if err = json.Unmarshal(data, &item.AIRecommendation); err != nil {
			return p, err
		}
		if err = json.Unmarshal(evidence, &item.SourceEvidence); err != nil {
			return p, err
		}
		if at.Valid {
			item.Review.ReviewedAt = at.Time
		}
		if item.Review.Decision != "PENDING" {
			p.Reviewed++
		}
		item.History = []AIReview{}
		p.Recommendations = append(p.Recommendations, item)
	}
	if err = rows.Err(); err != nil {
		return p, err
	}
	for i := range p.Recommendations {
		history, err := db.Conn.QueryContext(ctx, `SELECT h.decision,h.note,COALESCE(NULLIF(h.reviewer_identity,''),h.reviewer_id::text,''),COALESCE(NULLIF(h.reviewer_name,''),u.name,'Pengguna dihapus'),h.reviewed_at,h.version FROM ai_plan_reviews h LEFT JOIN users u ON u.id=h.reviewer_id WHERE h.draft_id=$1 AND h.recommendation_id=$2 ORDER BY h.version DESC`, id, p.Recommendations[i].ID)
		if err != nil {
			return p, err
		}
		for history.Next() {
			var h AIReview
			if err = history.Scan(&h.Decision, &h.Note, &h.ReviewerID, &h.ReviewerName, &h.ReviewedAt, &h.Version); err != nil {
				history.Close()
				return p, err
			}
			p.Recommendations[i].History = append(p.Recommendations[i].History, h)
		}
		err = history.Err()
		history.Close()
		if err != nil {
			return p, err
		}
	}
	return p, nil
}

func (db *Database) ReviewAIPlan(ctx context.Context, workspaceID int, draftID, recID, userID, decision, note string, version int) (AIReview, error) {
	var review AIReview
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return review, err
	}
	defer tx.Rollback()
	var current int
	err = tx.QueryRowContext(ctx, `SELECT u.name FROM users u JOIN roles role ON role.id=u.role_id WHERE u.id=$1 AND u.is_active AND role.name IN ('Admin','Management') AND EXISTS(SELECT 1 FROM workspace_access a WHERE a.user_id=u.id AND a.division_id=$2 AND a.can_manage) FOR SHARE OF u,role`, userID, workspaceID).Scan(&review.ReviewerName)
	if errors.Is(err, sql.ErrNoRows) {
		return review, ErrWorkspaceForbidden
	}
	if err != nil {
		return review, err
	}
	var storedHash string
	err = tx.QueryRowContext(ctx, `SELECT r.version,d.snapshot_hash FROM ai_plan_recommendations r JOIN ai_plan_drafts d ON d.id=r.draft_id WHERE d.workspace_id=$1 AND r.draft_id=$2 AND r.id=$3 FOR UPDATE OF r`, workspaceID, draftID, recID).Scan(&current, &storedHash)
	if err != nil {
		return review, err
	}
	if current != version {
		return review, ErrAIPlanConflict
	}
	if err = lockAISources(ctx, tx); err != nil {
		return review, err
	}
	snap, err := aiSnapshotTx(ctx, tx, workspaceID)
	if errors.Is(err, ErrAISnapshotTooLarge) {
		return review, ErrAIPlanStale
	}
	if err != nil {
		return review, err
	}
	hash, err := AISnapshotHash(snap)
	if err != nil {
		return review, err
	}
	if hash != storedHash {
		return review, ErrAIPlanStale
	}
	err = tx.QueryRowContext(ctx, `UPDATE ai_plan_recommendations SET decision=$4,note=$5,reviewer_id=$6::uuid,reviewer_identity=$6::text,reviewer_name=$7,reviewed_at=now(),version=version+1 WHERE draft_id=$1 AND id=$2 AND version=$3 RETURNING reviewed_at,version`, draftID, recID, version, decision, note, userID, review.ReviewerName).Scan(&review.ReviewedAt, &review.Version)
	if err != nil {
		return review, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO ai_plan_reviews(draft_id,recommendation_id,version,decision,note,reviewer_id,reviewer_identity,reviewer_name,reviewed_at) VALUES($1,$2,$3,$4,$5,$6::uuid,$6::text,$7,$8)`, draftID, recID, review.Version, decision, note, userID, review.ReviewerName, review.ReviewedAt)
	if err != nil {
		return review, err
	}
	if err = tx.Commit(); err != nil {
		return review, err
	}
	review.Decision = decision
	review.Note = note
	review.ReviewerID = userID
	return review, nil
}
