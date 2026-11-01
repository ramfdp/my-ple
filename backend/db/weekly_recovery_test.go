package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
)

func recoveryTestDatabase(t *testing.T) *Database {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required for PostgreSQL recovery integration test")
	}
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	schema := "recovery_" + strings.ReplaceAll(NewID(), "-", "")
	if _, err = conn.Exec(`CREATE SCHEMA ` + schema + `; SET search_path TO ` + schema); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Exec(`DROP SCHEMA ` + schema + ` CASCADE`); conn.Close() })
	_, err = conn.Exec(`CREATE TABLE reports(id UUID PRIMARY KEY,report_week DATE,primary_division TEXT,work_division TEXT,job_id INTEGER,number INTEGER,completed_at TIMESTAMPTZ,description TEXT,user_id UUID);
 CREATE UNIQUE INDEX ON reports(job_id,report_week,number) WHERE job_id IS NOT NULL;
 CREATE UNIQUE INDEX ON reports(primary_division,work_division,report_week,number) WHERE job_id IS NULL;
 CREATE TABLE weekly_report_archives(primary_division TEXT,week_start TIMESTAMPTZ,snapshot JSONB,status TEXT,pdf_sha256 TEXT DEFAULT '',reset_at TIMESTAMPTZ,week_end TIMESTAMPTZ DEFAULT '2026-10-30 00:00+07');
 CREATE TABLE job_context(id INTEGER,division_name TEXT,category_name TEXT);
 CREATE TABLE report_sequences(primary_division TEXT,work_division TEXT,week_start DATE,job_id INTEGER,last_number INTEGER,CONSTRAINT report_sequences_job_scope UNIQUE NULLS NOT DISTINCT(primary_division,work_division,week_start,job_id));
 INSERT INTO job_context VALUES(1,'A','B');
 INSERT INTO reports VALUES
 ('00000000-0000-0000-0000-000000000001','2026-10-30','A','B',1,3,'2026-10-31 00:00+07','original','00000000-0000-0000-0000-000000000099'),
 ('00000000-0000-0000-0000-000000000002','2026-10-30','C','D',NULL,2,'2026-10-31 01:00+07','other division','00000000-0000-0000-0000-000000000099');
 INSERT INTO report_sequences VALUES('A','B','2026-10-30',1,3),('C','D','2026-10-30',NULL,2);`)
	if err != nil {
		t.Fatal(err)
	}
	return NewDatabase(conn)
}
func TestWeeklyRecoveryPostgres(t *testing.T) {
	ctx := context.Background()
	t.Run("read-only preview and immutable one-time cohort", func(t *testing.T) {
		db := recoveryTestDatabase(t)
		p, err := db.PreviewWeeklyRecovery(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if p.Count != 2 || len(p.Divisions) != 2 || len(p.Fingerprint) != 64 || p.Applied {
			t.Fatalf("preview %#v", p)
		}
		var exists bool
		if err = db.Conn.QueryRow(`SELECT to_regclass('weekly_report_recovery_runs') IS NOT NULL`).Scan(&exists); err != nil || exists {
			t.Fatalf("preview created schema: %v", err)
		}
		cohorts, err := db.ApplyWeeklyRecovery(ctx, p.Count, p.Fingerprint)
		if err != nil {
			t.Fatal(err)
		}
		if len(cohorts) != 2 || cohorts[0].Start.Format("2006-01-02") != "2026-10-23" {
			t.Fatalf("cohorts %#v", cohorts)
		}
		var intact bool
		err = db.Conn.QueryRow(`SELECT report_week='2026-10-23' AND number=3 AND description='original' AND completed_at='2026-10-31 00:00+07' AND job_id=1 AND user_id='00000000-0000-0000-0000-000000000099' FROM reports WHERE id='00000000-0000-0000-0000-000000000001'`).Scan(&intact)
		if err != nil || !intact {
			t.Fatalf("data changed: %v", err)
		}
		for query, want := range map[string]int{`SELECT count(*) FROM report_sequences WHERE week_start='2026-10-30'`: 0, `SELECT last_number FROM report_sequences WHERE job_id=1 AND week_start='2026-10-23'`: 3, `SELECT count(*) FROM weekly_report_recovery_rows`: 2} {
			var got int
			if err = db.Conn.QueryRow(query).Scan(&got); err != nil || got != want {
				t.Fatalf("%s: %d !=%d %v", query, got, want, err)
			}
		}
		// Simulate finalized/reset old reports and a newly written current report.
		_, err = db.Conn.Exec(`DELETE FROM reports; INSERT INTO weekly_report_archives(primary_division,week_start,snapshot,status) VALUES('A','2026-10-23 00:00+07','[{"id":"00000000-0000-0000-0000-000000000001"}]','finalized'),('C','2026-10-23 00:00+07','[{"id":"00000000-0000-0000-0000-000000000002"}]','finalized'); UPDATE weekly_report_archives SET pdf_sha256='verified-test-digest',reset_at=now(); INSERT INTO reports VALUES('00000000-0000-0000-0000-000000000003','2026-10-30','A','B',1,1,'2026-11-01 00:00+07','new',NULL)`)
		if err != nil {
			t.Fatal(err)
		}
		again, err := db.PreviewWeeklyRecovery(ctx)
		if err != nil || !again.Applied || again.Fingerprint != p.Fingerprint || again.Count != p.Count {
			t.Fatalf("retry preview %#v: %v", again, err)
		}
		cohorts, err = db.ApplyWeeklyRecovery(ctx, p.Count, p.Fingerprint)
		if err != nil || len(cohorts) != 2 {
			t.Fatalf("retry cohorts %v: %v", cohorts, err)
		}
		if err = db.VerifyWeeklyRecovery(ctx); err != nil {
			t.Fatal(err)
		}
		var current int
		if err = db.Conn.QueryRow(`SELECT count(*) FROM reports WHERE report_week='2026-10-30'`).Scan(&current); err != nil || current != 1 {
			t.Fatalf("retry swept new row: %v", err)
		}
	})
	t.Run("changed content invalidates confirmation", func(t *testing.T) {
		db := recoveryTestDatabase(t)
		p, err := db.PreviewWeeklyRecovery(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Conn.Exec(`UPDATE reports SET description='edited' WHERE primary_division='A'`); err != nil {
			t.Fatal(err)
		}
		if _, err = db.ApplyWeeklyRecovery(ctx, p.Count, p.Fingerprint); err == nil {
			t.Fatal("stale content preview accepted")
		}
		if _, err = db.ApplyWeeklyRecovery(ctx, p.Count-1, p.Fingerprint); err == nil {
			t.Fatal("wrong count accepted")
		}
		var exists bool
		if err = db.Conn.QueryRow(`SELECT to_regclass('weekly_report_recovery_runs') IS NOT NULL`).Scan(&exists); err != nil || exists {
			t.Fatalf("failed confirmation created schema: %v", err)
		}
	})
	for _, kind := range []string{"target rows", "source pending", "target failed", "snapshot mismatch"} {
		t.Run(kind, func(t *testing.T) {
			db := recoveryTestDatabase(t)
			p, err := db.PreviewWeeklyRecovery(ctx)
			if err != nil {
				t.Fatal(err)
			}
			statement := map[string]string{
				"target rows":       `INSERT INTO reports VALUES('00000000-0000-0000-0000-000000000003','2026-10-23','A','B',1,1,'2026-10-25 00:00+07','unrelated',NULL)`,
				"source pending":    `INSERT INTO weekly_report_archives(primary_division,week_start,snapshot,status) VALUES('A','2026-10-30 00:00+07','[]','pending')`,
				"target failed":     `INSERT INTO weekly_report_archives(primary_division,week_start,snapshot,status) VALUES('A','2026-10-23 00:00+07','[]','failed')`,
				"snapshot mismatch": `INSERT INTO weekly_report_archives(primary_division,week_start,snapshot,status) VALUES('X','2026-10-16 00:00+07','[{"id":"00000000-0000-0000-0000-000000000001"}]','finalized')`,
			}[kind]
			if _, err = db.Conn.Exec(statement); err != nil {
				t.Fatal(err)
			}
			if _, err = db.ApplyWeeklyRecovery(ctx, p.Count, p.Fingerprint); err == nil {
				t.Fatal("unsafe recovery accepted")
			}
			var count int
			if err = db.Conn.QueryRow(`SELECT count(*) FROM reports WHERE report_week='2026-10-30'`).Scan(&count); err != nil || count != 2 {
				t.Fatal(fmt.Sprintf("source changed count%d err%v", count, err))
			}
		})
	}
}
func TestWeeklyRecoveryRetryGuardsPostgres(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"target pollution", "snapshot mismatch", "completion gate"} {
		t.Run(kind, func(t *testing.T) {
			db := recoveryTestDatabase(t)
			p, err := db.PreviewWeeklyRecovery(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = db.ApplyWeeklyRecovery(ctx, p.Count, p.Fingerprint); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "target pollution":
				_, err = db.Conn.Exec(`INSERT INTO reports VALUES('00000000-0000-0000-0000-000000000003','2026-10-23','A','B',1,4,now(),'not confirmed',NULL)`)
			case "snapshot mismatch":
				_, err = db.Conn.Exec(`INSERT INTO weekly_report_archives(primary_division,week_start,snapshot,status) VALUES('A','2026-10-23 00:00+07','[{"id":"00000000-0000-0000-0000-000000000003"}]','pending')`)
			case "completion gate":
				if err = db.ValidateWeeklyRecovery(ctx); err != nil {
					t.Fatal(err)
				}
				if err = db.VerifyWeeklyRecovery(ctx); err == nil {
					t.Fatal("reported completion before archival/reset")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = db.ApplyWeeklyRecovery(ctx, p.Count, p.Fingerprint); err == nil {
				t.Fatal("unsafe retry accepted")
			}
			if err = db.ValidateWeeklyRecovery(ctx); err == nil {
				t.Fatal("unsafe snapshot/reset validation accepted")
			}
		})
	}
}
