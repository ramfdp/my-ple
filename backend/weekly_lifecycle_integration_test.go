package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"my-ple-backend/controller"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWeeklyLifecycleAutomaticRollover(t *testing.T) {
	// A disposable database keeps catch-up from processing other test fixtures.
	base := checklistTestDB(t)
	name := "weekly_test_" + strings.ReplaceAll(db.NewID(), "-", "")
	if _, err := base.Conn.Exec(`CREATE DATABASE ` + name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := base.Conn.Exec(`DROP DATABASE ` + name + ` WITH (FORCE)`); err != nil {
			t.Error(err)
		}
	})
	dsn, _ := url.Parse(os.Getenv("CHECKLIST_TEST_DATABASE_URL"))
	dsn.Path = "/" + name
	q := dsn.Query()
	q.Set("TimeZone", "UTC")
	dsn.RawQuery = q.Encode()
	conn, err := sql.Open("pgx", dsn.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	database := db.NewDatabase(conn)
	ctx := context.Background()
	if err = database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EMAIL_ENABLED", "false")
	t.Setenv("WEEKLY_REPORT_STORAGE_DIR", t.TempDir())
	location, _ := time.LoadLocation("Asia/Jakarta")
	boundary := time.Date(2026, 9, 15, 0, 0, 0, 0, location)
	database.Now = func() time.Time { return boundary }
	var job int
	if err = conn.QueryRow(`SELECT id FROM job_context WHERE role_name='PE' ORDER BY id LIMIT 1`).Scan(&job); err != nil {
		t.Fatal(err)
	}
	tokens := map[string]string{}
	users := map[string]string{}
	for _, role := range []string{"Project Engineer", "Admin", "Finance"} {
		id := db.NewID()
		users[role] = id
		if _, err = conn.Exec(`INSERT INTO users(id,name,email,role_id,username,password_hash) SELECT $1::uuid,$2::text,$3,id,$1::text,'unused' FROM roles WHERE name=$2::text`, id, role, id+"@example.test"); err != nil {
			t.Fatal(err)
		}
		token, hash, e := middleware.NewSessionToken()
		if e != nil {
			t.Fatal(e)
		}
		if e = database.CreateSession(ctx, id, hash, time.Now().Add(time.Hour)); e != nil {
			t.Fatal(e)
		}
		tokens[role] = token
	}
	insert := func(at time.Time, number int) string {
		t.Helper()
		id := db.NewID()
		_, e := conn.Exec(`INSERT INTO reports(id,user_id,primary_division,work_division,number,report_role,work,description,completed_at,job_id) SELECT $1::uuid,$2,'Project Engineer',category_name,$3,'PE',name,$1::text,$4,id FROM job_context WHERE id=$5`, id, users["Project Engineer"], number, at, job)
		if e != nil {
			t.Fatal(e)
		}
		return id
	}
	oldID := insert(boundary.Add(-time.Second), 1)
	currentID := insert(boundary, 2)
	weeks, err := database.DueWeeks(ctx, boundary.Add(-time.Second))
	if err != nil {
		t.Fatal("Monday week selection", err)
	}
	if len(weeks) != 0 {
		t.Fatal("rollover occurred before Tuesday", weeks)
	}
	weeks, err = database.DueWeeks(ctx, boundary)
	if err != nil {
		t.Fatal("Tuesday week selection", err)
	}
	if len(weeks) != 1 || !weeks[0].Start.Equal(boundary.AddDate(0, 0, -7)) {
		t.Fatal("wrong reporting boundary", weeks)
	}
	lifecycle := controller.NewWeeklyLifecycle(database)
	generate := lifecycle.GeneratePDF
	lifecycle.GeneratePDF = func(string, model.ReportingPeriod, []model.WeeklyReport) ([]byte, error) {
		return nil, errors.New("simulated PDF failure")
	}
	if err = lifecycle.Run(ctx, boundary); err != nil {
		t.Fatal(err)
	}
	history, err := database.ArchiveHistory(ctx, "Project Engineer", "", 0)
	if err != nil || len(history) != 1 || history[0].Status != "failed" || history[0].ResetAt != nil {
		t.Fatalf("failure must keep snapshot: %+v %v", history, err)
	}
	failedID := history[0].ID
	var count int
	if err = conn.QueryRow(`SELECT count(*) FROM reports`).Scan(&count); err != nil || count != 2 {
		t.Fatal("PDF failure deleted reports", count, err)
	}
	snapshot, err := database.Archive(ctx, failedID)
	if err != nil || len(snapshot.Items) != 1 || snapshot.Items[0].ID != oldID {
		t.Fatal("snapshot contents", snapshot, err)
	}
	// Catch up two more missed weeks without exporting anything.
	insert(boundary.AddDate(0, 0, -15), 1)
	insert(boundary.AddDate(0, 0, -22), 1)
	lifecycle.GeneratePDF = generate
	if err = lifecycle.Run(ctx, boundary); err != nil {
		t.Fatal(err)
	}
	history, err = database.ArchiveHistory(ctx, "Project Engineer", "", 0)
	if err != nil || len(history) != 3 {
		t.Fatal("catch-up history", len(history), err)
	}
	for _, a := range history {
		if a.Status != "finalized" || a.ResetAt == nil || a.PDFGeneratedAt == nil {
			t.Fatal("incomplete archive", a)
		}
	}
	if history[0].ID != failedID {
		t.Fatal("retry replaced snapshot")
	}
	if err = conn.QueryRow(`SELECT count(*) FROM reports`).Scan(&count); err != nil || count != 1 {
		t.Fatal("old reports were not reset", count, err)
	}
	var remaining string
	if err = conn.QueryRow(`SELECT id FROM reports`).Scan(&remaining); err != nil || remaining != currentID {
		t.Fatal("current report lost", remaining, err)
	}
	period, err := database.WorkingPeriod(ctx, "Project Engineer", boundary)
	if err != nil || period.ReadOnly || !period.Start.Equal(boundary) {
		t.Fatal("working period did not reset", period, err)
	}
	if err = lifecycle.Run(ctx, boundary.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	again, err := database.ArchiveHistory(ctx, "Project Engineer", "", 0)
	if err != nil || len(again) != 3 {
		t.Fatal("duplicate archives", len(again), err)
	}
	server := httptest.NewServer(newHandler(database, "http://test.local"))
	defer server.Close()
	get := func(role, path string, want int) []byte {
		t.Helper()
		request, _ := http.NewRequest("GET", server.URL+path, nil)
		request.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: tokens[role]})
		response, e := http.DefaultClient.Do(request)
		if e != nil {
			t.Fatal(e)
		}
		defer response.Body.Close()
		body, e := io.ReadAll(response.Body)
		if e != nil {
			t.Fatal(e)
		}
		if response.StatusCode != want {
			t.Fatalf("%s %s: %d %s", role, path, response.StatusCode, body)
		}
		return body
	}
	// History download is available even though no export endpoint was called.
	path := fmt.Sprintf("/reports/history/%s", failedID)
	for _, role := range []string{"Project Engineer", "Admin"} {
		get(role, "/reports/history", 200)
		var detail struct {
			Archive model.WeeklyArchive `json:"archive"`
			Groups  []struct {
				Categories []struct {
					Jobs []struct {
						Progress []struct {
							Number int
							Report model.WeeklyReport
						}
					}
				}
			} `json:"groups"`
		}
		if e := json.Unmarshal(get(role, path, 200), &detail); e != nil || len(detail.Archive.Items) != 1 || detail.Archive.Items[0].Description != oldID {
			t.Fatal("history detail lost content", e)
		}
		if len(detail.Groups) != 1 || len(detail.Groups[0].Categories) != 1 || len(detail.Groups[0].Categories[0].Jobs) != 1 || len(detail.Groups[0].Categories[0].Jobs[0].Progress) != 1 {
			t.Fatal("history grouping missing")
		}
		progress := detail.Groups[0].Categories[0].Jobs[0].Progress[0]
		if progress.Number != 1 || progress.Report.Description != oldID {
			t.Fatal("history grouping must use archived data and per-job numbering")
		}
		pdf := get(role, path+"/pdf", 200)
		if !bytes.HasPrefix(pdf, []byte("%PDF-")) || !bytes.Contains(pdf, []byte("%%EOF")) {
			t.Fatal("invalid history PDF")
		}
	}
	get("Finance", path, 403)
	get("Finance", path+"/pdf", 403)
}
