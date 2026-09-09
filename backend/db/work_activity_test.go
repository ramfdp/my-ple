package db_test

import (
	"context"
	"database/sql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"my-ple-backend/db"
	"my-ple-backend/model"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWorkActivityRecentWindow(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for isolated PostgreSQL integration test")
	}
	ctx := context.Background()
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "activity_test_" + strings.ReplaceAll(db.NewID(), "-", "")
	if _, err = admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.ExecContext(ctx, `DROP SCHEMA `+schema+` CASCADE`)
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.RuntimeParams["search_path"] = schema + ",public"
	conn := stdlib.OpenDB(*cfg)
	defer conn.Close()
	database := db.NewDatabase(conn)
	for i := 0; i < 2; i++ {
		if err = database.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}

	now := time.Date(2026, 9, 9, 10, 0, 0, 0, time.UTC)
	database.Now = func() time.Time { return now }
	actor := db.NewID()
	if _, err := conn.ExecContext(ctx, `INSERT INTO users(id,name,email,username,password_hash,role_id,is_active) VALUES($1,'Activity Test','activity@example.com','activity-test','unused',(SELECT id FROM roles WHERE name='Management'),true)`, actor); err != nil {
		t.Fatal(err)
	}
	var division int
	if err := conn.QueryRowContext(ctx, `SELECT id FROM divisions WHERE name='Project Engineer'`).Scan(&division); err != nil {
		t.Fatal(err)
	}
	task, err := database.SaveWorkItem(ctx, actor, division, "", model.WorkItemInput{Title: "Activity test", Status: "TODO", Priority: "MEDIUM"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `UPDATE work_item_activity SET created_at=$1 WHERE work_item_id=$2`, now, task.ID); err != nil {
		t.Fatal(err)
	}
	for _, age := range []time.Duration{5 * 24 * time.Hour, 7 * 24 * time.Hour, 7*24*time.Hour + time.Second} {
		if _, err := conn.ExecContext(ctx, `INSERT INTO work_item_activity(id,work_item_id,actor_user_id,action,created_at) VALUES($1,$2,$3,'UPDATE',$4)`, db.NewID(), task.ID, actor, now.Add(-age)); err != nil {
			t.Fatal(err)
		}
	}
	recent, err := database.WorkActivity(ctx, actor, division, "")
	if err != nil || len(recent) != 3 {
		t.Fatalf("today, five days and exact boundary: %v %v", recent, err)
	}
	for i, a := range recent {
		if a.CreatedAt.Before(now.Add(-7 * 24 * time.Hour)) {
			t.Fatal("expired activity returned")
		}
		if i > 0 && a.CreatedAt.After(recent[i-1].CreatedAt) {
			t.Fatal("not newest first")
		}
	}
	history, err := database.WorkActivity(ctx, actor, division, task.ID)
	if err != nil || len(history) != 4 {
		t.Fatalf("history lost old record: %v %v", history, err)
	}
	now = now.Add(8 * 24 * time.Hour)
	recent, err = database.WorkActivity(ctx, actor, division, "")
	if err != nil || len(recent) != 0 {
		t.Fatalf("expected empty recent activity: %v %v", recent, err)
	}
	history, err = database.WorkActivity(ctx, actor, division, task.ID)
	if err != nil || len(history) != 4 {
		t.Fatalf("history must survive expiration: %v %v", history, err)
	}
	if err := database.AddWorkComment(ctx, actor, task.ID, "Komentar uji"); err != nil {
		t.Fatal(err)
	}
	comments, err := database.WorkComments(ctx, actor, task.ID)
	if err != nil || len(comments) != 1 || comments[0].Content != "Komentar uji" {
		t.Fatalf("comment submission: %v %v", comments, err)
	}
	history, err = database.WorkActivity(ctx, actor, division, task.ID)
	if err != nil || len(history) != 5 {
		t.Fatalf("comment audit: %v %v", history, err)
	}
	found := false
	for _, a := range history {
		found = found || a.Action == "COMMENT"
	}
	if !found {
		t.Fatal("comment audit event missing")
	}

}
