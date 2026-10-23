package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type AIAnalysisItem struct {
	TaskID       string     `json:"taskId"`
	Title        string     `json:"title,omitempty"`
	RelatedURL   string     `json:"relatedUrl,omitempty"`
	Summary      string     `json:"summary"`
	NextSteps    []string   `json:"nextSteps"`
	DocumentHash string     `json:"documentHash"`
	Decision     string     `json:"decision"`
	ReviewerName string     `json:"reviewerName,omitempty"`
	ReviewedAt   *time.Time `json:"reviewedAt,omitempty"`
	Version      int        `json:"version"`
}

type AIAnalysisBatch struct {
	ID           string           `json:"id"`
	WorkspaceID  int              `json:"workspaceId"`
	PlanDate     string           `json:"planDate"`
	SnapshotHash string           `json:"snapshotHash"`
	PromptHash   string           `json:"promptHash"`
	ModelMode    string           `json:"modelMode"`
	ModelProfile string           `json:"modelProfile"`
	GeneratedAt  time.Time        `json:"generatedAt"`
	Stale        bool             `json:"stale"`
	Finalized    bool             `json:"finalized,omitempty"`
	PlanCreated  bool             `json:"planCreated,omitempty"`
	Items        []AIAnalysisItem `json:"items"`
}

