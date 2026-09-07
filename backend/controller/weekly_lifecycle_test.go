package controller

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"my-ple-backend/db"
	"my-ple-backend/email"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// Every integration test owns a separate schema, never the application's tables.
func archiveTestDB(t *testing.T) *db.Database {
	t.Helper()
	value := os.Getenv("TEST_DATABASE_URL")
	if value == "" {
		t.Skip("TEST_DATABASE_URL required for isolated PostgreSQL integration")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	base, err := sql.Open("pgx", value)
	if err != nil {
		t.Fatal(err)
	}
	schema := "weekly_test_" + strings.ReplaceAll(db.NewID(), "-", "")
	if _, err = base.Exec(`CREATE SCHEMA ` + schema); err != nil {
		base.Close()
		t.Fatal(err)
	}
	q := parsed.Query()
	q.Set("search_path", schema+",public")
	q.Set("timezone", "America/New_York") // Deliberately different from reporting timezone.
	parsed.RawQuery = q.Encode()
	conn, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close(); base.Exec(`DROP SCHEMA ` + schema + ` CASCADE`); base.Close() })
	database := db.NewDatabase(conn)
	if err = database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(context.Background()); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	return database
}

func seedArchiveWeek(t *testing.T, database *db.Database) (time.Time, string, string) {
	t.Helper()
	ctx := context.Background()
	role, err := database.FindRoleByName(ctx, "Management")
	if err != nil {
		t.Fatal(err)
	}
	author := db.NewID()
	for i, name := range []string{"Original Employee", "Manager Two", "Inactive Manager", "Duplicate Manager", "Invalid Manager"} {
		id := db.NewID()
		if i == 0 {
			id = author
		}
		address := strings.ToLower(strings.ReplaceAll(name, " ", "")) + "@example.test"
		if i == 3 {
			address = "ORIGINALEMPLOYEE@example.test"
		}
		if i == 4 {
			address = "not-an-email"
		}
		_, err = database.CreateUser(ctx, model.User{ID: id, Name: name, Email: address, Username: id, RoleID: role.ID, PasswordHash: "test", IsActive: i != 2})
		if err != nil {
			t.Fatal(err)
		}
	}
	monday := time.Date(2020, 9, 7, 0, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	id := db.NewID()
	_, err = database.Conn.Exec(`INSERT INTO reports(id,user_id,primary_division,work_division,report_role,work,number,description,completed_at) VALUES($1,$2,'Project Engineer','MK supervisi','PE','Original Project',1,'Original Report',$3)`, id, author, monday.AddDate(0, 0, -2))
	if err != nil {
		t.Fatal(err)
	}
	return monday, id, author
}

func TestWeeklyLifecyclePostgres(t *testing.T) {
	database := archiveTestDB(t)
	ctx := context.Background()
	monday, reportID, author := seedArchiveWeek(t, database)
	database.Now = func() time.Time { return monday }
	s := NewWeeklyLifecycle(database)
	s.StorageDir = t.TempDir()
	sends := 0
	s.SendEmail = func(_ context.Context, to, subject, body, filename, id string, pdf []byte) error {
		sends++
		if !validPDF(pdf) || !strings.Contains(subject, "Project Engineer") || !strings.HasSuffix(filename, ".pdf") {
			t.Error("invalid attachment")
		}
		return nil
	}
	// Before Monday there is no completed week, hence no empty/premature archive.
	if err := s.Run(ctx, monday.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	var count int
	database.Conn.QueryRow(`SELECT count(*) FROM weekly_report_archives`).Scan(&count)
	if count != 0 {
		t.Fatal("premature archive")
	}
	if err := s.Run(ctx, monday); err != nil {
		t.Fatal(err)
	}
	history, err := database.ArchiveHistory(ctx, "Project Engineer", "2020-09-06", 0)
	if err != nil || len(history) != 1 {
		t.Fatalf("history: %v %v", history, err)
	}
	for date, want := range map[string]int{"2020-08-31": 1, "2020-09-06": 1, "2020-09-07": 0} {
		rows, e := database.ArchiveHistory(ctx, "Project Engineer", date, 0)
		if e != nil || len(rows) != want {
			t.Fatalf("date boundary %s: %d %v", date, len(rows), e)
		}
	}
	a, err := database.Archive(ctx, history[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != "finalized" || len(a.Items) != 1 || a.Items[0].EmployeeName != "Original Employee" || sends != 2 {
		t.Fatalf("archive=%+v sends=%d", a, sends)
	}
	database.Conn.QueryRow(`SELECT count(*) FROM reports WHERE id=$1`, reportID).Scan(&count)
	if count != 1 {
		t.Fatal("Monday removed source")
	}
	period, err := database.WorkingPeriod(ctx, "Project Engineer", monday.Add(12*time.Hour))
	if err != nil || !period.ReadOnly || !period.Start.Equal(monday.AddDate(0, 0, -7)) {
		t.Fatalf("grace: %+v %v", period, err)
	}
	if err = database.DeleteReport(ctx, reportID); !errors.Is(err, db.ErrReportClosed) {
		t.Fatalf("closed delete: %v", err)
	}
	input := model.WeeklyReportInput{PrimaryDivision: "Project Engineer", WorkDivision: "MK supervisi", Role: "PE", Work: "Original Project", Description: "New week"}
	if _, err = database.CreateReport(ctx, author, input); !errors.Is(err, db.ErrReportClosed) {
		t.Fatalf("Monday create allowed: %v", err)
	}
	if _, err = database.UpdateReport(ctx, reportID, model.WeeklyReportUpdateInput{}); !errors.Is(err, db.ErrReportClosed) {
		t.Fatalf("closed update: %v", err)
	}
	original, err := os.ReadFile(s.pdfPath(a))
	if err != nil {
		t.Fatal(err)
	}
	// Snapshot remains stable after changes to users and mutable working rows.
	database.Conn.Exec(`UPDATE users SET name='Changed',role_id=(SELECT id FROM roles WHERE name='HR') WHERE id=$1`, author)
	database.Conn.Exec(`UPDATE reports SET work='Changed Project',description='Changed Report',report_role='QA',work_division='QA1' WHERE id=$1`, reportID)
	stable, err := database.Archive(ctx, a.ID)
	if err != nil || stable.Items[0].EmployeeName != "Original Employee" || stable.Items[0].Work != "Original Project" {
		t.Fatal("snapshot changed")
	}
	visible, err := database.PeriodReports(ctx, "Project Engineer", period)
	if err != nil || len(visible) != 1 || visible[0].Work != "Original Project" {
		t.Fatal("Monday view differs from snapshot")
	}
	if _, err = database.Conn.Exec(`UPDATE weekly_report_archives SET snapshot='[]' WHERE id=$1`, a.ID); err == nil {
		t.Fatal("snapshot was mutable")
	}
	if err = s.Run(ctx, monday.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	// A restarted service recognizes already sent deliveries and the existing PDF.
	restarted := NewWeeklyLifecycle(database)
	restarted.StorageDir = s.StorageDir
	restarted.SendEmail = s.SendEmail
	if err = restarted.Run(ctx, monday.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(s.pdfPath(a))
	if !bytes.Equal(original, after) || sends != 2 {
		t.Fatal("restart regenerated or resent")
	}
	if err = s.Run(ctx, monday.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	database.Now = func() time.Time { return monday.Add(24 * time.Hour) }
	newReport, err := database.CreateReport(ctx, author, input)
	if err != nil || newReport.Number != 1 {
		t.Fatalf("new weekly sequence: %+v %v", newReport, err)
	}
	updated, err := database.UpdateReport(ctx, newReport.ID, model.WeeklyReportUpdateInput{PrimaryDivision: "Project Engineer", WorkDivision: "Non-MK supervisi", Role: "PE", Work: "MPS PIT", Description: "New week updated"})
	if err != nil || updated.Number != 1 {
		t.Fatalf("new week category sequence: %+v %v", updated, err)
	}
	if err = database.DeleteReport(ctx, newReport.ID); err != nil {
		t.Fatalf("new week delete: %v", err)
	}
	database.Conn.QueryRow(`SELECT count(*) FROM reports WHERE id=$1`, reportID).Scan(&count)
	if count != 0 {
		t.Fatal("Tuesday did not reset")
	}
	a, err = database.Archive(ctx, a.ID)
	if err != nil || a.ResetAt == nil {
		t.Fatal("reset tracking missing")
	}
	database.Conn.QueryRow(`SELECT count(*) FROM weekly_report_email_deliveries WHERE status='sent'`).Scan(&count)
	if count != 2 {
		t.Fatal("deliveries lost")
	}
	// Missing/corrupt PDF is regenerated from immutable snapshot, never current rows.
	if err = os.WriteFile(s.pdfPath(a), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	s.GeneratePDF = func(division string, p model.ReportingPeriod, reports []model.WeeklyReport) ([]byte, error) {
		if len(reports) != 1 || reports[0].Work != "Original Project" || reports[0].Description != "Original Report" {
			t.Fatal("recovery used mutable source")
		}
		return weeklyPDFDocument(division, p, reports, false)
	}
	restored, err := s.EnsurePDF(ctx, &a)
	if err != nil || !bytes.Contains(restored, []byte("Original Report")) {
		t.Fatalf("PDF recovery: %v", err)
	}
	t.Setenv("WEEKLY_REPORT_STORAGE_DIR", s.StorageDir)
	ctrl := NewWeeklyController(database)
	req := httptest.NewRequest("GET", "/reports/history/"+a.ID+"/pdf", nil)
	recorder := httptest.NewRecorder()
	ctrl.History(recorder, req)
	if recorder.Code != 401 {
		t.Fatal("public archive")
	}
	// Use the actual authentication middleware's session resolution for endpoints.
	token := "archive-test-token"
	if err = database.CreateSession(ctx, author, middleware.HashToken(token), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: token})
	recorder = httptest.NewRecorder()
	middleware.RequireAuth(database, http.HandlerFunc(ctrl.History)).ServeHTTP(recorder, req)
	if recorder.Code != 200 || recorder.Header().Get("Content-Type") != "application/pdf" || !bytes.Equal(recorder.Body.Bytes(), restored) {
		t.Fatalf("historical download: %d %s", recorder.Code, recorder.Body.String())
	}
	for path, status := range map[string]int{"/reports/history?date=bad": 400, "/reports/history?page=-1": 400, "/reports/history/no-such-id/pdf": 404, "/reports/history?division=HR": 200} {
		request := httptest.NewRequest("GET", path, nil)
		request.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: token})
		response := httptest.NewRecorder()
		middleware.RequireAuth(database, http.HandlerFunc(ctrl.History)).ServeHTTP(response, request)
		if response.Code != status {
			t.Fatalf("%s: %d want %d", path, response.Code, status)
		}
	}
	if err = database.DeleteUser(ctx, author); err != nil {
		t.Fatalf("delete after safe reset: %v", err)
	}
	archived, err := database.Archive(ctx, a.ID)
	if err != nil || archived.Items[0].EmployeeName != "Original Employee" {
		t.Fatal("deleting user changed history")
	}
}

func TestWeeklyFailuresAndConcurrentRestartPostgres(t *testing.T) {
	database := archiveTestDB(t)
	ctx := context.Background()
	monday, reportID, _ := seedArchiveWeek(t, database)
	s := NewWeeklyLifecycle(database)
	s.StorageDir = t.TempDir()
	s.GeneratePDF = func(string, model.ReportingPeriod, []model.WeeklyReport) ([]byte, error) {
		return nil, errors.New("test PDF failure")
	}
	s.SendEmail = func(context.Context, string, string, string, string, string, []byte) error {
		return errors.New("test SMTP failure")
	}
	if err := s.Run(ctx, monday.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	var count int
	database.Conn.QueryRow(`SELECT count(*) FROM reports WHERE id=$1`, reportID).Scan(&count)
	if count != 1 {
		t.Fatal("PDF failure lost source")
	}
	s.GeneratePDF = weeklyPDF
	// Retry at Monday for email-failure grace assertions.
	if err := s.Run(ctx, monday); err != nil {
		t.Fatal(err)
	}
	database.Conn.QueryRow(`SELECT count(*) FROM weekly_report_email_deliveries WHERE status='failed' AND attempt_count=1`).Scan(&count)
	if count != 2 {
		t.Fatal("email failure tracking")
	}
	database.Conn.QueryRow(`SELECT count(*) FROM reports WHERE id=$1`, reportID).Scan(&count)
	if count != 1 {
		t.Fatal("email failure lost Monday source")
	}
	var wg sync.WaitGroup
	s.SendEmail = func(context.Context, string, string, string, string, string, []byte) error { return nil }
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Run(ctx, monday.Add(time.Hour)); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	database.Conn.QueryRow(`SELECT count(*) FROM weekly_report_archives`).Scan(&count)
	if count != 1 {
		t.Fatal("duplicate archive")
	}
	database.Conn.QueryRow(`SELECT count(*) FROM weekly_report_email_deliveries WHERE status='sent' AND attempt_count=2`).Scan(&count)
	if count != 2 {
		t.Fatal("retry duplicated delivery")
	}
	// Ambiguous network acceptance must not be automatically retried after restart.
	database.Conn.Exec(`UPDATE weekly_report_email_deliveries SET status='failed',last_attempt_at=NULL`)
	s.SendEmail = func(context.Context, string, string, string, string, string, []byte) error { return email.ErrUncertain }
	if err := s.Run(ctx, monday.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.Run(ctx, monday.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	database.Conn.QueryRow(`SELECT count(*) FROM weekly_report_email_deliveries WHERE status='uncertain' AND attempt_count=3`).Scan(&count)
	if count != 2 {
		t.Fatal("uncertain send retried")
	}
}

func TestWeeklyDeliveryLimitsAndNoRecipientsPostgres(t *testing.T) {
	t.Run("bounded retries", func(t *testing.T) {
		database := archiveTestDB(t)
		monday, _, _ := seedArchiveWeek(t, database)
		s := NewWeeklyLifecycle(database)
		s.StorageDir = t.TempDir()
		ctx := context.Background()
		attempts := 0
		s.SendEmail = func(context.Context, string, string, string, string, string, []byte) error {
			attempts++
			return errors.New("temporary SMTP failure")
		}
		for i := 0; i < 7; i++ {
			if err := s.Run(ctx, monday.Add(time.Duration(i)*3*time.Hour)); err != nil {
				t.Fatal(err)
			}
		}
		if attempts != 10 {
			t.Fatalf("attempts=%d want two recipients times five", attempts)
		}
		var count int
		database.Conn.QueryRow(`SELECT count(*) FROM weekly_report_email_deliveries WHERE status='failed' AND attempt_count=5`).Scan(&count)
		if count != 2 {
			t.Fatal("retry budget not persisted")
		}
		database.Conn.Exec(`UPDATE weekly_report_email_deliveries SET status='sending'`)
		if err := s.Run(ctx, monday.Add(20*time.Hour)); err != nil {
			t.Fatal(err)
		}
		database.Conn.QueryRow(`SELECT count(*) FROM weekly_report_email_deliveries WHERE status='uncertain'`).Scan(&count)
		if count != 2 || attempts != 10 {
			t.Fatal("interrupted delivery resent")
		}
	})
	t.Run("no recipients and missed Tuesday", func(t *testing.T) {
		database := archiveTestDB(t)
		monday, id, _ := seedArchiveWeek(t, database)
		ctx := context.Background()
		database.Conn.Exec(`UPDATE users SET is_active=false`)
		s := NewWeeklyLifecycle(database)
		s.StorageDir = t.TempDir()
		s.SendEmail = func(context.Context, string, string, string, string, string, []byte) error {
			t.Fatal("ineligible recipient")
			return nil
		}
		if err := s.Run(ctx, monday.Add(48*time.Hour)); err != nil {
			t.Fatal(err)
		}
		var count int
		database.Conn.QueryRow(`SELECT count(*) FROM reports WHERE id=$1`, id).Scan(&count)
		if count != 0 {
			t.Fatal("missed reset not recovered")
		}
		database.Conn.QueryRow(`SELECT count(*) FROM weekly_report_archives WHERE status='finalized' AND reset_at IS NOT NULL`).Scan(&count)
		if count != 1 {
			t.Fatal("archive missing with no recipients")
		}
		database.Conn.QueryRow(`SELECT count(*) FROM weekly_report_email_deliveries`).Scan(&count)
		if count != 0 {
			t.Fatal("ineligible deliveries created")
		}
	})
}
