package controller

import (
	"bytes"
	"context"
	"fmt"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestJobPICPostgres(t *testing.T) {
	database := archiveTestDB(t)
	ctx := context.Background()
	role, _ := database.FindRoleByName(ctx, "Management")
	users := []model.User{}
	for i := 0; i < 3; i++ {
		user, err := database.CreateUser(ctx, model.User{ID: db.NewID(), Name: fmt.Sprintf("PIC %d", i), Username: fmt.Sprintf("pic%d", i), Email: fmt.Sprintf("pic%d@example.test", i), RoleID: role.ID, PasswordHash: "test", IsActive: i < 2})
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, user)
	}
	jobs, err := database.ListJobs(ctx, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	job := jobs[0]
	path := "/jobs/" + strconv.Itoa(job.ID)
	token, hash, _ := middleware.NewSessionToken()
	if err = database.CreateSession(ctx, users[0].ID, hash, time.Now().Add(time.Hour)); err != nil {
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
	roles, _ := database.ListRoles(ctx)
	for _, role := range roles {
		if _, err = database.Conn.Exec(`UPDATE users SET role_id=$2 WHERE id=$1`, users[0].ID, role.ID); err != nil {
			t.Fatal(err)
		}
		call("POST", path+"/pics", `{"userId":"`+users[1].ID+`"}`, 200)
		call("POST", path+"/pics", `{"userId":"`+users[1].ID+`"}`, 200)
		call("GET", path+"/pics", "", 200)
		call("DELETE", path+"/pics/"+users[1].ID, "", 200)
		call("DELETE", path+"/pics/"+users[1].ID, "", 404)
		call("POST", path+"/pics", `{"userId":"`+users[1].ID+`","name":"forged","isActive":false}`, 400)
	}
	call("POST", path+"/pics", `{"userId":"`+users[2].ID+`"}`, 400)
	call("POST", path+"/pics", `{"userId":"`+db.NewID()+`"}`, 404)
	call("POST", path+"/pics", `{"userId":"invalid"}`, 400)
	call("POST", "/jobs/999999/pics", `{"userId":"`+users[1].ID+`"}`, 404)
	result := call("GET", "/jobs/pic-users?q=pic1@example.test", "", 200)
	if !strings.Contains(result.Body.String(), users[1].ID) || strings.Contains(result.Body.String(), "salary") {
		t.Fatal("account search/privacy", result.Body.String())
	}
	result = call("GET", "/jobs/pic-users?q=pic2", "", 200)
	if strings.Contains(result.Body.String(), users[2].ID) {
		t.Fatal("inactive account selectable")
	}
	for _, method := range []string{"GET", "POST", "DELETE"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(method, path+"/pics", nil))
		if w.Code != 401 {
			t.Fatal("unauthenticated PIC access", w.Code)
		}
	}
	// Simultaneous requests for the same and different PICs must preserve both users once.
	var wg sync.WaitGroup
	errors := make(chan error, 20)
	start := make(chan struct{})
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errors <- database.AddJobPIC(ctx, job.ID, users[i%2].ID, users[0].ID)
		}(i)
	}
	close(start)
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	read, err := database.Job(ctx, job.ID)
	if err != nil || len(read.PICs) != 2 {
		t.Fatalf("concurrent PICs: %+v %v", read.PICs, err)
	}
	if _, err = database.Conn.Exec(`INSERT INTO job_pics(job_id,user_id) VALUES($1,$2)`, job.ID, users[0].ID); !db.UniqueViolation(err) {
		t.Fatal("uniqueness not enforced", err)
	}
	if err = database.AddJobPIC(ctx, jobs[1].ID, users[1].ID, users[0].ID); err != nil {
		t.Fatal(err)
	}
	if err = database.DeleteJob(ctx, jobs[1].ID); err != nil {
		t.Fatal("unused job/PIC deletion", err)
	}
	week, next := model.ReportingWeek(time.Now())
	database.Now = func() time.Time { return week.AddDate(0, 0, 2) }
	_, err = database.CreateReport(ctx, users[0].ID, model.WeeklyReportInput{JobID: &job.ID, PrimaryDivision: job.Division, Role: job.Role, WorkDivision: job.Category, Description: "PIC report"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = database.Conn.Exec(`UPDATE users SET is_active=false WHERE id=$1`, users[1].ID); err != nil {
		t.Fatal(err)
	}
	reports, err := database.ListReports(ctx, job.Division, week, next)
	if err != nil || len(reports) != 1 || len(reports[0].PICs) != 2 || reports[0].PICs[1].IsActive {
		t.Fatalf("live report PICs: %+v %v", reports, err)
	}
	archive, err := database.SnapshotWeek(ctx, db.DueWeek{Division: job.Division, Start: week})
	if err != nil {
		t.Fatal(err)
	}
	if err = database.RemoveJobPIC(ctx, job.ID, users[1].ID); err != nil {
		t.Fatal(err)
	}
	read, err = database.Job(ctx, job.ID)
	if err != nil || len(read.PICs) != 1 {
		t.Fatal("remove", err)
	}
	saved, err := database.Archive(ctx, archive.ID)
	if err != nil || len(saved.Items[0].PICs) != 2 {
		t.Fatal("archive PIC snapshot lost", err)
	}
	if err = database.DeleteJob(ctx, job.ID); err == nil {
		t.Fatal("PIC changes bypassed historical job protection")
	}
	// Hard account deletion removes only live assignments; immutable snapshots survive.
	if _, err = database.Conn.Exec(`UPDATE users SET is_active=true WHERE id=$1`, users[1].ID); err != nil {
		t.Fatal(err)
	}
	if err = database.AddJobPIC(ctx, job.ID, users[1].ID, users[0].ID); err != nil {
		t.Fatal(err)
	}
	if err = database.DeleteUser(ctx, users[1].ID); err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	read, err = database.Job(ctx, job.ID)
	if err != nil || len(read.PICs) != 1 {
		t.Fatal("delete/migration", err)
	}
}

func TestWeeklyPDFPICHeaders(t *testing.T) {
	week, next := model.ReportingWeek(time.Now())
	period := model.ReportingPeriod{Start: week, End: next}
	report := model.WeeklyReport{PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "Non-MK supervisi", Work: "MPS SGR", Number: 7, Description: "Report text", PICs: model.JobPICs{{Name: "Jarwo", IsActive: true}, {Name: "Budi", IsActive: true}}}
	data, err := weeklyPDFDocument(report.PrimaryDivision, period, []model.WeeklyReport{report, report}, false)
	if err != nil || bytes.Count(data, []byte("PIC: Jarwo, Budi")) != 1 || bytes.Count(data, []byte("(7)Tj")) != 2 {
		t.Fatal("PIC group header/numbering", err)
	}
	for i := 0; i < 500; i++ {
		report.PICs = append(report.PICs, model.JobPIC{Name: fmt.Sprintf("Responsible employee %03d", i), IsActive: true})
	}
	data, err = weeklyPDFDocument(report.PrimaryDivision, period, []model.WeeklyReport{report}, false)
	pages := bytes.Count(data, []byte("/Type /Page\n"))
	if err != nil || pages < 2 || bytes.Count(data, []byte("CONFIDENTIAL")) != pages || !bytes.Contains(data, []byte("499")) || !bytes.Contains(data, []byte("Report text")) {
		t.Fatal("long PIC pagination", pages, err)
	}
}
