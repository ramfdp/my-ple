package controller

import (
	"context"
	"encoding/json"
	"errors"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestJobManagementRBAC(t *testing.T) {
	for _, role := range []string{"Admin", "Management", "HR", "Project Engineer", "Finance", "IT", "Business Development", "PMO", "Future Role"} {
		request := httptest.NewRequest("POST", "/jobs", strings.NewReader(`invalid`))
		request.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: "test"})
		response := httptest.NewRecorder()
		middleware.RequireAuth(controllerSessionStore{role: role}, &JobController{}).ServeHTTP(response, request)
		if response.Code != 400 {
			t.Fatalf("authorized %s must reach input validation: %d", role, response.Code)
		}
	}
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		response := httptest.NewRecorder()
		(&JobController{}).ServeHTTP(response, httptest.NewRequest(method, "/jobs", nil))
		if response.Code != 401 {
			t.Fatalf("unauthenticated %s: %d", method, response.Code)
		}
	}

}

func TestJobMasterPostgres(t *testing.T) {
	database := archiveTestDB(t)
	ctx := context.Background()
	start, _ := model.ReportingWeek(time.Now())
	database.Now = func() time.Time { return start.AddDate(0, 0, 2) }
	role, _ := database.FindRoleByName(ctx, "Admin")
	user, err := database.CreateUser(ctx, model.User{ID: db.NewID(), Name: "Job test", Email: "jobs@example.test", Username: "jobs", RoleID: role.ID, PasswordHash: "test", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := database.ListJobs(ctx, nil, nil, nil, nil)
	if err != nil || len(jobs) != 61 {
		t.Fatalf("seed: %d %v", len(jobs), err)
	}
	var original model.Job
	for _, j := range jobs {
		if j.Role == "PE" && j.Category == "Non-MK supervisi" && j.Name == "MPS PIT" {
			original = j
		}
	}
	if original.ID == 0 {
		t.Fatal("missing PE seed")
	}
	custom := original
	custom.Name = "Dynamic job"
	custom.ID = 0
	custom, err = database.SaveJob(ctx, 0, custom, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := custom
	duplicate.Name = "  DYNAMIC JOB  "
	if _, err = database.SaveJob(ctx, 0, duplicate, user.ID); !db.UniqueViolation(err) {
		t.Fatalf("duplicate allowed: %v", err)
	}
	input := model.WeeklyReportInput{JobID: &custom.ID, PrimaryDivision: custom.Division, Role: custom.Role, WorkDivision: custom.Category, Description: "Created using master", Work: "forged text"}
	wrong := input
	wrong.PrimaryDivision = "HR"
	if _, err = database.CreateReport(ctx, user.ID, wrong); !errors.Is(err, db.ErrInvalidReport) {
		t.Fatalf("wrong division: %v", err)
	}
	wrong = input
	wrong.Role = "QA"
	if _, err = database.CreateReport(ctx, user.ID, wrong); !errors.Is(err, db.ErrInvalidReport) {
		t.Fatalf("wrong role: %v", err)
	}
	wrong = input
	wrong.WorkDivision = "MK supervisi"
	if _, err = database.CreateReport(ctx, user.ID, wrong); !errors.Is(err, db.ErrInvalidReport) {
		t.Fatalf("wrong category: %v", err)
	}
	report, err := database.CreateReport(ctx, user.ID, input)
	if err != nil || report.Work != custom.Name || report.JobID == nil {
		t.Fatalf("create: %+v %v", report, err)
	}
	if _, err = database.SetJobStatus(ctx, custom.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err = database.CreateReport(ctx, user.ID, input); !errors.Is(err, db.ErrInvalidReport) {
		t.Fatalf("inactive accepted: %v", err)
	}
	active := true
	options, err := database.ListJobs(ctx, &custom.DivisionID, custom.RoleID, custom.JobCategoryID, &active)
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range options {
		if j.ID == custom.ID {
			t.Fatal("inactive selectable")
		}
	}
	updated, err := database.UpdateReport(ctx, report.ID, model.WeeklyReportUpdateInput{JobID: &custom.ID, PrimaryDivision: input.PrimaryDivision, Role: input.Role, WorkDivision: input.WorkDivision, Description: "Still editable"})
	if err != nil || updated.Number != report.Number {
		t.Fatalf("inactive existing update: %v", err)
	}
	if err = database.DeleteJob(ctx, custom.ID); err == nil {
		t.Fatal("referenced job deleted")
	}
	moved := custom
	moved.RoleID = nil
	moved.JobCategoryID = nil
	if _, err = database.SaveJob(ctx, custom.ID, moved, user.ID); err == nil {
		t.Fatal("referenced context moved")
	}
	custom.Name = "Renamed job"
	custom.IsActive = false
	if _, err = database.SaveJob(ctx, custom.ID, custom, user.ID); err != nil {
		t.Fatal(err)
	}
	reports, err := database.ListReports(ctx, input.PrimaryDivision, start, start.AddDate(0, 0, 7))
	if err != nil || len(reports) != 1 || reports[0].Work != "Renamed job" {
		t.Fatalf("relational read: %+v %v", reports, err)
	}
	pdf, err := weeklyPDF(input.PrimaryDivision, model.ReportingPeriod{Start: start, End: start.AddDate(0, 0, 6)}, reports)
	if err != nil || len(pdf) < 500 {
		t.Fatalf("inactive PDF: %v", err)
	}
	archive, err := database.SnapshotWeek(ctx, db.DueWeek{Division: input.PrimaryDivision, Start: start})
	if err != nil {
		t.Fatal(err)
	}
	var refs int
	if err = database.Conn.QueryRow(`SELECT count(*) FROM weekly_archive_jobs WHERE archive_id=$1 AND job_id=$2`, archive.ID, custom.ID).Scan(&refs); err != nil || refs != 1 {
		t.Fatalf("archive references: %d %v", refs, err)
	}
	// Once used, even removal of a working report cannot permit deleting its master.
	if _, err = database.Conn.Exec(`DELETE FROM reports WHERE id=$1`, report.ID); err != nil {
		t.Fatal(err)
	}
	if err = database.DeleteJob(ctx, custom.ID); err == nil {
		t.Fatal("archived job deleted")
	}
	unused := original
	unused.Name = "Unused"
	unused, err = database.SaveJob(ctx, 0, unused, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = database.DeleteJob(ctx, unused.ID); err != nil {
		t.Fatal(err)
	}
	if err = database.DeleteJob(ctx, original.ID); err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = database.Job(ctx, original.ID); err == nil {
		t.Fatal("seed recreated deleted job")
	}
	if read, err := database.Archive(ctx, archive.ID); err != nil || read.Items[0].Work != "Renamed job" {
		t.Fatalf("snapshot changed: %v", err)
	}
}

func TestJobBackfillPreservesLegacyPostgres(t *testing.T) {
	database := archiveTestDB(t)
	ctx := context.Background()
	role, _ := database.FindRoleByName(ctx, "Admin")
	user, err := database.CreateUser(ctx, model.User{ID: db.NewID(), Name: "Legacy", Email: "legacy@example.test", Username: "legacy", RoleID: role.ID, PasswordHash: "test", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{db.NewID(), db.NewID(), db.NewID(), db.NewID()}
	for i, values := range [][4]string{{"Project Engineer", "QA 1", "QA 1", "  internal  "}, {"Project Engineer", "PE", "Non-MK supervisi", "MPS PIT"}, {"Project Engineer", "", "Non-MK supervisi", "MPS PIT"}, {"HR", "", "Pelaporan", "Unknown"}} {
		_, err = database.Conn.Exec(`INSERT INTO reports(id,user_id,primary_division,report_role,work_division,work,number,description,completed_at) VALUES($1,$2,$3,$4,$5,$6,$7,'legacy',NOW())`, ids[i], user.ID, values[0], values[1], values[2], values[3], i+1)
		if err != nil {
			t.Fatal(err)
		}
	}
	archiveID := db.NewID()
	snapshot := `[{"id":"legacy-archive","primaryDivision":"Project Engineer","role":"PE","workDivision":"Non-MK supervisi","work":"MPS PIT"},{"id":"unknown-archive","role":"PE","workDivision":"Non-MK supervisi","work":"unmatched"}]`
	if _, err = database.Conn.Exec(`INSERT INTO weekly_report_archives(id,primary_division,week_start,week_end,snapshot) VALUES($1,'Project Engineer','2020-01-06','2020-01-13',$2)`, archiveID, snapshot); err != nil {
		t.Fatal(err)
	}
	// Re-run only the one-time backfill against legacy fixtures in this isolated schema.
	if _, err = database.Conn.Exec(`DELETE FROM data_migrations WHERE name='job-backfill-v1'`); err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var matched, unmatched, refs int
	if err = database.Conn.QueryRow(`SELECT count(*) FILTER(WHERE job_id IS NOT NULL), count(*) FILTER(WHERE job_id IS NULL) FROM reports`).Scan(&matched, &unmatched); err != nil || matched != 2 || unmatched != 2 {
		t.Fatalf("backfill matched=%d unmatched=%d err=%v", matched, unmatched, err)
	}
	if err = database.Conn.QueryRow(`SELECT count(*) FROM unmatched_job_references`).Scan(&unmatched); err != nil || unmatched != 3 {
		t.Fatalf("unmatched report: %d %v", unmatched, err)
	}
	if err = database.Conn.QueryRow(`SELECT count(*) FROM weekly_archive_jobs WHERE archive_id=$1`, archiveID).Scan(&refs); err != nil || refs != 1 {
		t.Fatalf("archive backfill: %d %v", refs, err)
	}
	var intact bool
	if err = database.Conn.QueryRow(`SELECT snapshot=$2::jsonb FROM weekly_report_archives WHERE id=$1`, archiveID, snapshot).Scan(&intact); err != nil || !intact {
		t.Fatal("snapshot mutated", err)
	}
	var text string
	if err = database.Conn.QueryRow(`SELECT work FROM reports WHERE id=$1`, ids[0]).Scan(&text); err != nil || text != "  internal  " {
		t.Fatal("legacy text mutated", err)
	}
	if err = database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = database.Conn.QueryRow(`SELECT count(*) FROM jobs`).Scan(&count); err != nil || count != 61 {
		t.Fatal("seed is not idempotent", err, count)
	}
}

func TestJobAPIPostgres(t *testing.T) {
	database := archiveTestDB(t)
	ctx := context.Background()
	role, _ := database.FindRoleByName(ctx, "Management")
	user, err := database.CreateUser(ctx, model.User{ID: db.NewID(), Name: "API", Email: "api@example.test", Username: "api", RoleID: role.ID, PasswordHash: "test", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	token, hash, err := middleware.NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if err = database.CreateSession(ctx, user.ID, hash, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	handler := middleware.RequireAuth(database, &JobController{DB: database})
	call := func(method, path, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: token})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		return w
	}
	roles, err := database.ListRoles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range roles {
		t.Run(role.Name, func(t *testing.T) {
			if _, err = database.Conn.Exec(`UPDATE users SET role_id=$2 WHERE id=$1`, user.ID, role.ID); err != nil {
				t.Fatal(err)
			}
			call("GET", "/jobs/masters", "", 200)
			call("GET", "/jobs?division_id=invalid", "", 400)
			call("GET", "/jobs?active=invalid", "", 400)
			jobs, _ := database.ListJobs(ctx, nil, nil, nil, nil)
			job := jobs[0]
			job.Name = "API job"
			job.ID = 0
			body, _ := json.Marshal(job)
			response := call("POST", "/jobs", string(body), 201)
			var created struct {
				Job model.Job `json:"job"`
			}
			if err = json.Unmarshal(response.Body.Bytes(), &created); err != nil {
				t.Fatal(err)
			}
			path := "/jobs/" + strconv.Itoa(created.Job.ID)
			call("GET", path, "", 200)
			call("POST", "/jobs", string(body), 409)
			job.Name = "API edited"
			body, _ = json.Marshal(job)
			call("PUT", path, string(body), 200)
			call("PATCH", path+"/status", `{}`, 400)
			call("PATCH", path+"/status", `{"isActive":false}`, 200)
			response = call("GET", "/jobs?active=true", "", 200)
			if strings.Contains(response.Body.String(), "API edited") {
				t.Fatal("inactive in active API")
			}
			call("PATCH", path+"/status", `{"isActive":true}`, 200)
			call("DELETE", path, "", 204)
			call("GET", path, "", 404)
		})
	}
}