func AnalysisBatchID(b AIAnalysisBatch) string {
	items := append([]AIAnalysisItem(nil), b.Items...)
	sort.Slice(items, func(i, j int) bool { return items[i].TaskID < items[j].TaskID })
	parts := []string{"analysis-v1", fmt.Sprint(b.WorkspaceID), b.PlanDate, b.SnapshotHash, b.PromptHash, b.ModelMode, b.ModelProfile}
	for _, item := range items {
		parts = append(parts, item.TaskID, item.DocumentHash)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func (db *Database) ImportAIAnalysis(ctx context.Context, b AIAnalysisBatch) error {
	if b.ID != AnalysisBatchID(b) || len(b.Items) > 20 || b.WorkspaceID <= 0 || b.GeneratedAt.IsZero() {
		return ErrAIPlanInvalid
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockAISources(ctx, tx); err != nil {
		return err
	}
	snap, err := aiSnapshotTx(ctx, tx, b.WorkspaceID)
	if err != nil {
		return err
	}
	hash, err := AISnapshotHash(snap)
	if err != nil {
		return err
	}
	if hash != b.SnapshotHash {
		return ErrAIPlanStale
	}
	actionable, selected := selectedAITasks(snap, b.PlanDate)
	if len(b.Items) != min(20, len(actionable)) {
		return ErrAIPlanInvalid
	}
	seen := map[string]bool{}
	available := false
	for _, item := range b.Items {
		if !selected[item.TaskID] || seen[item.TaskID] || len(item.Summary) > 1000 || len(item.NextSteps) > 8 || len(item.DocumentHash) != 64 || item.Version != 0 || item.Decision != "PENDING" && item.Decision != "UNAVAILABLE" {
			return ErrAIPlanInvalid
		}
		if item.Decision == "PENDING" && (strings.TrimSpace(item.Summary) == "" || len(item.NextSteps) == 0) {
			return ErrAIPlanInvalid
		}
		for _, step := range item.NextSteps {
			if strings.TrimSpace(step) == "" || len([]rune(step)) > 250 {
				return ErrAIPlanInvalid
			}
		}
		seen[item.TaskID] = true
	}
	if _, err = tx.ExecContext(ctx, `UPDATE ai_analysis_batches SET finalized_at=now() WHERE workspace_id=$1 AND plan_date=$2 AND id<>$3 AND finalized_at IS NULL`, b.WorkspaceID, b.PlanDate, b.ID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO ai_analysis_batches(id,workspace_id,plan_date,snapshot_hash,prompt_hash,model_mode,model_profile,generated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`, b.ID, b.WorkspaceID, b.PlanDate, b.SnapshotHash, b.PromptHash, b.ModelMode, b.ModelProfile, b.GeneratedAt)
	if err != nil {
		return err
	}
	for _, item := range b.Items {
		steps, _ := json.Marshal(item.NextSteps)
		_, err = tx.ExecContext(ctx, `INSERT INTO ai_task_analyses(batch_id,task_id,summary,next_steps,document_hash,decision) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, b.ID, item.TaskID, item.Summary, steps, item.DocumentHash, item.Decision)
		if err != nil {
			return err
		}
		var same bool
		err = tx.QueryRowContext(ctx, `SELECT summary=$3 AND next_steps=$4::jsonb AND document_hash=$5 FROM ai_task_analyses WHERE batch_id=$1 AND task_id=$2`, b.ID, item.TaskID, item.Summary, steps, item.DocumentHash).Scan(&same)
		if err != nil || !same {
			return ErrAIPlanConflict
		}
		if item.Decision == "PENDING" {
			available = true
		}
	}
	if !available {
		if _, err = tx.ExecContext(ctx, `UPDATE ai_analysis_batches SET finalized_at=COALESCE(finalized_at,now()) WHERE id=$1`, b.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *Database) AIAnalysis(ctx context.Context, workspaceID int, batchID string) (AIAnalysisBatch, error) {
	var b AIAnalysisBatch
	query := `SELECT id,workspace_id,plan_date::text,snapshot_hash,prompt_hash,model_mode,model_profile,generated_at,finalized_at IS NOT NULL,EXISTS(SELECT 1 FROM ai_plan_drafts d WHERE d.draft->>'analysisBatchId'=ai_analysis_batches.id::text) FROM ai_analysis_batches WHERE workspace_id=$1`
	args := []any{workspaceID}
	if batchID != "" {
		query += ` AND id=$2`
		args = append(args, batchID)
	} else {
		query += ` ORDER BY created_at DESC,id DESC LIMIT 1`
	}
	err := db.Conn.QueryRowContext(ctx, query, args...).Scan(&b.ID, &b.WorkspaceID, &b.PlanDate, &b.SnapshotHash, &b.PromptHash, &b.ModelMode, &b.ModelProfile, &b.GeneratedAt, &b.Finalized, &b.PlanCreated)
	if err != nil {
		return b, err
	}
	rows, err := db.Conn.QueryContext(ctx, `SELECT a.task_id::text,COALESCE(t.title,''),COALESCE(t.related_url,''),a.summary,a.next_steps,a.document_hash,a.decision,a.reviewer_name,a.reviewed_at,a.version FROM ai_task_analyses a LEFT JOIN work_items t ON t.id=a.task_id AND t.division_id=$2 WHERE a.batch_id=$1 ORDER BY a.task_id`, b.ID, b.WorkspaceID)
	if err != nil {
		return b, err
	}
	defer rows.Close()
	b.Items = []AIAnalysisItem{}
	for rows.Next() {
		var item AIAnalysisItem
		var steps []byte
		var at sql.NullTime
		if err = rows.Scan(&item.TaskID, &item.Title, &item.RelatedURL, &item.Summary, &steps, &item.DocumentHash, &item.Decision, &item.ReviewerName, &at, &item.Version); err != nil {
			return b, err
		}
		if err = json.Unmarshal(steps, &item.NextSteps); err != nil {
			return b, err
		}
		if at.Valid {
			item.ReviewedAt = &at.Time
		}
		b.Items = append(b.Items, item)
	}
	return b, rows.Err()
}

func (db *Database) ReviewAIAnalysis(ctx context.Context, workspaceID int, batchID, taskID, userID, decision string, version int) error {
	if decision != "APPROVED" && decision != "REJECTED" {
		return ErrAIPlanInvalid
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockAISources(ctx, tx); err != nil {
		return err
	}
	var snapshotHash string
	err = tx.QueryRowContext(ctx, `SELECT snapshot_hash FROM ai_analysis_batches WHERE id=$1 AND workspace_id=$2 AND finalized_at IS NULL FOR UPDATE`, batchID, workspaceID).Scan(&snapshotHash)
	if err != nil {
		return err
	}
	snap, err := aiSnapshotTx(ctx, tx, workspaceID)
	if err != nil {
		return err
	}
	hash, err := AISnapshotHash(snap)
	if err != nil {
		return err
	}
	if hash != snapshotHash {
		return ErrAIPlanStale
	}
	var name string
	if err = tx.QueryRowContext(ctx, `SELECT u.name FROM users u JOIN workspace_access a ON a.user_id=u.id AND a.division_id=$2 WHERE u.id=$1 AND u.is_active FOR SHARE OF u,a`, userID, workspaceID).Scan(&name); errors.Is(err, sql.ErrNoRows) {
		return ErrWorkspaceForbidden
	} else if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE ai_task_analyses SET decision=$5,reviewer_id=$4,reviewer_identity=$4::text,reviewer_name=$6,reviewed_at=now(),version=version+1 WHERE batch_id=$1 AND task_id=$2 AND version=$3 AND decision='PENDING'`, batchID, taskID, version, userID, decision, name)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ErrAIPlanConflict
	}
	if _, err = tx.ExecContext(ctx, `UPDATE ai_analysis_batches SET finalized_at=now() WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM ai_task_analyses WHERE batch_id=$1 AND decision IN ('PENDING','APPROVED'))`, batchID); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *Database) ReadyAIAnalyses(ctx context.Context, workspaceID int) ([]AIAnalysisBatch, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT id FROM ai_analysis_batches b WHERE workspace_id=$1 AND finalized_at IS NULL AND NOT EXISTS(SELECT 1 FROM ai_task_analyses a WHERE a.batch_id=b.id AND a.decision='PENDING') AND EXISTS(SELECT 1 FROM ai_task_analyses a WHERE a.batch_id=b.id AND a.decision='APPROVED') ORDER BY created_at LIMIT 20`, workspaceID)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return nil, err
	}
	batches := make([]AIAnalysisBatch, 0, len(ids))
	for _, id := range ids {
		b, e := db.AIAnalysis(ctx, workspaceID, id)
		if e != nil {
			return nil, e
		}
		batches = append(batches, b)
	}
	return batches, nil
}
