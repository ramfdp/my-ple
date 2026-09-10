package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"io"
	"my-ple-backend/controller"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"my-ple-backend/notification"
	"my-ple-backend/realtime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRevisionIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set (isolated test database required)")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Hostname() != "127.0.0.1" || u.Path != "/my_ple_revision_test" {
		t.Fatal("test requires local my_ple_revision_test database")
	}
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	database := db.NewDatabase(conn)
	ctx := context.Background()
	database.Now = func() time.Time { return time.Date(2026, 9, 10, 3, 0, 0, 0, time.UTC) }
	if err = database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx); err != nil {
		t.Fatal("repeat migration", err)
	}
	t.Setenv("EMAIL_ENABLED", "true")
	t.Setenv("SMTP_HOST", "smtp.gmail.com")
	t.Setenv("SMTP_FROM", "aiple5766@gmail.com")
	t.Setenv("SMTP_USERNAME", "aiple5766@gmail.com")
	t.Setenv("SMTP_PASSWORD", "test-transport-never-used")
	type actor struct{ id, token string }
	createUser := func(role, address string) actor {
		t.Helper()
		id := db.NewID()
		if address == "" {
			address = id + "@example.test"
		}
		_, e := conn.Exec(`INSERT INTO users(id,name,email,role_id,username,password_hash) SELECT $1,$2,$3,id,$5,'unused-test-hash' FROM roles WHERE name=$4`, id, "Test "+role, address, role, id)
		if e != nil {
			t.Fatal(e)
		}
		token, hash, e := middleware.NewSessionToken()
		if e != nil {
			t.Fatal(e)
		}
		if e = database.CreateSession(ctx, id, hash, time.Now().Add(time.Hour)); e != nil {
			t.Fatal(e)
		}
		return actor{id, token}
	}
	manager := createUser("Admin", "")
	pe := createUser("Project Engineer", "")
	qa := createUser("Project Engineer", "")
	member := createUser("Project Engineer", "")
	outsider := createUser("IT", "")
	invalid := createUser("Project Engineer", "invalid-"+db.NewID())
	var division, job, qaJob int
	var jobName, category string
	if err = conn.QueryRow(`SELECT id,division_id,name,category_name FROM job_context WHERE role_name='PE' ORDER BY id LIMIT 1`).Scan(&job, &division, &jobName, &category); err != nil {
		t.Fatal(err)
	}
	if err = conn.QueryRow(`SELECT id FROM job_context WHERE role_name='QA' ORDER BY id LIMIT 1`).Scan(&qaJob); err != nil {
		t.Fatal(err)
	}
	if _, err = conn.Exec(`INSERT INTO job_pics(job_id,user_id) VALUES($1,$2)`, qaJob, qa.id); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(newHandler(database, "http://test.local"))
	defer server.Close()
	request := func(a actor, method, path string, body any) (int, map[string]json.RawMessage) {
		data, _ := json.Marshal(body)
		r, _ := http.NewRequest(method, server.URL+path, bytes.NewReader(data))
		r.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: a.token})
		r.Header.Set("Content-Type", "application/json")
		response, e := http.DefaultClient.Do(r)
		if e != nil {
			t.Error(e)
			return 0, nil
		}
		defer response.Body.Close()
		raw, _ := io.ReadAll(response.Body)
		var result map[string]json.RawMessage
		_ = json.Unmarshal(raw, &result)
		return response.StatusCode, result
	}
	mustTask := func(a actor, method, path string, body any) model.WorkItem {
		t.Helper()
		status, result := request(a, method, path, body)
		if status != 200 && status != 201 {
			t.Fatalf("%s %s: %d %s", method, path, status, result["error"])
		}
		var task model.WorkItem
		if e := json.Unmarshal(result["task"], &task); e != nil {
			t.Fatal(e)
		}
		return task
	}
	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		if e := conn.QueryRow(query, args...).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}

	t.Run("existing login RBAC profile and booking regression", func(t *testing.T) {
		loginUser := createUser("Project Engineer", "")
		hash, e := bcrypt.GenerateFromPassword([]byte("isolated-test-password"), bcrypt.MinCost)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = conn.Exec(`UPDATE users SET password_hash=$2 WHERE id=$1`, loginUser.id, string(hash)); e != nil {
			t.Fatal(e)
		}
		status, _ := request(actor{}, "POST", "/auth/login", map[string]string{"username": loginUser.id, "password": "isolated-test-password"})
		if status != 200 {
			t.Fatal("login", status)
		}
		for _, path := range []string{"/auth/me", "/profile", "/dashboard", "/workspaces", "/notifications", "/notifications/unread-count", "/zoom-resources", fmt.Sprintf("/workspaces/%d/workload", division)} {
			if status, _ := request(member, "GET", path, nil); status != 200 {
				t.Fatal(path, status)
			}
		}
		for _, path := range []string{"/users", "/roles"} {
			if status, _ := request(manager, "GET", path, nil); status != 200 {
				t.Fatal("employee admin", path, status)
			}
			if status, _ := request(member, "GET", path, nil); status != 403 {
				t.Fatal("employee RBAC", path, status)
			}
		}
		for _, path := range []string{"/reports.pdf?primaryDivision=Project%20Engineer", "/reports.docx?primaryDivision=Project%20Engineer", "/workspaces"} {
			if status, _ := request(actor{}, "GET", path, nil); status != 401 {
				t.Fatal("unauthenticated access", path, status)
			}
		}
		start := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
		input := model.ZoomBookingInput{Title: "Isolated booking regression", Description: "Test", ZoomResourceID: 1, StartAt: start, EndAt: start.Add(time.Hour)}
		status, result := request(member, "POST", "/zoom-bookings", input)
		if status != 201 {
			t.Fatal("booking create", status, string(result["error"]))
		}
		var booking model.ZoomBooking
		if e = json.Unmarshal(result["booking"], &booking); e != nil {
			t.Fatal(e)
		}
		defer request(member, "DELETE", "/zoom-bookings/"+booking.ID, nil)
		input.Title = "Updated booking regression"
		if status, _ := request(member, "PATCH", "/zoom-bookings/"+booking.ID, input); status != 200 {
			t.Fatal("booking edit", status)
		}
		if status, _ := request(outsider, "PATCH", "/zoom-bookings/"+booking.ID, input); status != 404 {
			t.Fatal("booking ownership", status)
		}
		q := url.Values{"start": {start.Format(time.RFC3339)}, "end": {start.Add(24 * time.Hour).Format(time.RFC3339)}}
		if status, _ := request(member, "GET", "/zoom-bookings?"+q.Encode(), nil); status != 200 {
			t.Fatal("booking list", status)
		}
	})

	t.Run("weekly concurrent partial updates and deletion", func(t *testing.T) {
		input := model.WeeklyReportInput{JobID: &job, PrimaryDivision: "Project Engineer", WorkDivision: category, Role: "PE", Work: jobName, Description: "original"}
		a, e := database.CreateReport(ctx, pe.id, input)
		if e != nil {
			t.Fatal(e)
		}
		b, e := database.CreateReport(ctx, pe.id, input)
		if e != nil {
			t.Fatal(e)
		}
		c, e := database.CreateReport(ctx, pe.id, input)
		if e != nil {
			t.Fatal(e)
		}
		status, _ := request(pe, "PATCH", "/reports/"+b.ID, map[string]any{"version": 1, "description": "PE saved"})
		if status != 200 {
			t.Fatal(status)
		}
		status, result := request(qa, "PUT", "/reports/"+b.ID, map[string]any{"version": 1, "description": "stale QA"})
		if status != 409 {
			t.Fatal("stale update", status)
		}
		var latest model.WeeklyReport
		_ = json.Unmarshal(result["report"], &latest)
		if latest.Description != "PE saved" || latest.Version != 2 {
			t.Fatalf("latest %+v", latest)
		}
		var wg sync.WaitGroup
		statuses := make(chan int, 2)
		for _, who := range []actor{pe, qa} {
			wg.Add(1)
			go func(who actor) {
				defer wg.Done()
				s, _ := request(who, "PATCH", "/reports/"+b.ID, map[string]any{"version": 2, "description": who.id})
				statuses <- s
			}(who)
		}
		wg.Wait()
		close(statuses)
		winners, conflicts := 0, 0
		for s := range statuses {
			if s == 200 {
				winners++
			}
			if s == 409 {
				conflicts++
			}
		}
		if winners != 1 || conflicts != 1 {
			t.Fatalf("winners %d conflicts %d", winners, conflicts)
		}
		for _, id := range []string{a.ID, b.ID} {
			if e = database.DeleteReport(ctx, id); e != nil {
				t.Fatal(e)
			}
		}
		start, next := model.ReportingWeek(database.Now())
		reports, e := database.ListReports(ctx, "Project Engineer", start, next)
		if e != nil {
			t.Fatal(e)
		}
		found := false
		for _, r := range reports {
			if r.ID == c.ID {
				found = true
				if r.DisplayNumber < 1 || r.Number != c.Number {
					t.Fatal("identity/number changed")
				}
			}
		}
		if !found {
			t.Fatal("surviving report missing")
		}
	})

	t.Run("assignment outbox priority and QA", func(t *testing.T) {
		input := map[string]any{"title": "Revision test task", "requiresQa": true, "assigneeIds": []string{pe.id}, "requestId": db.NewID(), "dueDate": "2026-09-20"}
		task := mustTask(manager, "POST", fmt.Sprintf("/workspaces/%d/tasks", division), input)
		path := "/work-items/" + task.ID
		again := mustTask(manager, "POST", fmt.Sprintf("/workspaces/%d/tasks", division), input)
		if again.ID != task.ID {
			t.Fatal("duplicate create")
		}
		assignmentCount := func() int {
			return count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE work_item_id=$1 AND notification_type='work_item_assignment'`, task.ID)
		}
		if assignmentCount() != 1 {
			t.Fatal("single assignment")
		}
		// A non-editor division member can assign, without gaining other edit rights.
		for _, body := range []map[string]any{
			{"version": task.Version, "assigneeIds": []string{outsider.id}},
			{"version": task.Version, "assigneeIds": []string{qa.id}, "title": "smuggled"},
		} {
			s, _ := request(member, "PATCH", path+"/assignees", body)
			if s != 400 && s != 403 {
				t.Fatal("invalid assignment accepted", s)
			}
		}
		task = mustTask(member, "PATCH", path+"/assignees", map[string]any{"version": task.Version, "assigneeIds": []string{pe.id, qa.id, invalid.id}})
		if assignmentCount() != 3 {
			t.Fatal("multiple new recipients")
		}
		task = mustTask(member, "PATCH", path+"/assignees", map[string]any{"version": task.Version, "assigneeIds": []string{pe.id, qa.id, invalid.id}})
		if assignmentCount() != 3 {
			t.Fatal("unchanged assignees duplicated assignment")
		}
		if count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE work_item_id=$1 AND recipient_user_id=$2 AND status='skipped' AND last_error='invalid_or_empty_email'`, task.ID, invalid.id) != 1 {
			t.Fatal("invalid recipient not tracked")
		}
		task = mustTask(manager, "PATCH", path, map[string]any{"version": task.Version, "title": task.Title})
		if assignmentCount() != 3 {
			t.Fatal("save duplicated assignment")
		}
		task = mustTask(member, "PATCH", path+"/priority", map[string]any{"version": task.Version, "priority": "HIGH"})
		if assignmentCount() != 3 {
			t.Fatal("priority duplicated assignment")
		}
		for _, body := range []map[string]any{{"version": task.Version, "priority": "LOW", "title": "smuggled"}, {"version": task.Version, "priority": "BOGUS"}} {
			s, _ := request(member, "PATCH", path+"/priority", body)
			if s != 400 {
				t.Fatal("payload accepted", s)
			}
		}
		s, _ := request(outsider, "PATCH", path+"/priority", map[string]any{"version": task.Version, "priority": "LOW"})
		if s != 403 {
			t.Fatal("scope leak", s)
		}
		s, _ = request(member, "DELETE", path, map[string]any{"version": task.Version})
		if s != 403 {
			t.Fatal("priority granted deletion", s)
		}
		task = mustTask(member, "PATCH", path+"/assignees", map[string]any{"version": task.Version, "assigneeIds": []string{pe.id, invalid.id}})
		task = mustTask(member, "PATCH", path+"/assignees", map[string]any{"version": task.Version, "assigneeIds": []string{pe.id, qa.id, invalid.id}})
		if assignmentCount() != 4 {
			t.Fatal("reassignment missing")
		}
		if count(`SELECT count(*) FROM notifications WHERE entity_id=$1 AND type='workspace_task'`, task.ID) == 0 {
			t.Fatal("existing activity notification lost")
		}
		task = mustTask(pe, "PATCH", path, map[string]any{"version": task.Version, "status": "IN_PROGRESS"})
		s, _ = request(pe, "PATCH", path, map[string]any{"version": task.Version, "status": "DONE"})
		if s != 400 {
			t.Fatal("QA bypass", s)
		}
		task = mustTask(pe, "PATCH", path+"/qa", map[string]any{"version": task.Version, "qaStage": "WAITING"})
		s, _ = request(pe, "PATCH", path+"/qa", map[string]any{"version": task.Version, "qaStage": "CHECKING"})
		if s != 403 {
			t.Fatal("PE checking", s)
		}
		task = mustTask(qa, "PATCH", path+"/qa", map[string]any{"version": task.Version, "qaStage": "CHECKING"})
		task = mustTask(qa, "PATCH", path+"/qa", map[string]any{"version": task.Version, "qaStage": "COMPLETE"})
		if task.Status != "IN_PROGRESS" || task.QAStage != "COMPLETE" {
			t.Fatal("QA incomplete")
		}
		var validator string
		var validatedAt time.Time
		if e := conn.QueryRow(`SELECT qa_validated_by,qa_validated_at FROM work_items WHERE id=$1`, task.ID).Scan(&validator, &validatedAt); e != nil || validator != qa.id || validatedAt.IsZero() {
			t.Fatal("QA validation audit missing", e)
		}
		task = mustTask(pe, "PATCH", path, map[string]any{"version": task.Version, "status": "DONE"})
		if task.Status != "DONE" {
			t.Fatal("validated task cannot finish")
		}
		if count(`SELECT count(*) FROM work_items WHERE id=$1`, task.ID) != 1 {
			t.Fatal("duplicate QA task")
		}
		sent := map[string]int{}
		first := true
		worker := notification.ZoomEmailWorker{DB: database, SendAssignment: func(_ context.Context, to, id string, p model.WorkAssignmentEmail) error {
			if p.TaskID != "" && p.TaskID != task.ID {
				return errors.New("unexpected task")
			}
			if first {
				first = false
				return errors.New("controlled transport failure")
			}
			sent[id]++
			return nil
		}}
		// Only this test's rows are eligible; historical test-run rows are not sent.
		if _, e := conn.Exec(`UPDATE zoom_booking_email_deliveries SET next_attempt_at=NOW()+interval '1 day' WHERE work_item_id IS DISTINCT FROM $1`, task.ID); e != nil {
			t.Fatal(e)
		}
		if e := worker.Run(ctx); e != nil {
			t.Fatal(e)
		}
		if _, e := conn.Exec(`UPDATE zoom_booking_email_deliveries SET next_attempt_at=NOW() WHERE work_item_id=$1 AND status='failed'`, task.ID); e != nil {
			t.Fatal(e)
		}
		if e := worker.Run(ctx); e != nil {
			t.Fatal(e)
		}
		if e := worker.Run(ctx); e != nil {
			t.Fatal(e)
		}
		for id, n := range sent {
			if n != 1 {
				t.Fatalf("duplicate transport success %s", id)
			}
		}
		if count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE work_item_id=$1 AND notification_type='work_item_assignment' AND status='sent'`, task.ID) != 3 {
			t.Fatal("assignment transport results")
		}
		if assignmentCount() != 4 {
			t.Fatal("retry re-enqueued assignment")
		}
	})

	t.Run("every active task requires audited QA validation", func(t *testing.T) {
		task := mustTask(member, "POST", fmt.Sprintf("/workspaces/%d/tasks", division), map[string]any{"title": "QA required for completion", "assigneeIds": []string{pe.id}, "requiresQa": false})
		path := "/work-items/" + task.ID
		task = mustTask(member, "PATCH", path, map[string]any{"version": task.Version, "status": "IN_PROGRESS"})
		status, body := request(member, "PATCH", path, map[string]any{"version": task.Version, "status": "DONE"})
		if status != 400 || len(body["error"]) == 0 {
			t.Fatal("unchecked task reached DONE", status)
		}
		status, _ = request(member, "PATCH", path+"/qa", map[string]any{"version": task.Version, "qaStage": "COMPLETE"})
		if status != 403 {
			t.Fatal("non-QA validation", status)
		}
		status, _ = request(qa, "PATCH", path+"/qa", map[string]any{"version": task.Version, "qaStage": "COMPLETE", "qaValidatedBy": member.id})
		if status != 400 {
			t.Fatal("client supplied validator accepted", status)
		}
		task = mustTask(qa, "PATCH", path+"/qa", map[string]any{"version": task.Version, "qaStage": "COMPLETE"})
		if task.Status != "IN_PROGRESS" {
			t.Fatal("validation silently completed work")
		}
		status, response := request(member, "GET", path, nil)
		var audit struct {
			At   *time.Time `json:"qaValidatedAt"`
			By   *string    `json:"qaValidatedBy"`
			Name string     `json:"qaValidatedByName"`
		}
		if err := json.Unmarshal(response["task"], &audit); err != nil || status != 200 || audit.At == nil || audit.By == nil || *audit.By != qa.id || audit.Name == "" {
			t.Fatal("validation audit missing from detail", status, err)
		}
		task = mustTask(member, "PATCH", path, map[string]any{"version": task.Version, "status": "DONE"})
		if task.Status != "DONE" {
			t.Fatal("validated completion failed")
		}
	})

	t.Run("long report HTTP persistence and PDF DOCX export", func(t *testing.T) {
		longText := strings.Repeat("Long weekly paragraph with intentional line break.\n", 120) + strings.Repeat("LongUnbrokenWord", 130) + "\nEND-" + db.NewID()
		input := model.WeeklyReportInput{JobID: &job, PrimaryDivision: "Project Engineer", WorkDivision: category, Role: "PE", Work: jobName, Description: longText}
		status, response := request(pe, "POST", "/reports", input)
		if status != 201 {
			t.Fatal("long create", status, string(response["error"]))
		}
		var report model.WeeklyReport
		if err := json.Unmarshal(response["report"], &report); err != nil || report.Description != longText {
			t.Fatal("create truncation", err)
		}
		longText += "\nFinal updated paragraph."
		status, response = request(qa, "PATCH", "/reports/"+report.ID, map[string]any{"version": report.Version, "description": longText})
		if status != 200 {
			t.Fatal("long update", status, string(response["error"]))
		}
		var stored string
		if err := conn.QueryRow(`SELECT description FROM reports WHERE id=$1`, report.ID).Scan(&stored); err != nil || stored != longText {
			t.Fatal("database truncation", err)
		}
		for _, format := range []string{"pdf", "docx"} {
			r, _ := http.NewRequest("GET", server.URL+"/reports."+format+"?primaryDivision=Project%20Engineer", nil)
			r.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: pe.token})
			res, err := http.DefaultClient.Do(r)
			if err != nil {
				t.Fatal(err)
			}
			data, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if res.StatusCode != 200 {
				t.Fatalf("%s export %d: %s", format, res.StatusCode, data)
			}
			if format == "pdf" {
				if !bytes.HasPrefix(data, []byte("%PDF-")) || !bytes.Contains(data, []byte("%%EOF")) {
					t.Fatal("invalid PDF")
				}
				continue
			}
			archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Fatal("not a DOCX zip", err)
			}
			parts := map[string]bool{}
			var text strings.Builder
			for _, file := range archive.File {
				parts[file.Name] = true
				if file.Name != "word/document.xml" {
					continue
				}
				rc, err := file.Open()
				if err != nil {
					t.Fatal(err)
				}
				decoder := xml.NewDecoder(rc)
				for {
					token, err := decoder.Token()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal("malformed document XML", err)
					}
					if chars, ok := token.(xml.CharData); ok {
						text.Write(chars)
					}
				}
				rc.Close()
			}
			for _, part := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml"} {
				if !parts[part] {
					t.Fatal("DOCX missing", part)
				}
			}
			// Word can encode intentional line breaks as XML elements, not literal newlines.
			if !strings.Contains(strings.ReplaceAll(text.String(), "\n", ""), strings.ReplaceAll(longText, "\n", "")) {
				t.Fatal("DOCX lost long report content")
			}
		}
	})

	t.Run("legacy cached PDF is regenerated without changing snapshot", func(t *testing.T) {
		start, end := model.ReportingWeek(database.Now())
		a := model.WeeklyArchive{ID: db.NewID(), Division: "Test " + db.NewID(), WeekStart: start, WeekEnd: end, Items: []model.ArchiveItem{{WeeklyReport: model.WeeklyReport{ID: db.NewID(), Work: "Bridge", Description: "Snapshot", Number: 9}}}}
		snapshot, _ := json.Marshal(a.Items)
		old := []byte("%PDF-1.4\nCONFIDENTIAL\n%%EOF")
		_, e := conn.Exec(`INSERT INTO weekly_report_archives(id,primary_division,week_start,week_end,snapshot,pdf_sha256) VALUES($1,$2,$3,$4,$5,$6)`, a.ID, a.Division, a.WeekStart, a.WeekEnd, string(snapshot), fmt.Sprintf("%x", sha256.Sum256(old)))
		if e != nil {
			t.Fatal(e)
		}
		lifecycle := controller.NewWeeklyLifecycle(database)
		lifecycle.StorageDir = t.TempDir()
		path := filepath.Join(lifecycle.StorageDir, a.Period().Start.Format("2006"), a.ID+".pdf")
		if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(path, old, 0600); e != nil {
			t.Fatal(e)
		}
		pdf, e := lifecycle.EnsurePDF(ctx, &a)
		if e != nil {
			t.Fatal(e)
		}
		if bytes.Equal(pdf, old) || bytes.Contains(pdf, []byte("CONFIDENTIAL")) {
			t.Fatal("legacy PDF retained")
		}
		var version int
		var number int
		if e = conn.QueryRow(`SELECT pdf_template_version,(snapshot->0->>'number')::int FROM weekly_report_archives WHERE id=$1`, a.ID).Scan(&version, &number); e != nil {
			t.Fatal(e)
		}
		if version != 2 || number != 9 {
			t.Fatal("template/snapshot", version, number)
		}
		again, e := lifecycle.EnsurePDF(ctx, &a)
		if e != nil || !bytes.Equal(pdf, again) {
			t.Fatal("new template cache not reused", e)
		}
	})
	t.Run("websocket recipient isolation", func(t *testing.T) {
		dial := func(a actor) *websocket.Conn {
			t.Helper()
			header := http.Header{"Origin": []string{"http://test.local"}, "Cookie": []string{middleware.SessionCookie + "=" + a.token}}
			c, _, e := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws/notifications", header)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { c.Close() })
			return c
		}
		inside, second, outside := dial(pe), dial(qa), dial(outsider)
		task := mustTask(manager, "POST", fmt.Sprintf("/workspaces/%d/tasks", division), map[string]any{"title": "socket task", "assigneeIds": []string{}})
		inside.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, data, e := inside.ReadMessage()
		if e != nil {
			t.Fatal(e)
		}
		if !bytes.Contains(data, []byte("workspace.changed")) {
			t.Fatal(string(data))
		}
		second.SetReadDeadline(time.Now().Add(time.Second))
		_, peer, e := second.ReadMessage()
		if e != nil || !bytes.Equal(data, peer) {
			t.Fatal("second session missed change", e)
		}
		outside.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		if _, _, e = outside.ReadMessage(); e == nil {
			t.Fatal("cross-division event")
		}
		awaitChange := func(socket *websocket.Conn, updated model.WorkItem) {
			t.Helper()
			socket.SetReadDeadline(time.Now().Add(3 * time.Second))
			for {
				var change notification.Change
				if e := socket.ReadJSON(&change); e != nil {
					t.Fatal("mutation realtime event missing", e)
				}
				if change.Event == "workspace.changed" && change.EntityID == updated.ID && change.Revision == int64(updated.Version) {
					return
				}
			}
		}
		path := "/work-items/" + task.ID
		task = mustTask(member, "PATCH", path+"/assignees", map[string]any{"version": task.Version, "assigneeIds": []string{pe.id}})
		awaitChange(inside, task)
		awaitChange(second, task)
		task = mustTask(pe, "PATCH", path, map[string]any{"version": task.Version, "status": "IN_PROGRESS"})
		awaitChange(inside, task)
		awaitChange(second, task)
		task = mustTask(qa, "PATCH", path+"/qa", map[string]any{"version": task.Version, "qaStage": "COMPLETE"})
		awaitChange(inside, task)
		awaitChange(second, task)
		status, response := request(pe, "GET", path, nil)
		var fresh map[string]json.RawMessage
		_ = json.Unmarshal(response["task"], &fresh)
		if status != 200 || string(fresh["qaValidatedBy"]) != fmt.Sprintf("%q", qa.id) {
			t.Fatal("peer cannot read committed validation")
		}
		s, _ := request(pe, "POST", "/auth/logout", nil)
		if s != 204 && s != 200 {
			t.Fatal(s)
		}
		inside.SetReadDeadline(time.Now().Add(time.Second))
		if _, _, e = inside.ReadMessage(); e == nil {
			t.Fatal("socket remained after logout")
		}
	})
	t.Run("PostgreSQL relay reaches another backend hub", func(t *testing.T) {
		relayCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		hub := realtime.NewHub("http://test.local")
		service := notification.Service{DB: database, Hub: hub}
		remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hub.ServeHTTP(database, w, r) }))
		defer remote.Close()
		header := http.Header{"Origin": []string{"http://test.local"}, "Cookie": []string{middleware.SessionCookie + "=" + qa.token}}
		ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(remote.URL, "http"), header)
		if err != nil {
			t.Fatal(err)
		}
		defer ws.Close()
		go service.Relay(relayCtx, dsn)
		// Retry publication until LISTEN is ready; this does not create data or send email.
		go func() {
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-relayCtx.Done():
					return
				case <-ticker.C:
					_, _ = conn.ExecContext(relayCtx, `SELECT pg_notify('my_ple_changes',$1)`, fmt.Sprintf(`{"event":"workspace.changed","workspaceId":%d,"entityId":"relay-test","revision":7}`, division))
				}
			}
		}()
		ws.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, payload, err := ws.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(payload, []byte("relay-test")) {
			t.Fatal("wrong relay payload")
		}
	})
}
