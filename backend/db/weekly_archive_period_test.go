package db

import (
	"context"
	"database/sql"
	"my-ple-backend/model"
	"os"
	"testing"
	"time"
)

func TestFridayArchiveQueriesPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	_, err = conn.Exec(`CREATE TEMP TABLE reports(id UUID,primary_division TEXT,report_week DATE);
 CREATE TEMP TABLE report_sequences(primary_division TEXT,week_start DATE);
 CREATE TEMP TABLE weekly_report_archives(id UUID,primary_division TEXT,week_start TIMESTAMPTZ,week_end TIMESTAMPTZ,snapshot JSONB,status TEXT,finalized_at TIMESTAMPTZ,pdf_filename TEXT,pdf_sha256 TEXT,pdf_generated_at TIMESTAMPTZ,reset_at TIMESTAMPTZ,created_at TIMESTAMPTZ,pdf_storage_key TEXT);
 INSERT INTO weekly_report_archives VALUES
 ('00000000-0000-0000-0000-000000000001','Friday','2026-10-30 00:00+07','2026-11-06 00:00+07','[]','pending',NULL,'','',NULL,NULL,NOW(),''),
 ('00000000-0000-0000-0000-000000000002','Legacy','2026-10-26 00:00+07','2026-11-02 00:00+07','[]','pending',NULL,'','',NULL,NULL,NOW(),'');`)
	if err != nil {
		t.Fatal(err)
	}
	database := &Database{Conn: conn}
	ctx := context.Background()
	for _, c := range []struct {
		division, date string
		want           int
	}{{"Friday", "2026-11-06", 1}, {"Friday", "2026-11-07", 0}, {"Legacy", "2026-11-01", 1}, {"Legacy", "2026-11-02", 0}} {
		got, err := database.ArchiveHistory(ctx, c.division, c.date, 0)
		if err != nil || len(got) != c.want {
			t.Fatalf("history %+v got%v err%v", c, got, err)
		}
	}
	for _, c := range []struct {
		at   string
		want int
	}{{"2026-11-06T16:59:59Z", 1}, {"2026-11-06T17:00:00Z", 2}} {
		at, _ := time.Parse(time.RFC3339, c.at)
		got, err := database.DueWeeks(ctx, at)
		if err != nil || len(got) != c.want {
			t.Fatalf("due %+v got%v err%v", c, got, err)
		}
	}
	if _, err = conn.Exec(`UPDATE weekly_report_archives SET status='finalized',pdf_sha256='verified' WHERE primary_division='Friday'`); err != nil {
		t.Fatal(err)
	}
	archive, err := database.Archive(ctx, "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatal(err)
	}
	saturday, _ := time.Parse(time.RFC3339, "2026-11-06T17:00:00Z")
	if _, err = database.ResetArchivedWeek(ctx, archive, saturday); err != nil {
		t.Fatalf("Saturday reset denied: %v", err)
	}
}

func TestFridayArchiveCannotResetBeforeSaturday(t *testing.T) {
	start, _ := time.Parse(time.RFC3339, "2026-10-29T17:00:00Z")
	at, _ := time.Parse(time.RFC3339, "2026-11-06T16:59:59Z")
	a := model.WeeklyArchive{WeekStart: start, WeekEnd: start.AddDate(0, 0, 7), Status: "finalized", SHA256: "verified"}
	if _, err := (&Database{}).ResetArchivedWeek(context.Background(), a, at); err != ErrReportClosed {
		t.Fatalf("early reset: %v", err)
	}
}
