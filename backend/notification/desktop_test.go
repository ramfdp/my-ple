package notification

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"my-ple-backend/db"
)

func testSubscription(t *testing.T) webpush.Subscription {
	t.Helper()
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return webpush.Subscription{Endpoint: "https://fcm.googleapis.com/fcm/send/example", Keys: webpush.Keys{P256dh: base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()), Auth: base64.RawURLEncoding.EncodeToString(make([]byte, 16))}}
}

func TestDesktopSubscriptionValidation(t *testing.T) {
	sub := testSubscription(t)
	if !ValidDesktopSubscription(sub) {
		t.Fatal("valid subscription rejected")
	}
	for _, endpoint := range []string{"http://fcm.googleapis.com/x", "https://example.com/x", "https://fcm.googleapis.com:444/x", "https://fcm.googleapis.com.example.com/x", "https://a@fcm.googleapis.com/x"} {
		bad := sub
		bad.Endpoint = endpoint
		if ValidDesktopSubscription(bad) {
			t.Errorf("accepted %s", endpoint)
		}
	}
	sub.Keys.Auth = "bad"
	if ValidDesktopSubscription(sub) {
		t.Error("accepted invalid auth key")
	}
}

type pushTransport func(*http.Request) (*http.Response, error)

func (f pushTransport) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestDesktopDeliveryPostgres(t *testing.T) {
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
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := conn.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	schema := "desktop_" + strings.ReplaceAll(db.NewID(), "-", "")
	exec("CREATE SCHEMA " + schema + "; SET search_path TO " + schema + ", public")
	defer conn.Exec("DROP SCHEMA " + schema + " CASCADE")
	database := db.NewDatabase(conn)
	ctx := context.Background()
	if err := database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	actor, viewer, sid, pid := db.NewID(), db.NewID(), db.NewID(), db.NewID()
	for _, id := range []string{actor, viewer} {
		exec("INSERT INTO users(id,name,email,role_id,username,password_hash) SELECT $1::uuid,'test',$1::uuid::text||'@example.test',id,$1::uuid::text,'test' FROM roles WHERE name='Admin'", id)
	}
	exec("INSERT INTO sessions(id,user_id,token_hash,expires_at) VALUES($1,$2,'test',now()+interval '1 hour')", sid, viewer)
	sub := testSubscription(t)
	exec("INSERT INTO desktop_subscriptions(id,session_id,endpoint,p256dh,auth) VALUES($1,$2,$3,$4,$5)", pid, sid, sub.Endpoint, sub.Keys.P256dh, sub.Keys.Auth)
	add := func(sender string) {
		exec("INSERT INTO notifications(id,type,actor_user_id,entity_type,entity_id,data) VALUES($1,'zoom_booking_created',$2,'zoom_booking',$1,'{}')", db.NewID(), sender)
	}
	add(viewer)
	count := func(want int) {
		t.Helper()
		var got int
		if err := conn.QueryRow("SELECT count(*) FROM desktop_deliveries").Scan(&got); err != nil || got != want {
			t.Fatalf("queue %d want %d, %v", got, want, err)
		}
	}
	count(0)
	add(actor)
	count(1)
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	calls, status := 0, 503
	opts := webpush.Options{Subscriber: "test@example.test", VAPIDPrivateKey: priv, VAPIDPublicKey: pub, HTTPClient: pushTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Content-Encoding") != "aes128gcm" || r.Header.Get("Authorization") == "" {
			t.Fatal("missing encryption or authorization")
		}
		rec := httptest.NewRecorder()
		rec.WriteHeader(status)
		return rec.Result(), nil
	})}
	send := func() {
		t.Helper()
		if err := sendNextDesktop(ctx, database, &opts); err != nil {
			t.Fatal(err)
		}
	}
	send()
	count(1)
	exec("UPDATE desktop_deliveries SET next_attempt=now()")
	status = 201
	send()
	count(0)
	add(actor)
	exec("UPDATE sessions SET revoked_at=now()")
	send()
	count(0)
	if calls != 2 {
		t.Fatal("unexpected delivery after logout")
	}
	exec("UPDATE sessions SET revoked_at=NULL")
	// These producers insert recipients after the notification.
	for _, kind := range []string{"workspace_task", "work_item_assigned", "leave_request_created", "overtime_submitted"} {
		event := db.NewID()
		switch kind {
		case "workspace_task", "work_item_assigned":
			exec(`WITH event AS (
				INSERT INTO notifications(id,type,actor_user_id,entity_type,entity_id,workspace_division_id,data)
				SELECT $1,$2,$3,'work_item',$1,id,'{}' FROM divisions ORDER BY id LIMIT 1 RETURNING id)
				INSERT INTO notification_recipients SELECT id,$4 FROM event`, event, kind, actor, viewer)
		case "leave_request_created":
			exec("INSERT INTO leave_requests(id,user_id,request_id,request_type,start_date,end_date,reason) VALUES($1,$2,$1,'CUTI',current_date,current_date,'test')", event, actor)
			exec(`WITH event AS (
				INSERT INTO notifications(id,type,actor_user_id,entity_type,entity_id,data)
				VALUES($1,$2,$3,'leave_request',$1,'{}') RETURNING id)
				INSERT INTO notification_recipients(notification_id,user_id) SELECT id,$4 FROM event`, event, kind, actor, viewer)
		case "overtime_submitted":
			exec("INSERT INTO overtime_sheets(id,user_id,period,status) VALUES($1,$2,date_trunc('month',now())::date,'SUBMITTED')", event, actor)
			exec(`WITH event AS (
				INSERT INTO notifications(id,type,actor_user_id,entity_type,entity_id,data)
				VALUES($1,$2,$3,'overtime_sheet',$1,'{}') RETURNING id)
				INSERT INTO notification_recipients(notification_id,user_id) SELECT id,$4 FROM event`, event, kind, actor, viewer)
		}
		count(1)
		exec("DELETE FROM notification_recipients WHERE notification_id=$1", event)
		send()
		count(0)
		if calls != 2 {
			t.Fatal("sent after recipient removal", kind)
		}
	}
	t.Setenv("VAPID_PRIVATE_KEY", priv)
	t.Setenv("VAPID_PUBLIC_KEY", pub)
	t.Setenv("VAPID_SUBJECT", "test@example.test")
	t.Setenv("EMAIL_ENABLED", "false")
	when := time.Date(2026, 11, 25, 8, 16, 0, 0, time.FixedZone("WIB", 7*3600))
	database.Now = func() time.Time { return when }
	remind := func() { NewAttendanceReminderService(database).checkAndSendReminders(ctx) }
	remind()
	remind()
	count(1)
	exec("DELETE FROM desktop_deliveries")
	remind()
	count(0)
	when = when.AddDate(0, 0, 1)
	exec("INSERT INTO attendance(id,user_id,attendance_date,mode,clock_in) VALUES($1,$2,$3,'WFO',now())", db.NewID(), viewer, when.Format("2006-01-02"))
	remind()
	count(0)
	when = when.AddDate(0, 0, 1)
	exec("INSERT INTO leave_requests(id,user_id,request_id,request_type,start_date,end_date,reason,status,reviewed_by,reviewed_at) VALUES($1,$2,$1,'CUTI',$3,$3,'test','APPROVED',$4,now())", db.NewID(), viewer, when.Format("2006-01-02"), actor)
	remind()
	count(0)
	add(actor)
	status = 410
	send()
	count(0)
	var left int
	if err := conn.QueryRow("SELECT count(*) FROM desktop_subscriptions").Scan(&left); err != nil || left != 0 {
		t.Fatal("expired subscription retained", err)
	}
}
