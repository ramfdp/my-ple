package notification_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"my-ple-backend/controller"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"my-ple-backend/notification"
	"my-ple-backend/realtime"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLeaveWorkflowPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for isolated PostgreSQL integration test")
	}
	ctx := context.Background()
	adminConn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer adminConn.Close()
	schema := "leave_test_" + strings.ReplaceAll(db.NewID(), "-", "")
	if _, err = adminConn.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer adminConn.ExecContext(ctx, `DROP SCHEMA `+schema+` CASCADE`)
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

	employee, other, admin, manager, inactive := db.NewID(), db.NewID(), db.NewID(), db.NewID(), db.NewID()
	for _, u := range []struct{ id, name, role string }{{employee, "Employee", "Project Engineer"}, {other, "Other", "PMO"}, {admin, "Admin", "Admin"}, {manager, "Manager", "Management"}, {inactive, "Inactive", "Management"}} {
		if _, err = conn.ExecContext(ctx, `INSERT INTO users(id,name,email,username,password_hash,role_id,is_active) VALUES($1,$2,$3,$2,'unused',(SELECT id FROM roles WHERE name=$4),$5)`, u.id, u.name, strings.ToLower(u.name)+"@example.com", u.role, u.id != inactive); err != nil {
			t.Fatal(err)
		}
	}
	hub := realtime.NewHub("http://leave-test")
	ctl := &controller.LeaveController{DB: database, Notifications: &notification.Service{DB: database, Hub: hub}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws/notifications" {
			hub.ServeHTTP(database, w, r)
		} else {
			middleware.RequireAuth(database, ctl).ServeHTTP(w, r)
		}
	}))
	defer server.Close()
	tokens := map[string]string{}
	for _, u := range []string{employee, other, admin, manager} {
		token, hash, e := middleware.NewSessionToken()
		if e != nil {
			t.Fatal(e)
		}
		tokens[u] = token
		if e = database.CreateSession(ctx, u, hash, time.Now().Add(time.Hour)); e != nil {
			t.Fatal(e)
		}
	}
	request := func(user, method, path string, body any, want int) model.LeaveRequest {
		t.Helper()
		data, _ := json.Marshal(body)
		req, _ := http.NewRequest(method, server.URL+path, strings.NewReader(string(data)))
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: tokens[user]})
		res, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		var result struct {
			Request model.LeaveRequest
			Error   string
		}
		_ = json.NewDecoder(res.Body).Decode(&result)
		if res.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d (%s)", method, path, res.StatusCode, want, result.Error)
		}
		return result.Request
	}
	sockets := map[string]*websocket.Conn{}
	for _, u := range []string{employee, other, admin, manager} {
		c, _, e := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws/notifications", http.Header{"Origin": []string{"http://leave-test"}, "Cookie": []string{middleware.SessionCookie + "=" + tokens[u]}})
		if e != nil {
			t.Fatal(e)
		}
		sockets[u] = c
		defer c.Close()
		defer hub.CloseUser(u)
	}
	input := model.LeaveInput{RequestID: db.NewID(), RequestType: "CUTI", StartDate: "2026-09-15", EndDate: "2026-09-17", Reason: "Keperluan keluarga", Notes: "Catatan"}
	cuti := request(employee, "POST", "/leave-requests", input, 200)
	if cuti.Status != "PENDING" || cuti.UserID != employee || cuti.Division != "Project Engineer" {
		t.Fatal(cuti)
	}
	repeat := request(employee, "POST", "/leave-requests", input, 200)
	if repeat.ID != cuti.ID {
		t.Fatal("duplicate request")
	}
	for _, u := range []string{admin, manager} {
		sockets[u].SetReadDeadline(time.Now().Add(time.Second))
		var msg map[string]string
		if e := sockets[u].ReadJSON(&msg); e != nil || msg["event"] != "leave.changed" {
			t.Fatal("missing private invalidation", msg, e)
		}
	}
	sockets[other].SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	if _, _, e := sockets[other].ReadMessage(); e == nil {
		t.Fatal("unrelated user received leave event")
	}
	count := func(query string, want int, args ...any) {
		t.Helper()
		var got int
		if e := conn.QueryRowContext(ctx, query, args...).Scan(&got); e != nil || got != want {
			t.Fatalf("count got %d want %d err %v", got, want, e)
		}
	}
	count(`SELECT count(*) FROM leave_requests`, 1)
	count(`SELECT count(*) FROM notifications WHERE entity_id=$1`, 1, cuti.ID)
	count(`SELECT count(*) FROM notification_recipients nr JOIN notifications n ON n.id=nr.notification_id WHERE n.entity_id=$1`, 2, cuti.ID)
	count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE leave_request_id=$1`, 2, cuti.ID)
	for _, u := range []string{admin, manager} {
		feed, e := database.ListNotifications(ctx, u)
		if e != nil || len(feed) != 1 || feed[0].Leave == nil {
			t.Fatal("approver feed", feed, e)
		}
	}
	for _, u := range []string{employee, other, inactive} {
		feed, e := database.ListNotifications(ctx, u)
		if e != nil || len(feed) != 0 {
			t.Fatal("private feed leaked", feed, e)
		}
	}
	request("", "GET", "/leave-requests", nil, 401)
	request(other, "GET", "/leave-requests/"+cuti.ID, nil, 404)
	request(other, "GET", "/leave-requests/approvals", nil, 403)
	request(other, "GET", "/leave-requests/approvals/"+cuti.ID, nil, 403)
	request(other, "POST", "/leave-requests/approvals/"+cuti.ID+"/decision", model.LeaveDecision{Status: "APPROVED"}, 403)
	request(employee, "POST", "/leave-requests", map[string]any{"userId": other}, 400)
	bad := input
	bad.RequestID = db.NewID()
	bad.EndDate = "2026-09-14"
	request(employee, "POST", "/leave-requests", bad, 400)
	bad = input
	bad.StartDate = "2026-02-30"
	request(employee, "POST", "/leave-requests", bad, 400)
	bad = input
	bad.RequestType = "OTHER"
	request(employee, "POST", "/leave-requests", bad, 400)
	bad = input
	bad.Reason = "  "
	request(employee, "POST", "/leave-requests", bad, 400)
	input.RequestID = db.NewID()
	input.RequestType = "IZIN"
	izin := request(employee, "POST", "/leave-requests", input, 200)
	if izin.Status != "PENDING" {
		t.Fatal(izin)
	}
	approved := request(admin, "POST", "/leave-requests/approvals/"+cuti.ID+"/decision", model.LeaveDecision{Status: "APPROVED", ReviewerNote: "Disetujui sesuai jadwal"}, 200)
	if approved.ReviewedBy == nil || *approved.ReviewedBy != admin || approved.ReviewedAt == nil || approved.Status != "APPROVED" {
		t.Fatal(approved)
	}
	request(manager, "POST", "/leave-requests/approvals/"+cuti.ID+"/decision", model.LeaveDecision{Status: "REJECTED"}, 409)
	rejected := request(manager, "POST", "/leave-requests/approvals/"+izin.ID+"/decision", model.LeaveDecision{Status: "REJECTED", ReviewerNote: "Agenda project"}, 200)
	if rejected.Status != "REJECTED" || rejected.ReviewedBy == nil || *rejected.ReviewedBy != manager || rejected.ReviewerNote != "Agenda project" {
		t.Fatal(rejected)
	}
	for _, id := range []string{cuti.ID, izin.ID} {
		count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE leave_request_id=$1 AND notification_type<>'leave_request_created' AND recipient_user_id=$2`, 1, id, employee)
		count(`SELECT count(*) FROM notification_recipients nr JOIN notifications n ON n.id=nr.notification_id WHERE n.entity_id=$1 AND n.type<>'leave_request_created' AND nr.user_id=$2`, 1, id, employee)
	}
	feed, e := database.ListNotifications(ctx, employee)
	if e != nil || len(feed) != 2 {
		t.Fatal("owner decisions", feed, e)
	}
	sockets[employee].SetReadDeadline(time.Now().Add(time.Second))
	var event map[string]string
	if e = sockets[employee].ReadJSON(&event); e != nil || event["event"] != "leave.changed" {
		t.Fatal("owner websocket", event, e)
	}
	input.RequestID = db.NewID()
	own := request(admin, "POST", "/leave-requests", input, 200)
	for _, decision := range []string{"APPROVED", "REJECTED"} {
		request(admin, "POST", "/leave-requests/approvals/"+own.ID+"/decision", model.LeaveDecision{Status: decision}, 403)
	}
	request(manager, "POST", "/leave-requests/approvals/"+own.ID+"/decision", model.LeaveDecision{Status: "APPROVED"}, 200)
	input.RequestID = db.NewID()
	race := request(employee, "POST", "/leave-requests", input, 200)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i, u := range []string{admin, manager} {
		wg.Add(1)
		go func(user string, i int) {
			defer wg.Done()
			status := []string{"APPROVED", "REJECTED"}[i]
			_, e := database.DecideLeave(ctx, user, race.ID, model.LeaveDecision{Status: status})
			results <- e
		}(u, i)
	}
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if errors.Is(e, db.ErrLeaveConflict) {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatal("race", success, conflicts)
	}
	// Inject delivery failure into the EXISTING worker: no real SMTP is contacted.
	sends := 0
	worker := notification.ZoomEmailWorker{DB: database, SendLeave: func(context.Context, string, string, model.LeaveNotification) error {
		sends++
		return errors.New("SMTP unavailable")
	}}
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if sends == 0 {
		t.Fatal("no email attempts")
	}
	final, e := database.LeaveRequest(ctx, employee, cuti.ID, false)
	if e != nil || final.Status != "APPROVED" {
		t.Fatal("SMTP rolled back decision", e)
	}
	count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE status='failed'`, sends)
	// Enqueue failure occurs after commit; durable recovery runs on the same worker.
	if _, err = conn.ExecContext(ctx, `CREATE FUNCTION reject_leave_mail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'enqueue failure'; END $$; CREATE TRIGGER reject_mail BEFORE INSERT ON zoom_booking_email_deliveries FOR EACH ROW EXECUTE FUNCTION reject_leave_mail()`); err != nil {
		t.Fatal(err)
	}
	input.RequestID = db.NewID()
	survives := request(employee, "POST", "/leave-requests", input, 200)
	request(admin, "POST", "/leave-requests/approvals/"+survives.ID+"/decision", model.LeaveDecision{Status: "APPROVED"}, 200)
	count(`SELECT count(*) FROM leave_requests WHERE id=$1 AND status='APPROVED'`, 1, survives.ID)
	if _, err = conn.ExecContext(ctx, `DROP TRIGGER reject_mail ON zoom_booking_email_deliveries`); err != nil {
		t.Fatal(err)
	}
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	count(`SELECT count(*) FROM notifications WHERE entity_id=$1`, 2, survives.ID)
	count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE leave_request_id=$1`, 3, survives.ID)
	if _, err = database.NotifyLeave(ctx, survives.ID); err != nil {
		t.Fatal(err)
	}
	count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE leave_request_id=$1`, 3, survives.ID)
	// Actual successful sends use stable per-message IDs, and do not repeat on a second poll.
	if _, err = conn.ExecContext(ctx, `UPDATE zoom_booking_email_deliveries SET next_attempt_at=NOW() WHERE status='failed'`); err != nil {
		t.Fatal(err)
	}
	delivered := map[string]bool{}
	worker.SendLeave = func(_ context.Context, to, id string, p model.LeaveNotification) error {
		if delivered[id] {
			t.Error("duplicate delivery")
		}
		delivered[id] = true
		if to == "other@example.com" || to == "inactive@example.com" {
			t.Error("invalid recipient")
		}
		return nil
	}
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	before := len(delivered)
	if before == 0 {
		t.Fatal("no successful deliveries")
	}
	if err = worker.Run(ctx); err != nil || len(delivered) != before {
		t.Fatal("duplicate poll", err)
	}
	mine, e := database.LeaveRequests(ctx, employee, false, "", "", 0)
	if e != nil || len(mine) != 4 {
		t.Fatal("employee history", len(mine), e)
	}
	for i, row := range mine {
		if row.UserID != employee {
			t.Fatal("ownership leak")
		}
		if i > 0 && row.CreatedAt.After(mine[i-1].CreatedAt) {
			t.Fatal("history ordering")
		}
	}
	approvedList, e := database.LeaveRequests(ctx, admin, true, "APPROVED", "", 0)
	if e != nil || len(approvedList) < 3 {
		t.Fatal("approval history", e)
	}
	for _, row := range approvedList {
		if row.Status != "APPROVED" {
			t.Fatal("status filter")
		}
	}
	// Recipient authorization is current, not just an old notification snapshot.
	if _, e = conn.ExecContext(ctx, `UPDATE users SET role_id=(SELECT id FROM roles WHERE name='PMO') WHERE id=$1`, admin); e != nil {
		t.Fatal(e)
	}
	feed, e = database.ListNotifications(ctx, admin)
	if e != nil {
		t.Fatal(e)
	}
	for _, n := range feed {
		if n.Type == "leave_request_created" {
			t.Fatal("demoted approver still sees private requests")
		}
	}
	request(admin, "POST", "/leave-requests/approvals/"+race.ID+"/decision", model.LeaveDecision{Status: "APPROVED"}, 403)

}
