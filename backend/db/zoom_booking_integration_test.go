package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"my-ple-backend/model"
)

func TestZoomBookingConflictAndOwnershipPostgres(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is required")
	}
	conn, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	database := NewDatabase(conn)
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`DELETE FROM users WHERE name = 'Zoom Test' AND email LIKE '%@example.test'`); err != nil {
		t.Fatal(err)
	}
	role, err := database.FindRoleByName(context.Background(), "Project Engineer")
	if err != nil {
		t.Fatal(err)
	}
	user := model.User{ID: NewID(), Name: "Zoom Test", Email: NewID() + "@example.test", Username: "zoom-" + NewID()[:8], RoleID: role.ID, PasswordHash: "test", EmploymentStatus: "Permanent Employee", IsActive: true}
	if _, err := database.CreateUser(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(`DELETE FROM zoom_bookings WHERE user_id = $1`, user.ID)
		_, _ = conn.Exec(`DELETE FROM users WHERE id = $1`, user.ID)
	})

	start := time.Date(2099, 9, 10, 9, 0, 0, 0, time.UTC)
	created, err := database.CreateZoomBooking(context.Background(), user.ID, model.ZoomBookingInput{Title: "First", ZoomResourceID: 1, StartAt: start, EndAt: start.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.CreateZoomBooking(context.Background(), user.ID, model.ZoomBookingInput{Title: "Overlap", ZoomResourceID: 1, StartAt: start.Add(30 * time.Minute), EndAt: start.Add(90 * time.Minute)}); !errors.Is(err, ErrZoomBookingConflict) {
		t.Fatalf("overlap error = %v", err)
	}
	for _, resourceID := range []int{2, 3} {
		if _, err := database.CreateZoomBooking(context.Background(), user.ID, model.ZoomBookingInput{Title: "Parallel", ZoomResourceID: resourceID, StartAt: start.Add(30 * time.Minute), EndAt: start.Add(90 * time.Minute)}); err != nil {
			t.Fatalf("parallel resource %d rejected: %v", resourceID, err)
		}
	}
	if _, err := database.CreateZoomBooking(context.Background(), user.ID, model.ZoomBookingInput{Title: "Back to back", ZoomResourceID: 1, StartAt: start.Add(time.Hour), EndAt: start.Add(2 * time.Hour)}); err != nil {
		t.Fatalf("back-to-back rejected: %v", err)
	}
	items, err := database.ListZoomBookings(context.Background(), start, start.Add(3*time.Hour))
	if err != nil || len(items) != 4 || items[0].ZoomResource.ID == 0 {
		t.Fatalf("list = %d, %v", len(items), err)
	}
	if _, err := database.UpdateZoomBooking(context.Background(), created.ID, NewID(), model.ZoomBookingInput{Title: "Nope", ZoomResourceID: 1, StartAt: start, EndAt: start.Add(time.Hour)}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("non-owner update error = %v", err)
	}
	resources, err := database.ListZoomResources(context.Background())
	if err != nil || len(resources) != 3 || resources[0].Code != "NP" || resources[1].Code != "MRP" || resources[2].Code != "IB2" {
		t.Fatalf("resources = %#v, %v", resources, err)
	}
}

func TestZoomBookingNotificationLifecyclePostgres(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is required")
	}
	conn, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	database := NewDatabase(conn)
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	role, err := database.FindRoleByName(context.Background(), "Project Engineer")
	if err != nil {
		t.Fatal(err)
	}
	actor := model.User{ID: NewID(), Name: "Notification Actor", Email: NewID() + "@example.test", Username: "actor-" + NewID()[:8], RoleID: role.ID, PasswordHash: "test", EmploymentStatus: "Permanent Employee", IsActive: true}
	recipient := model.User{ID: NewID(), Name: "Notification Recipient", Email: NewID() + "@example.test", Username: "recipient-" + NewID()[:8], RoleID: role.ID, PasswordHash: "test", EmploymentStatus: "Permanent Employee", IsActive: true}
	for _, user := range []model.User{actor, recipient} {
		if _, err := database.CreateUser(context.Background(), user); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _, _ = conn.Exec(`DELETE FROM users WHERE id IN ($1,$2)`, actor.ID, recipient.ID) })

	start := time.Date(2099, 10, 4, 13, 0, 0, 0, time.UTC)
	_, notification, err := database.CreateZoomBookingWithNotification(context.Background(), actor.ID, model.ZoomBookingInput{Title: "Notification", ZoomResourceID: 2, StartAt: start, EndAt: start.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if notification.Actor.ID != actor.ID || notification.Booking == nil || notification.Booking.ZoomName != "Zoom 2" {
		t.Fatalf("notification = %#v", notification)
	}
	if own, err := database.ListNotifications(context.Background(), actor.ID); err != nil || len(own) != 0 {
		t.Fatalf("actor feed = %#v, %v", own, err)
	}
	feed, err := database.ListNotifications(context.Background(), recipient.ID)
	if err != nil || len(feed) != 1 || feed[0].ID != notification.ID || feed[0].ReadAt != nil {
		t.Fatalf("recipient feed = %#v, %v", feed, err)
	}
	if count, err := database.UnreadNotificationCount(context.Background(), recipient.ID); err != nil || count != 1 {
		t.Fatalf("unread = %d, %v", count, err)
	}
	if err := database.MarkNotificationRead(context.Background(), notification.ID, recipient.ID); err != nil {
		t.Fatal(err)
	}
	if count, _ := database.UnreadNotificationCount(context.Background(), recipient.ID); count != 0 {
		t.Fatalf("unread after read = %d", count)
	}
	if err := database.DismissNotification(context.Background(), notification.ID, recipient.ID); err != nil {
		t.Fatal(err)
	}
	if feed, _ := database.ListNotifications(context.Background(), recipient.ID); len(feed) != 0 {
		t.Fatalf("dismissed feed = %#v", feed)
	}
	var masters int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM notifications WHERE id=$1`, notification.ID).Scan(&masters); err != nil || masters != 1 {
		t.Fatalf("master count = %d, %v", masters, err)
	}
	if _, err := conn.Exec(`UPDATE notifications SET created_at=NOW()-INTERVAL '73 hours' WHERE id=$1`, notification.ID); err != nil {
		t.Fatal(err)
	}
	if deleted, err := database.DeleteExpiredNotifications(context.Background()); err != nil || deleted != 1 {
		t.Fatalf("deleted = %d, %v", deleted, err)
	}
	var states int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM notification_user_states WHERE notification_id=$1`, notification.ID).Scan(&states); err != nil || states != 0 {
		t.Fatalf("state count = %d, %v", states, err)
	}
}
