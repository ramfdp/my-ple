package controller

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"my-ple-backend/db"
	"my-ple-backend/model"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestWeeklyRecoveryPDFFailureRetryPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("weekly_recovery_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.RuntimeParams["search_path"] = schema
	conn := stdlib.OpenDB(*config)
	defer conn.Close()
	_, err = conn.Exec(`CREATE TABLE roles(id INTEGER PRIMARY KEY,name TEXT);
 CREATE TABLE users(id UUID PRIMARY KEY,name TEXT,role_id INTEGER,email TEXT,is_active BOOLEAN);
 CREATE TABLE jobs(id INTEGER PRIMARY KEY,name TEXT);
 CREATE TABLE job_pic_summary(job_id INTEGER,pics JSONB);
 CREATE TABLE job_context(id INTEGER,division_name TEXT,category_name TEXT);
 CREATE TABLE reports(id UUID PRIMARY KEY,user_id UUID,primary_division TEXT,work_division TEXT,report_role TEXT,work TEXT,number INTEGER,description TEXT,completed_at TIMESTAMPTZ,job_id INTEGER,report_week DATE);
 CREATE TABLE report_sequences(primary_division TEXT,work_division TEXT,week_start DATE,job_id INTEGER,last_number INTEGER,CONSTRAINT report_sequences_job_scope UNIQUE NULLS NOT DISTINCT(primary_division,work_division,week_start,job_id));
 CREATE TABLE weekly_report_archives(id UUID PRIMARY KEY,primary_division TEXT,week_start TIMESTAMPTZ,week_end TIMESTAMPTZ,snapshot JSONB,status TEXT DEFAULT 'pending',finalized_at TIMESTAMPTZ,pdf_filename TEXT DEFAULT '',pdf_sha256 TEXT DEFAULT '',pdf_generated_at TIMESTAMPTZ,reset_at TIMESTAMPTZ,created_at TIMESTAMPTZ DEFAULT NOW(),pdf_storage_key TEXT DEFAULT '',pdf_template_version INTEGER DEFAULT 0,last_error TEXT DEFAULT '',UNIQUE(primary_division,week_start,week_end));
 CREATE TABLE weekly_archive_jobs(archive_id UUID,report_id TEXT,job_id INTEGER);
 CREATE TABLE weekly_report_email_deliveries(id UUID PRIMARY KEY,archive_id UUID,recipient_user_id UUID,recipient_email TEXT,status TEXT DEFAULT 'pending',UNIQUE(archive_id,recipient_email));
 INSERT INTO roles VALUES(1,'Management');
 INSERT INTO users VALUES('00000000-0000-0000-0000-000000000001','Test Manager',1,'manager@example.com',TRUE);
 INSERT INTO reports VALUES('00000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000001','A','B','Project Engineer','Test Work',1,'Preserve me','2026-10-31 10:00+07',NULL,'2026-10-30');`)
	if err != nil {
		t.Fatal(err)
	}
	database := db.NewDatabase(conn)
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	database.Now = func() time.Time { return now }
	ctx := context.Background()
	preview, err := database.PreviewWeeklyRecovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	lifecycle := NewWeeklyLifecycle(database)
	lifecycle.StorageDir = t.TempDir()
	worker, err := conn.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = worker.ExecContext(ctx, `SELECT pg_advisory_lock(71260401)`); err != nil {
		worker.Close()
		t.Fatal(err)
	}
	_, blockedErr := lifecycle.RunRecovery(ctx, preview.Count, preview.Fingerprint, now)
	_, unlockErr := worker.ExecContext(ctx, `SELECT pg_advisory_unlock(71260401)`)
	worker.Close()
	if blockedErr == nil || unlockErr != nil {
		t.Fatalf("worker exclusion: %v %v", blockedErr, unlockErr)
	}
	var ledgerCreated bool
	if err = conn.QueryRow(`SELECT to_regclass('weekly_report_recovery_runs') IS NOT NULL`).Scan(&ledgerCreated); err != nil || ledgerCreated {
		t.Fatalf("blocked recovery changed schema: %v %v", ledgerCreated, err)
	}

	lifecycle.GeneratePDF = func(string, model.ReportingPeriod, []model.WeeklyReport) ([]byte, error) {
		return nil, errors.New("PDF unavailable")
	}
	lifecycle.SendEmail = func(context.Context, string, string, string, string, string, []byte) error {
		t.Fatal("recovery must not send mail")
		return nil
	}
	if _, err = lifecycle.RunRecovery(ctx, preview.Count, preview.Fingerprint, now); err == nil {
		t.Fatal("PDF failure ignored")
	}
	var count int
	if err = conn.QueryRow(`SELECT count(*) FROM reports WHERE id='00000000-0000-0000-0000-000000000010'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("PDF failure lost working row: %d %v", count, err)
	}
	if _, err = conn.Exec(`INSERT INTO reports VALUES('00000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000001','A','B','Project Engineer','New Work',1,'New current row','2026-11-01 10:00+07',NULL,'2026-10-30')`); err != nil {
		t.Fatal(err)
	}
	lifecycle.GeneratePDF = func(string, model.ReportingPeriod, []model.WeeklyReport) ([]byte, error) {
		return []byte("%PDF-1.4\n%%EOF"), nil
	}
	for attempt := 0; attempt < 2; attempt++ {
		results, err := lifecycle.RunRecovery(ctx, preview.Count, preview.Fingerprint, now)
		if err != nil || len(results) != 1 {
			t.Fatalf("retry %d: %+v %v", attempt, results, err)
		}
	}
	for query, want := range map[string]int{
		`SELECT count(*) FROM reports WHERE id='00000000-0000-0000-0000-000000000010'`:                              0,
		`SELECT count(*) FROM reports WHERE id='00000000-0000-0000-0000-000000000011' AND report_week='2026-10-30'`: 1,
		`SELECT count(*) FROM weekly_report_archives WHERE status='finalized' AND reset_at IS NOT NULL`:             1,
		`SELECT count(*) FROM weekly_report_email_deliveries WHERE status='pending'`:                                1,
	} {
		if err = conn.QueryRow(query).Scan(&count); err != nil || count != want {
			t.Fatalf("%s: got%d want%d err%v", query, count, want, err)
		}
	}
}
