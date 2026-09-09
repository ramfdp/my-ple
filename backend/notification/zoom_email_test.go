package notification

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"my-ple-backend/db"
	"my-ple-backend/email"
	"my-ple-backend/model"
)

func TestZoomEmailPostgres(t *testing.T) {
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
	schema := "zoom_test_" + strings.ReplaceAll(db.NewID(), "-", "")
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
	creator := db.NewID()
	other := db.NewID()
	for i, r := range []struct {
		id, address string
		active      bool
	}{{creator, "Creator@example.com", true}, {other, "other@example.com", true}, {db.NewID(), " creator@EXAMPLE.com ", true}, {db.NewID(), "", true}, {db.NewID(), "bad-address", true}, {db.NewID(), "inactive@example.com", false}} {
		_, err = conn.ExecContext(ctx, `INSERT INTO users(id,name,email,username,password_hash,role_id,is_active) VALUES($1,$2,$3,$2,'unused',(SELECT id FROM roles WHERE name='PMO'),$4)`, r.id, fmt.Sprintf("test%d", i), r.address, r.active)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err = conn.ExecContext(ctx, `UPDATE zoom_resources SET meeting_url='https://zoom.us/j/'||id,passcode='123'`); err != nil {
		t.Fatal(err)
	}
	input := model.ZoomBookingInput{Title: "Coordination", Description: strings.Repeat("progress ", 100), ZoomResourceID: 2, StartAt: time.Now().Add(time.Hour), EndAt: time.Now().Add(2 * time.Hour)}
	service := &Service{DB: database}
	_, err = service.CreateZoomBooking(ctx, creator, input)
	if err != nil {
		t.Fatal(err)
	}
	count := func(query string, want int) {
		t.Helper()
		var n int
		if err := conn.QueryRowContext(ctx, query).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != want {
			t.Fatalf("count %d want %d: %s", n, want, query)
		}
	}
	count(`SELECT count(*) FROM zoom_booking_email_deliveries`, 2)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = service.CreateZoomBooking(cancelled, creator, input); err == nil {
		t.Fatal("cancelled booking accepted")
	}
	invalid := input
	invalid.EndAt = invalid.StartAt
	if _, err = service.CreateZoomBooking(ctx, creator, invalid); err == nil {
		t.Fatal("invalid booking accepted")
	}
	count(`SELECT count(*) FROM zoom_booking_email_deliveries`, 2)
	feed, err := database.ListNotifications(ctx, creator)
	if err != nil || len(feed) != 0 {
		t.Fatalf("creator feed: %v %v", feed, err)
	}
	feed, err = database.ListNotifications(ctx, other)
	if err != nil || len(feed) != 1 || feed[0].Booking.Description != input.Description {
		t.Fatalf("other feed: %v %v", feed, err)
	}
	if _, err = service.CreateZoomBooking(ctx, creator, input); !errors.Is(err, db.ErrZoomBookingConflict) {
		t.Fatalf("duplicate request: %v", err)
	}
	count(`SELECT count(*) FROM zoom_booking_email_deliveries`, 2)
	sends := map[string]int{}
	worker := ZoomEmailWorker{DB: database, Send: func(_ context.Context, to, id string, b model.ZoomBooking, confirmation bool) error {
		sends[to]++
		if confirmation != (to == "creator@example.com") || b.Description != input.Description || b.ZoomResource.MeetingURL != "https://zoom.us/j/2" || b.ZoomResource.HostKey != "" {
			t.Fatal("incorrect mail payload")
		}
		if to == "other@example.com" {
			return errors.New("simulated failure with secret that must not be logged")
		}
		return nil
	}}
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	count(`SELECT count(*) FROM zoom_bookings`, 1)
	count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE status='sent'`, 1)
	for i := 0; i < 4; i++ {
		if _, err = conn.ExecContext(ctx, `UPDATE zoom_booking_email_deliveries SET next_attempt_at=NOW()`); err != nil {
			t.Fatal(err)
		}
		if err = worker.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if sends["creator@example.com"] != 1 || sends["other@example.com"] != 3 {
		t.Fatal(sends)
	}
	// A duplicate enqueue cannot create another delivery for the booking/address.
	_, err = conn.ExecContext(ctx, `INSERT INTO zoom_booking_email_deliveries(id,booking_id,recipient_email,is_creator,payload) SELECT $1,booking_id,recipient_email,is_creator,payload FROM zoom_booking_email_deliveries LIMIT 1 ON CONFLICT(booking_id,recipient_email) DO NOTHING`, db.NewID())
	if err != nil {
		t.Fatal(err)
	}
	count(`SELECT count(*) FROM zoom_booking_email_deliveries`, 2)
	// Disabled creation still saves and never queues or sends email.
	t.Setenv("ZOOM_EMAIL_ENABLED", "false")
	input.ZoomResourceID = 3
	if _, err = service.CreateZoomBooking(ctx, creator, input); err != nil {
		t.Fatal(err)
	}
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	count(`SELECT count(*) FROM zoom_booking_email_deliveries`, 2)
	// SMTP acceptance uncertainty must never automatically retry.
	t.Setenv("ZOOM_EMAIL_ENABLED", "true")
	_, err = conn.ExecContext(ctx, `UPDATE zoom_booking_email_deliveries SET status='pending',attempt_count=0,next_attempt_at=NOW() WHERE recipient_email='other@example.com'`)
	if err != nil {
		t.Fatal(err)
	}
	worker.Send = func(context.Context, string, string, model.ZoomBooking, bool) error { return email.ErrUncertain }
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE status='uncertain' AND attempt_count=1`, 1)
	// Deactivation after queuing must also suppress delivery.
	_, err = conn.ExecContext(ctx, `UPDATE users SET is_active=false WHERE id=$1`, other)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.ExecContext(ctx, `UPDATE zoom_booking_email_deliveries SET status='pending',attempt_count=0,next_attempt_at=NOW() WHERE recipient_email='other@example.com'`)
	if err != nil {
		t.Fatal(err)
	}
	worker.Send = func(context.Context, string, string, model.ZoomBooking, bool) error {
		t.Fatal("inactive recipient sent email")
		return nil
	}
	if err = worker.Run(ctx); err != nil {
		t.Fatal(err)
	}
	count(`SELECT count(*) FROM zoom_booking_email_deliveries WHERE status='skipped'`, 1)
	// A queue failure is isolated by a savepoint; the accepted booking survives.
	_, err = conn.ExecContext(ctx, `ALTER TABLE zoom_booking_email_deliveries RENAME TO temporarily_unavailable`)
	if err != nil {
		t.Fatal(err)
	}
	input.ZoomResourceID = 1
	if _, err = service.CreateZoomBooking(ctx, creator, input); err != nil {
		t.Fatal(err)
	}
	count(`SELECT count(*) FROM zoom_bookings`, 3)
}
