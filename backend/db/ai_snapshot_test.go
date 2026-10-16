package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestAISnapshotPostgres(t *testing.T) {
	dsn := os.Getenv("AI_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AI_TEST_DATABASE_URL absent")
	}
	u, err := url.Parse(dsn)
	if err != nil || !strings.Contains(u.Path, "ai_snapshot_test") || u.Hostname() != "127.0.0.1" {
		t.Fatal("test URL must target local ai_snapshot_test database")
	}
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	ctx := context.Background()
	for _, ddl := range []string{
		`CREATE TEMP TABLE divisions(id integer PRIMARY KEY)`,
		`CREATE TEMP TABLE jobs(id integer PRIMARY KEY,division_id integer,name text)`,
		`CREATE TEMP TABLE work_items(id uuid PRIMARY KEY,division_id integer,job_id integer,title text,description text,start_date date,due_date date,status text,priority text,revision_count integer,requires_qa boolean,qa_stage text,archived_at timestamptz,version integer)`,
		`INSERT INTO divisions VALUES(1),(2)`,
		`INSERT INTO jobs VALUES(10,1,'Project')`,
		`INSERT INTO work_items VALUES
		('550e8400-e29b-41d4-a716-446655440001',1,10,'Open','Evidence','2026-10-14','2026-10-17','IN_PROGRESS','HIGH',0,false,'',NULL,1),
		('550e8400-e29b-41d4-a716-446655440002',1,10,'Done','','2026-10-14',NULL,'DONE','LOW',0,false,'',NULL,1),
		('550e8400-e29b-41d4-a716-446655440003',1,10,'Archived','','2026-10-14',NULL,'TODO','LOW',0,false,'',now(),1),
		('550e8400-e29b-41d4-a716-446655440004',2,NULL,'Other','','2026-10-14',NULL,'TODO','LOW',0,false,'',NULL,1)`,
		`ALTER TABLE work_items ADD COLUMN related_url text NOT NULL DEFAULT ''`,
	} {
		if _, err = conn.ExecContext(ctx, ddl); err != nil {
			t.Fatal(err)
		}
	}
	db := NewDatabase(conn)
	s, err := db.AISnapshot(ctx, 1)
	if err != nil || len(s.Tasks) != 1 || s.Tasks[0].ID != "550e8400-e29b-41d4-a716-446655440001" || s.Tasks[0].JobName != "Project" || s.Tasks[0].WorkspaceID != 1 {
		t.Fatalf("snapshot=%+v err=%v", s, err)
	}
	if _, err = db.AISnapshot(ctx, 999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing workspace: %v", err)
	}
	for i := 0; i < 200; i++ {
		id := fmt.Sprintf("550e8400-e29b-41d4-a716-%012x", i+100)
		if _, err = conn.ExecContext(ctx, `INSERT INTO work_items(id,division_id,title,description,status,priority,revision_count,requires_qa,qa_stage,version) VALUES($1,1,'More','','TODO','LOW',0,false,'',1)`, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.AISnapshot(ctx, 1); !errors.Is(err, ErrAISnapshotTooLarge) {
		t.Fatalf("oversize: %v", err)
	}
}
