package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"my-ple-backend/model"
)

func TestAIPlansPostgres(t *testing.T) {
	dsn := os.Getenv("AI_PLAN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AI_PLAN_TEST_DATABASE_URL absent")
	}
	u, e := url.Parse(dsn)
	if e != nil || u.Hostname() != "127.0.0.1" || !strings.Contains(u.Path, "ai_plan_test") {
		t.Fatal("isolated local AI plan database required")
	}
	conn, e := sql.Open("pgx", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	ctx := context.Background()
	database := NewDatabase(conn)
	if e = database.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	if e = database.Migrate(ctx); e != nil {
		t.Fatal("migration not idempotent:", e)
	}
	user := "550e8400-e29b-41d4-a716-446655440111"
	task := "550e8400-e29b-41d4-a716-446655440112"
	hashed, e := bcrypt.GenerateFromPassword([]byte("fixture-only"), bcrypt.MinCost)
	if e != nil {
		t.Fatal(e)
	}
	_, e = conn.ExecContext(ctx, `INSERT INTO users(id,name,email,role_id,username,password_hash) SELECT $1,'Fixture Manager','phase5-fixture@example.invalid',id,'phase5-fixture',$2 FROM roles WHERE name='Management' ON CONFLICT(id) DO UPDATE SET password_hash=EXCLUDED.password_hash`, user, string(hashed))
	if e != nil {
		t.Fatal(e)
	}
	var workspace int
	if e = conn.QueryRowContext(ctx, `SELECT id FROM divisions ORDER BY id LIMIT 1`).Scan(&workspace); e != nil {
		t.Fatal(e)
	}
	_, e = conn.ExecContext(ctx, `INSERT INTO work_items(id,division_id,title,description,due_date,status,priority,created_by) VALUES($1,$2,'Fixture task','Synthetic evidence','2026-10-15','TODO','HIGH',$3) ON CONFLICT(id) DO NOTHING`, task, workspace, user)
	if e != nil {
		t.Fatal(e)
	}
	snap, e := database.AISnapshot(ctx, workspace)
	if e != nil {
		t.Fatal(e)
	}
	hash, e := AISnapshotHash(snap)
	if e != nil {
		t.Fatal(e)
	}
	d := AIDraft{SchemaVersion: "1.0", WorkspaceID: workspace, PlanDate: "2026-10-16", Status: "PENDING_REVIEW", EvidenceStatus: "SUFFICIENT", GeneratedAt: time.Now().UTC(), InputSnapshotHash: hash, PromptVersion: "daily-v3", PromptHash: strings.Repeat("a", 64), ModelMode: "mock", ModelProfile: "mock-v1", Selection: &AISelection{Actionable: len(snap.Tasks), Selected: min(20, len(snap.Tasks)), Deferred: max(0, len(snap.Tasks)-20)}, Recommendations: []AIRecommendation{}}
	id := sha256.Sum256([]byte(strings.Join([]string{strconv.Itoa(workspace), d.PlanDate, hash, d.PromptVersion, d.PromptHash, d.ModelMode, d.ModelProfile, "1.0"}, "|")))
	d.ID = hex.EncodeToString(id[:])
	rid := sha256.Sum256([]byte(d.ID + "|" + task + "|ADDRESS_OVERDUE"))
	d.Recommendations = append(d.Recommendations, AIRecommendation{ID: hex.EncodeToString(rid[:]), Title: "Tindak lanjut fixture", Description: "Synthetic recommendation", Priority: "HIGH", SourceTaskIDs: []string{task}, RecommendationType: "ADDRESS_OVERDUE", SuggestedDueDate: d.PlanDate, Reason: "Fixture is overdue"})
	if e = database.ImportAIPlan(ctx, d); e != nil {
		t.Fatal(e)
	}
	if e = database.ImportAIPlan(ctx, d); e != nil {
		t.Fatal("duplicate import:", e)
	}
	bad := d
	bad.ModelProfile = "changed"
	if e = database.ImportAIPlan(ctx, bad); !errors.Is(e, ErrAIPlanConflict) {
		t.Fatal("conflicting import:", e)
	}
	var otherWorkspace int
	if e = conn.QueryRowContext(ctx, `SELECT id FROM divisions WHERE id<>$1 ORDER BY id LIMIT 1`, workspace).Scan(&otherWorkspace); e != nil {
		t.Fatal(e)
	}
	otherTask := "550e8400-e29b-41d4-a716-446655440999"
	if _, e = conn.ExecContext(ctx, `INSERT INTO work_items(id,division_id,title,status,priority,created_by) VALUES($1,$2,'Other workspace fixture','TODO','HIGH',$3)`, otherTask, otherWorkspace, user); e != nil {
		t.Fatal(e)
	}
	bad = d
	bad.Recommendations = append([]AIRecommendation(nil), d.Recommendations...)
	bad.Recommendations[0].SourceTaskIDs = []string{otherTask}
	if e = database.ImportAIPlan(ctx, bad); !errors.Is(e, ErrAIPlanInvalid) {
		t.Fatal("cross-workspace source reference:", e)
	}
	p, e := database.AIPlanDetail(ctx, workspace, d.ID)
	if e != nil || len(p.Recommendations) != 1 || p.Stale {
		t.Fatal("imported plan missing or stale:", e)
	}
	if _, e = database.ReviewAIPlan(ctx, workspace, d.ID, d.Recommendations[0].ID, user, "SUITABLE", "Fixture review", 0); e != nil {
		t.Fatal(e)
	}
	if _, e = database.ReviewAIPlan(ctx, workspace, d.ID, d.Recommendations[0].ID, user, "REJECTED", "", 0); !errors.Is(e, ErrAIPlanConflict) {
		t.Fatal("review conflict:", e)
	}
	if _, e = database.ReviewAIPlan(ctx, workspace, d.ID, d.Recommendations[0].ID, user, "NEEDS_REVISION", "Second review", 1); e != nil {
		t.Fatal(e)
	}
	p, e = database.AIPlanDetail(ctx, workspace, d.ID)
	if e != nil || p.Reviewed != 1 || len(p.Recommendations[0].History) != 2 || p.Recommendations[0].History[1].Decision != "SUITABLE" {
		t.Fatal("audit missing:", e)
	}
	admin := "550e8400-e29b-41d4-a716-446655440113"
	employee := "550e8400-e29b-41d4-a716-446655440114"
	for _, actor := range []struct{ id, role, name string }{{admin, "Admin", "Fixture Admin"}, {employee, "Project Engineer", "Fixture Employee"}} {
		_, e = conn.ExecContext(ctx, `INSERT INTO users(id,name,email,role_id,username,password_hash) SELECT $1::uuid,$2,$1::text||'@example.invalid',id,$1::text,$3 FROM roles WHERE name=$4`, actor.id, actor.name, string(hashed), actor.role)
		if e != nil {
			t.Fatal(e)
		}
	}
	if _, e = database.ReviewAIPlan(ctx, workspace, d.ID, d.Recommendations[0].ID, employee, "SUITABLE", "", 2); !errors.Is(e, ErrWorkspaceForbidden) {
		t.Fatal("ordinary employee reviewed:", e)
	}
	if _, e = database.ReviewAIPlan(ctx, workspace, d.ID, d.Recommendations[0].ID, admin, "SUITABLE", "Admin review", 2); e != nil {
		t.Fatal("admin review:", e)
	}
	if _, e = database.ReviewAIPlan(ctx, workspace, d.ID, d.Recommendations[0].ID, user, "NEEDS_REVISION", "Manager follow-up", 3); e != nil {
		t.Fatal("manager follow-up:", e)
	}
	var status string
	var version int
	if e = conn.QueryRowContext(ctx, `SELECT status,version FROM work_items WHERE id=$1`, task).Scan(&status, &version); e != nil || status != "TODO" || version != 1 {
		t.Fatal("Job Tracker changed:", e)
	}
	if _, e = conn.ExecContext(ctx, `UPDATE work_items SET version=version+1 WHERE id=$1`, task); e != nil {
		t.Fatal(e)
	}
	p, e = database.AIPlanDetail(ctx, workspace, d.ID)
	if e != nil || !p.Stale {
		t.Fatal("source change not flagged:", e)
	}
	if e = database.ImportAIPlan(ctx, d); !errors.Is(e, ErrAIPlanStale) {
		t.Fatal("stale import accepted:", e)
	}
	_, e = conn.ExecContext(ctx, `UPDATE work_items SET version=version-1 WHERE id=$1`, task)
	if e != nil {
		t.Fatal(e)
	}
	// Hold an uncommitted source edit until review waits on its table lock.
	sourceTx, e := conn.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = sourceTx.ExecContext(ctx, `UPDATE work_items SET version=version+1 WHERE id=$1`, task); e != nil {
		t.Fatal(e)
	}
	reviewDone := make(chan error, 1)
	go func() {
		_, reviewErr := database.ReviewAIPlan(ctx, workspace, d.ID, d.Recommendations[0].ID, user, "SUITABLE", "Concurrent review", 4)
		reviewDone <- reviewErr
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		if e = conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE relation='work_items'::regclass AND NOT granted)`).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("review did not wait for concurrent source edit")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if e = sourceTx.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = <-reviewDone; !errors.Is(e, ErrAIPlanStale) {
		t.Fatal("concurrent source edit was reviewed:", e)
	}
	var reviewCount int
	if e = conn.QueryRowContext(ctx, `SELECT count(*) FROM ai_plan_reviews WHERE draft_id=$1`, d.ID).Scan(&reviewCount); e != nil || reviewCount != 4 {
		t.Fatal("rejected review changed audit:", e, reviewCount)
	}
	if _, e = conn.ExecContext(ctx, `UPDATE work_items SET version=version-1 WHERE id=$1`, task); e != nil {
		t.Fatal(e)
	}
	lockTx, e := conn.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = lockAISources(ctx, lockTx); e != nil {
		t.Fatal(e)
	}
	writeDone := make(chan error, 1)
	go func() {
		_, writeErr := conn.ExecContext(ctx, `UPDATE work_items SET version=version+1 WHERE id=$1`, task)
		writeDone <- writeErr
	}()
	deadline = time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		if e = conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE relation='work_items'::regclass AND NOT granted)`).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("source edit did not wait for snapshot lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if e = lockTx.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = <-writeDone; e != nil {
		t.Fatal(e)
	}
	if _, e = conn.ExecContext(ctx, `UPDATE work_items SET version=version-1 WHERE id=$1`, task); e != nil {
		t.Fatal(e)
	}
	// A date lookup must find drafts beyond the first 20 history entries.
	for i := 0; i < 25; i++ {
		older := d
		older.PlanDate = time.Date(2026, 9, 1+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		older.Recommendations = []AIRecommendation{}
		older.EvidenceStatus = "INSUFFICIENT"
		digest := sha256.Sum256([]byte(strings.Join([]string{strconv.Itoa(workspace), older.PlanDate, hash, older.PromptVersion, older.PromptHash, older.ModelMode, older.ModelProfile, "1.0"}, "|")))
		older.ID = hex.EncodeToString(digest[:])
		raw, marshalErr := json.Marshal(older)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if _, e = conn.ExecContext(ctx, `INSERT INTO ai_plan_drafts(id,workspace_id,plan_date,snapshot_hash,draft) VALUES($1,$2,$3,$4,$5)`, older.ID, workspace, older.PlanDate, hash, raw); e != nil {
			t.Fatal(e)
		}
		if i == 0 {
			id = digest
		}
	}
	page, e := database.AIPlanHistory(ctx, workspace, 20, 0)
	if e != nil || len(page) != 20 {
		t.Fatal("history page:", e, len(page))
	}
	oldestID := hex.EncodeToString(id[:])
	for _, item := range page {
		if item.ID == oldestID {
			t.Fatal("oldest draft unexpectedly on first page")
		}
	}
	lookup, e := database.AIPlanIDByDate(ctx, workspace, "2026-09-01")
	if e != nil || lookup != oldestID {
		t.Fatal("date lookup missed older draft:", e)
	}
	// Exercise ordinary Workspace creation and editing separately from AI storage.
	created, e := database.SaveWorkItem(ctx, user, workspace, "", model.WorkItemInput{Title: "Workspace CRUD fixture", Description: "Fixture only", Status: "TODO", Priority: "MEDIUM", AssigneeIDs: []string{}})
	if e != nil {
		t.Fatal("workspace create:", e)
	}
	created.Title = "Workspace CRUD updated"
	updated, e := database.SaveWorkItem(ctx, user, workspace, created.ID, model.WorkItemInput{Title: created.Title, Description: created.Description, Status: created.Status, Priority: created.Priority, Version: created.Version})
	if e != nil || updated.Title != created.Title {
		t.Fatal("workspace update:", e)
	}
	if e = database.ArchiveWorkItem(ctx, user, updated.ID, updated.Version); e != nil {
		t.Fatal("workspace archive:", e)
	}
	var archivedVersion int
	if e = conn.QueryRowContext(ctx, `SELECT version FROM work_items WHERE id=$1`, updated.ID).Scan(&archivedVersion); e != nil {
		t.Fatal(e)
	}
	if _, e = database.RestoreWorkItem(ctx, user, updated.ID, archivedVersion); e != nil {
		t.Fatal("workspace restore:", e)
	}
	// Recreate the prior restrictive schema, then prove startup migration upgrades it twice.
	for _, table := range []string{"ai_plan_recommendations", "ai_plan_reviews"} {
		if _, e = conn.ExecContext(ctx, `ALTER TABLE `+table+` DROP CONSTRAINT `+table+`_reviewer_id_fkey`); e != nil {
			t.Fatal(e)
		}
		if _, e = conn.ExecContext(ctx, `ALTER TABLE `+table+` ADD CONSTRAINT `+table+`_reviewer_id_fkey FOREIGN KEY(reviewer_id) REFERENCES users(id)`); e != nil {
			t.Fatal(e)
		}
		if _, e = conn.ExecContext(ctx, `ALTER TABLE `+table+` DROP COLUMN reviewer_identity, DROP COLUMN reviewer_name`); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = conn.ExecContext(ctx, `ALTER TABLE ai_plan_reviews ALTER COLUMN reviewer_id SET NOT NULL`); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		if e = database.Migrate(ctx); e != nil {
			t.Fatal("legacy migration:", e)
		}
	}
	if e = database.DeleteUser(ctx, user); e != nil {
		t.Fatal("employee deletion after AI reviews:", e)
	}
	p, e = database.AIPlanDetail(ctx, workspace, d.ID)
	if e != nil || len(p.Recommendations[0].History) != 4 || p.Recommendations[0].Review.ReviewerName != "Fixture Manager" || p.Recommendations[0].History[0].ReviewerID != user || p.Recommendations[0].History[1].ReviewerName != "Fixture Admin" {
		t.Fatal("review history lost after employee deletion:", e)
	}
}
