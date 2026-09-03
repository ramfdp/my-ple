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
