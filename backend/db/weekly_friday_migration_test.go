package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFridayNumbers(t *testing.T) {
	j := sql.NullInt64{Int64: 1, Valid: true}
	got, err := fridayNumbers([]fridayReport{{jobID: j, number: 1}, {jobID: j, number: 3}, {jobID: j, number: 1}, {division: "A", category: "B", number: 2}, {division: "A", category: "C", number: 2}}, []fridayReport{{jobID: j, number: 1}, {division: "A", category: "B", number: 2}})
	if err != nil || !reflect.DeepEqual(got, []int{4, 3, 5, 3, 2}) {
		t.Fatalf("numbers=%v err=%v", got, err)
	}
	if _, err = fridayNumbers([]fridayReport{{jobID: j, number: 2147483647}}, []fridayReport{{jobID: j, number: 2147483647}}); err == nil {
		t.Fatal("missing overflow guard")
	}
}
func fridayTestTransaction(t *testing.T) *sql.Tx {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required for PostgreSQL integration test")
	}
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	tx, err := conn.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tx.Rollback() })
	_, err = tx.Exec(`CREATE TEMP TABLE reports(id UUID PRIMARY KEY,report_week DATE,primary_division TEXT,work_division TEXT,job_id INTEGER,number INTEGER,completed_at TIMESTAMPTZ,description TEXT);
 CREATE UNIQUE INDEX ON reports(job_id,report_week,number) WHERE job_id IS NOT NULL;
 CREATE UNIQUE INDEX ON reports(primary_division,work_division,report_week,number) WHERE job_id IS NULL;
 CREATE TEMP TABLE weekly_report_archives(primary_division TEXT,week_start TIMESTAMPTZ,snapshot JSONB,status TEXT);
 CREATE TEMP TABLE job_context(id INTEGER,division_name TEXT,category_name TEXT);
 CREATE TEMP TABLE report_sequences(primary_division TEXT,work_division TEXT,week_start DATE,job_id INTEGER,last_number INTEGER,CONSTRAINT report_sequences_job_scope UNIQUE NULLS NOT DISTINCT(primary_division,work_division,week_start,job_id));
 CREATE TEMP TABLE weekly_period_migrations(name TEXT PRIMARY KEY,applied_at TIMESTAMPTZ NOT NULL);
 CREATE TEMP TABLE weekly_period_migration_reports(migration_name TEXT,report_id UUID,original_week DATE,original_number INTEGER,migrated_week DATE,migrated_number INTEGER,PRIMARY KEY(migration_name,report_id));`)
	if err != nil {
		t.Fatal(err)
	}
	return tx
}
func TestFridayMigrationPostgres(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	t.Run("preservation collisions counters idempotency", func(t *testing.T) {
		tx := fridayTestTransaction(t)
		_, err := tx.Exec(`INSERT INTO job_context VALUES(1,'A','B');
 INSERT INTO reports VALUES
 ('00000000-0000-0000-0000-000000000001','2026-10-30','A','B',1,1,'2026-10-30 00:00+07','target'),
 ('00000000-0000-0000-0000-000000000002','2026-10-26','A','B',1,1,'2026-10-31 00:00+07','move'),
 ('00000000-0000-0000-0000-000000000003','2026-11-02','A','B',1,1,'2026-11-03 00:00+07','move too'),
 ('00000000-0000-0000-0000-000000000004','2026-10-26','C','B',NULL,1,'2026-10-31 00:00+07','pending'),
 ('00000000-0000-0000-0000-000000000005','2026-10-26','D','B',NULL,1,'2026-10-31 00:00+07','failed'),
 ('00000000-0000-0000-0000-000000000006','2026-10-26','E','B',NULL,1,'2026-10-31 00:00+07','snapshot ID'),
 ('00000000-0000-0000-0000-000000000007','2026-10-19','A','B',1,1,'2026-10-20 00:00+07','history');
 INSERT INTO weekly_report_archives VALUES('C','2026-10-26 00:00+07','[]','pending'),('D','2026-10-26 00:00+07','[]','failed'),('E','2026-10-19 00:00+07','[{"id":"00000000-0000-0000-0000-000000000006"}]','finalized');
 INSERT INTO report_sequences VALUES('A','B','2026-10-26',1,1),('A','B','2026-11-02',1,1);`)
		if err != nil {
			t.Fatal(err)
		}
		if err = migrateWeeklyFriday(ctx, tx, now); err != nil {
			t.Fatal(err)
		}
		for query, want := range map[string]int{
			`SELECT count(*) FROM reports WHERE report_week='2026-10-30'`:                           3,
			`SELECT count(*) FROM reports WHERE report_week='2026-10-26'`:                           3,
			`SELECT count(*) FROM reports WHERE report_week='2026-10-19'`:                           1,
			`SELECT last_number FROM report_sequences WHERE job_id=1 AND week_start='2026-10-30'`:   3,
			`SELECT count(*) FROM report_sequences WHERE week_start IN ('2026-10-26','2026-11-02')`: 0,
		} {
			var got int
			if err = tx.QueryRow(query).Scan(&got); err != nil || got != want {
				t.Fatalf("%s got%d want%d err%v", query, got, want, err)
			}
		}
		var intact bool
		if err = tx.QueryRow(`SELECT description='move' AND completed_at='2026-10-31 00:00+07' AND job_id=1 AND number=2 FROM reports WHERE id='00000000-0000-0000-0000-000000000002'`).Scan(&intact); err != nil || !intact {
			t.Fatalf("data changed: %v", err)
		}
		if err = migrateWeeklyFriday(ctx, tx, now.AddDate(0, 0, 30)); err != nil {
			t.Fatal(err)
		}
		var count int
		if err = tx.QueryRow(`SELECT count(*) FROM weekly_period_migration_reports`).Scan(&count); err != nil || count != 2 {
			t.Fatalf("ledger count %d: %v", count, err)
		}
	})
	for _, scenario := range []string{"late", "target archive"} {
		t.Run(scenario, func(t *testing.T) {
			tx := fridayTestTransaction(t)
			if _, err := tx.Exec(`INSERT INTO reports VALUES('00000000-0000-0000-0000-000000000001','2026-10-26','A','B',NULL,1,'2026-11-01 00:00+07','original')`); err != nil {
				t.Fatal(err)
			}
			at := now
			if scenario == "late" {
				at = time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
			} else {
				if _, err := tx.Exec(`INSERT INTO weekly_report_archives VALUES('A','2026-10-30 00:00+07','[]','pending')`); err != nil {
					t.Fatal(err)
				}
			}
			if err := migrateWeeklyFriday(ctx, tx, at); err == nil {
				t.Fatal("expected safe abort")
			}
			var unchanged bool
			if err := tx.QueryRow(`SELECT report_week='2026-10-26' AND number=1 FROM reports`).Scan(&unchanged); err != nil || !unchanged {
				t.Fatalf("preflight mutated row: %v", err)
			}
		})
	}
	t.Run("SQL Saturday assignment and exclusive key range", func(t *testing.T) {
		tx := fridayTestTransaction(t)
		start := strings.Index(weeklyArchiveSQL, "CREATE OR REPLACE FUNCTION assign_report_week()")
		end := strings.Index(weeklyArchiveSQL[start:], "DO $$ BEGIN") + start
		trigger := strings.ReplaceAll(weeklyArchiveSQL[start:end], "FUNCTION assign_report_week()", "FUNCTION pg_temp.assign_report_week()")
		if _, err := tx.Exec(trigger); err != nil {
			t.Fatal(err)
		}
		for i, v := range []struct{ at, want string }{{"2026-11-06 16:59:59+00", "2026-10-30"}, {"2026-11-06 17:00:00+00", "2026-11-06"}, {"2026-11-13 16:59:59+00", "2026-11-06"}} {
			var got string
			err := tx.QueryRow(`INSERT INTO reports(id,primary_division,work_division,number,completed_at) VALUES($1,'A','B',$2,$3) RETURNING report_week::text`, fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1), i+1, v.at).Scan(&got)
			if err != nil || got != v.want {
				t.Fatalf("%s got%s want%s: %v", v.at, got, v.want, err)
			}
		}
		var count int
		if err := tx.QueryRow(`SELECT count(*) FROM reports WHERE report_week>='2026-10-30' AND report_week<'2026-11-06'`).Scan(&count); err != nil || count != 1 {
			t.Fatalf("range count %d: %v", count, err)
		}
	})
}
