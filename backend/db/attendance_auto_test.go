package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"my-ple-backend/model"
)

func TestAttendanceAutoClockOutPostgres(t *testing.T) {
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
	_, err = conn.Exec(`CREATE TEMP TABLE users(id uuid primary key);
 CREATE TEMP TABLE attendance(id uuid primary key,user_id uuid,attendance_date date,mode text,clock_in timestamptz,clock_out timestamptz,latitude float8,longitude float8,notes text);`)
	if err != nil {
		t.Fatal(err)
	}
	id := NewID()
	if _, err = conn.Exec("INSERT INTO users VALUES ($1)", id); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Date(2026, 11, 19, 8, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	d := &Database{Conn: conn, Now: func() time.Time { return now }}
	zero := 0.0
	clockIn := func() model.Attendance {
		t.Helper()
		item, err := d.ClockIn(ctx, id, model.AttendanceInput{Mode: "WFA", Latitude: &zero, Longitude: &zero, Accuracy: &zero}, AttendanceOffice{})
		if err != nil {
			t.Fatal(err)
		}
		return item
	}
	first := clockIn()
	now = now.Add(13*time.Hour - time.Second)
	if err = d.AutoClockOut(ctx); err != nil {
		t.Fatal(err)
	}
	if clockIn().ClockOut != nil {
		t.Fatal("closed before 21:00")
	}
	now = now.Add(time.Second)
	if err = d.AutoClockOut(ctx); err != nil {
		t.Fatal(err)
	}
	var out time.Time
	if err = conn.QueryRow("SELECT clock_out FROM attendance WHERE id=$1", first.ID).Scan(&out); err != nil {
		t.Fatal(err)
	}
	if !out.Equal(now) {
		t.Fatalf("cutoff clock-out = %v", out)
	}
	now = now.Add(11 * time.Hour)
	second := clockIn()
	if second.ID == first.ID || second.ClockOut != nil {
		t.Fatal("expected fresh open attendance next morning")
	}
	now = now.Add(24 * time.Hour)
	third := clockIn()
	if third.ID == second.ID || third.ClockOut != nil {
		t.Fatal("missed sweep blocked next clock-in")
	}
	now = now.Add(9 * time.Hour)
	manual, err := d.ClockOut(ctx, id, third.ID)
	if err != nil || manual.ClockOut == nil || !manual.ClockOut.Equal(now) {
		t.Fatalf("manual clock-out: %v, %v", manual.ClockOut, err)
	}
	now = now.Add(5 * time.Hour)
	if err = d.AutoClockOut(ctx); err != nil {
		t.Fatal(err)
	}
	if out := clockIn().ClockOut; out == nil || !out.Equal(*manual.ClockOut) {
		t.Fatalf("manual clock-out changed: %v", out)
	}
}
