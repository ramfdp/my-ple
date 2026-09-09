package notification_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"my-ple-backend/controller"
	"my-ple-backend/email"
	"my-ple-backend/middleware"
	"my-ple-backend/notification"
	"my-ple-backend/realtime"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"my-ple-backend/db"
	"my-ple-backend/model"
)

func TestWorkAssignmentPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for isolated PostgreSQL integration test")
	}
	ctx := context.Background()
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "assignment_test_" + strings.ReplaceAll(db.NewID(), "-", "")
	if _, err = admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.ExecContext(ctx, `DROP SCHEMA `+schema+` CASCADE`)
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.RuntimeParams["search_path"] = schema + ",public"
	conn := stdlib.OpenDB(*cfg)
	defer conn.Close()
	database := db.NewDatabase(conn)
	for i := 0; i < 2; i++ {
		if err = database.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	for k, v := range map[string]string{"ZOOM_EMAIL_ENABLED": "true", "SMTP_HOST": "smtp.gmail.com", "SMTP_FROM": "aiple5766@gmail.com", "SMTP_USERNAME": "aiple5766@gmail.com", "SMTP_PASSWORD": "test-placeholder", "SMTP_PORT": "587"} {
		t.Setenv(k, v)
	}
	actor, budi, andi, siti, outsider := db.NewID(), db.NewID(), db.NewID(), db.NewID(), db.NewID()
	for _, u := range []struct{ id, name, role string }{{actor, "Manager", "Management"}, {budi, "Budi", "Project Engineer"}, {andi, "Andi", "Project Engineer"}, {siti, "Siti", "Project Engineer"}, {outsider, "Other", "PMO"}} {
		if _, err = conn.ExecContext(ctx, `INSERT INTO users(id,name,email,username,password_hash,role_id,is_active) VALUES($1,$2,$3,$2,'unused',(SELECT id FROM roles WHERE name=$4),true)`, u.id, u.name, strings.ToLower(u.name)+"@example.com", u.role); err != nil {
			t.Fatal(err)
		}
	}
	var division, job int
	if err = conn.QueryRowContext(ctx, `SELECT id FROM divisions WHERE name='Project Engineer'`).Scan(&division); err != nil {
		t.Fatal(err)
	}
	if err = conn.QueryRowContext(ctx, `SELECT id FROM job_context WHERE name='MPS SGR' AND role_name='PE'`).Scan(&job); err != nil {
		t.Fatal(err)
	}
	in := model.WorkItemInput{RequestID: db.NewID(), Title: "Review Weekly Progress", Description: "Review <script>bad</script>", JobID: &job, Status: "TODO", Priority: "HIGH", StartDate: "2026-09-09", DueDate: "2026-09-12", AssigneeIDs: []string{budi, andi}}
	task, err := database.SaveWorkItem(ctx, actor, division, "", in)
	if err != nil {
		t.Fatal(err)
	}
	for _, recipient := range []string{budi, andi} {
		feed, err := database.ListNotifications(ctx, recipient)
		if err != nil || len(feed) != 1 || feed[0].Type != "work_item_assigned" {
			t.Fatalf("assignment feed: %#v %v", feed, err)
		}
	}

	count := func(query string, want int, args ...any) {
		t.Helper()
		var got int
		if err := conn.QueryRowContext(ctx, query, args...).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s: got %d want %d", query, got, want)
		}
	}
	assignmentCount := func(want int) { count(`SELECT count(*) FROM notifications WHERE type='work_item_assigned'`, want) }
	queueCount := func(want int) {
		count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE notification_type='work_item_assignment'`, want)
	}
	queueCount(2)
	if task.JobID == nil || *task.JobID != job {
		t.Fatal("wrong authoritative job ID")
	}
	again, err := database.SaveWorkItem(ctx, actor, division, "", in)
	if err != nil || again.ID != task.ID {
		t.Fatalf("create retry: %v %v", again, err)
	}
	assignmentCount(1)
	queueCount(2)
	in.RequestID = ""
	in.Version = task.Version
	in.Description = "description only"
	task, err = database.SaveWorkItem(ctx, actor, division, task.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	assignmentCount(1)
	queueCount(2)
	in.Version = task.Version
	in.AssigneeIDs = append(in.AssigneeIDs, siti)
	task, err = database.SaveWorkItem(ctx, actor, division, task.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	assignmentCount(2)
	queueCount(3)
	count(`SELECT count(*) FROM notification_recipients r JOIN notifications n ON n.id=r.notification_id WHERE n.type='work_item_assigned' AND r.user_id=$1`, 1, siti)
	for _, u := range []string{actor, outsider} {
		count(`SELECT count(*) FROM notification_recipients r JOIN notifications n ON n.id=r.notification_id WHERE n.type='work_item_assigned' AND r.user_id=$1`, 0, u)
	}
	if _, err = database.SaveWorkItem(ctx, actor, division, task.ID, in); !errors.Is(err, db.ErrWorkspaceConflict) {
		t.Fatal("stale edit accepted", err)
	}
	in.Version = task.Version
	in.AssigneeIDs = []string{budi, siti}
	task, err = database.SaveWorkItem(ctx, actor, division, task.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	assignmentCount(2)
	queueCount(3)
	in.Version = task.Version
	task, err = database.SaveWorkItem(ctx, actor, division, task.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	assignmentCount(2)
	queueCount(3)
	// SMTP failure is after commit; a removed assignee is skipped before sending.
	sends := map[string]int{}
	worker := notification.ZoomEmailWorker{DB: database, SendAssignment: func(_ context.Context, to, id string, a model.WorkAssignmentEmail) error {
		sends[to]++
		if a.Title != in.Title || a.ProjectName != "MPS SGR" || a.AssignedByName != "Manager" {
			t.Fatal(a)
		}
		return errors.New("simulated SMTP unavailable")
	}}
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if sends["budi@example.com"] != 1 || sends["siti@example.com"] != 1 || sends["andi@example.com"] != 0 {
		t.Fatal(sends)
	}
	saved, err := database.WorkItem(ctx, actor, task.ID)
	if err != nil || len(saved.Assignees) != 2 {
		t.Fatal("SMTP changed assignment", saved, err)
	}
	count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE status='failed'`, 2)
	count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE status='skipped'`, 1)
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if sends["budi@example.com"] != 1 {
		t.Fatal("retry ran before backoff")
	}
	if _, err = conn.ExecContext(ctx, `UPDATE zoom_booking_email_deliveries SET next_attempt_at=NOW()`); err != nil {
		t.Fatal(err)
	}
	worker.SendAssignment = func(_ context.Context, to, id string, a model.WorkAssignmentEmail) error { sends[to]++; return nil }
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE status='sent' AND attempt_count=2`, 2)
	if sends["budi@example.com"] != 2 {
		t.Fatal("completed delivery resent")
	}
	// Self-assignment is a real assignment, including visibility/read/dismiss rules.
	self := model.WorkItemInput{Title: "Self", Status: "TODO", Priority: "LOW", AssigneeIDs: []string{budi}}
	selfTask, err := database.SaveWorkItem(ctx, budi, division, "", self)
	if err != nil {
		t.Fatal(err)
	}
	feed, err := database.ListNotifications(ctx, budi)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range feed {
		if n.EntityID == selfTask.ID && n.Type == model.NotificationWorkItemAssigned {
			found = true
			if err = database.MarkNotificationRead(ctx, n.ID, budi); err != nil {
				t.Fatal(err)
			}
			if err = database.DismissNotification(ctx, n.ID, budi); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !found {
		t.Fatal("self assignment hidden")
	}
	// Three initial recipients independently queued, failed saves never notify.
	all := model.WorkItemInput{Title: "Three", Status: "TODO", Priority: "HIGH", AssigneeIDs: []string{budi, andi, siti}}
	if _, err = database.SaveWorkItem(ctx, actor, division, "", all); err != nil {
		t.Fatal(err)
	}
	assignmentCount(4)
	queueCount(7)
	all.AssigneeIDs = []string{outsider}
	if _, err = database.SaveWorkItem(ctx, actor, division, "", all); !errors.Is(err, db.ErrWorkspaceForbidden) {
		t.Fatal("cross division allowed", err)
	}
	all.AssigneeIDs = []string{budi}
	all.Title = ""
	if _, err = database.SaveWorkItem(ctx, actor, division, "", all); !errors.Is(err, db.ErrWorkspaceInvalid) {
		t.Fatal("invalid accepted", err)
	}
	assignmentCount(4)
	queueCount(7)
	// A broken outbox INSERT is isolated by the savepoint, leaving task + website event.
	if _, err = conn.ExecContext(ctx, `CREATE FUNCTION reject_assignment_email() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test enqueue failure'; END $$; CREATE TRIGGER reject_mail BEFORE INSERT ON zoom_booking_email_deliveries FOR EACH ROW EXECUTE FUNCTION reject_assignment_email()`); err != nil {
		t.Fatal(err)
	}
	all.Title = "Outbox failure"
	if _, err = database.SaveWorkItem(ctx, actor, division, "", all); err != nil {
		t.Fatal("enqueue rolled back task", err)
	}
	assignmentCount(5)
	queueCount(7)
	if _, err = conn.ExecContext(ctx, `DROP TRIGGER reject_mail ON zoom_booking_email_deliveries`); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZOOM_EMAIL_ENABLED", "false")
	all.Title = "Disabled email"
	if _, err = database.SaveWorkItem(ctx, actor, division, "", all); err != nil {
		t.Fatal(err)
	}
	assignmentCount(6)
	queueCount(7)
	t.Setenv("ZOOM_EMAIL_ENABLED", "true")
	// Exercise the real controller with cookie sessions and the existing WebSocket hub.
	hub := realtime.NewHub("http://workspace-test")
	svc := &notification.Service{DB: database, Hub: hub}
	ctl := &controller.WorkspaceController{DB: database, Notifications: svc}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws/notifications" {
			hub.ServeHTTP(database, w, r)
		} else {
			middleware.RequireAuth(database, ctl).ServeHTTP(w, r)
		}
	}))
	defer server.Close()
	tokens := map[string]string{}
	for _, u := range []string{actor, budi, outsider} {
		token, hash, e := middleware.NewSessionToken()
		if e != nil {
			t.Fatal(e)
		}
		tokens[u] = token
		if e = database.CreateSession(ctx, u, hash, time.Now().Add(time.Hour)); e != nil {
			t.Fatal(e)
		}
	}
	request := func(user, method, path string, body any) *http.Response {
		t.Helper()
		data, _ := json.Marshal(body)
		req, _ := http.NewRequest(method, server.URL+path, strings.NewReader(string(data)))
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: tokens[user]})
		res, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		return res
	}
	res := request(budi, "GET", fmt.Sprintf("/workspaces/%d/jobs", division), nil)
	var jobs struct {
		Jobs []struct {
			ID          int
			Name, Label string
		}
	}
	if err = json.NewDecoder(res.Body).Decode(&jobs); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal(res.Status)
	}
	ids, labels := map[int]bool{}, map[string]bool{}
	sgr := 0
	for _, j := range jobs.Jobs {
		if ids[j.ID] {
			t.Fatal("duplicate ID")
		}
		ids[j.ID] = true
		if j.Name == "MPS SGR" {
			sgr++
			if labels[j.Label] || j.Label == j.Name {
				t.Fatal("ambiguous contextual job", j)
			}
			labels[j.Label] = true
		}
	}
	if sgr != 3 {
		t.Fatal("distinct PE/QA1/QA2 records were merged", sgr)
	}
	res = request(outsider, "GET", fmt.Sprintf("/workspaces/%d/jobs", division), nil)
	res.Body.Close()
	if res.StatusCode != 403 {
		t.Fatal("job division isolation", res.Status)
	}
	sockets := map[string]*websocket.Conn{}
	for _, u := range []string{budi, outsider} {
		h := http.Header{"Origin": []string{"http://workspace-test"}, "Cookie": []string{middleware.SessionCookie + "=" + tokens[u]}}
		c, _, e := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws/notifications", h)
		if e != nil {
			t.Fatal(e)
		}
		sockets[u] = c
		defer c.Close()
		defer hub.CloseUser(u)
	}
	all.Title = "HTTP assignment"
	res = request(actor, "POST", fmt.Sprintf("/workspaces/%d/tasks", division), all)
	res.Body.Close()
	if res.StatusCode != 201 {
		t.Fatal(res.Status)
	}
	sockets[budi].SetReadDeadline(time.Now().Add(time.Second))
	var event struct {
		Event       string
		WorkspaceID int
	}
	if err = sockets[budi].ReadJSON(&event); err != nil || event.Event != "workspace.changed" || event.WorkspaceID != division {
		t.Fatal(event, err)
	}
	feed, err = database.ListNotifications(ctx, budi)
	if err != nil || feed[0].Workspace.Title != "HTTP assignment" {
		t.Fatal("socket refresh feed", feed, err)
	}
	sockets[outsider].SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	if _, _, err = sockets[outsider].ReadMessage(); err == nil {
		t.Fatal("workspace event leaked across divisions")
	}
	// Re-check membership at delivery and read time after a transfer.
	if _, err = conn.ExecContext(ctx, `UPDATE users SET role_id=(SELECT id FROM roles WHERE name='PMO') WHERE id=$1`, budi); err != nil {
		t.Fatal(err)
	}
	feed, err = database.ListNotifications(ctx, budi)
	if err != nil || len(feed) != 0 {
		t.Fatal("transferred user retained private feed")
	}
	worker.SendAssignment = func(_ context.Context, to, id string, a model.WorkAssignmentEmail) error {
		if to == "budi@example.com" {
			t.Fatal("email leaked after transfer")
		}
		return email.ErrUncertain
	}
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE status='uncertain'`, 2)
	t.Run("concurrent creation retries", func(t *testing.T) {
		key := db.NewID()
		type result struct {
			task model.WorkItem
			err  error
		}
		results := make(chan result, 2)
		start := make(chan struct{})
		for i := 0; i < 2; i++ {
			go func() {
				<-start
				input := model.WorkItemInput{RequestID: key, Title: "Concurrent", Status: "TODO", Priority: "LOW", AssigneeIDs: []string{andi}}
				task, err := database.SaveWorkItem(ctx, actor, division, "", input)
				results <- result{task, err}
			}()
		}
		close(start)
		a, b := <-results, <-results
		if a.err != nil || b.err != nil || a.task.ID != b.task.ID {
			t.Fatal("duplicate concurrent create", a, b)
		}
		count(`SELECT count(*) FROM notifications WHERE entity_id=$1 AND type='work_item_assigned'`, 1, a.task.ID)
		count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE work_item_id=$1`, 1, a.task.ID)
	})
	t.Run("relationship failure rolls back everything", func(t *testing.T) {
		if _, err = conn.ExecContext(ctx, `CREATE FUNCTION reject_assignment() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test relationship failure'; END $$; CREATE TRIGGER reject_assignment BEFORE INSERT ON work_item_assignees FOR EACH ROW EXECUTE FUNCTION reject_assignment()`); err != nil {
			t.Fatal(err)
		}
		key := db.NewID()
		input := model.WorkItemInput{RequestID: key, Title: "Must roll back", Status: "TODO", Priority: "LOW", AssigneeIDs: []string{andi}}
		if _, err = database.SaveWorkItem(ctx, actor, division, "", input); err == nil {
			t.Fatal("relationship failure accepted")
		}
		count(`SELECT count(*) FROM work_items WHERE creation_request_id=$1`, 0, key)
		count(`SELECT count(*) FROM notifications WHERE data->>'title'='Must roll back'`, 0)
		count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE payload->>'title'='Must roll back'`, 0)
	})

}
