package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
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
}
