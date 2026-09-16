package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"my-ple-backend/db"
	"my-ple-backend/model"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// Reuse the existing fixture helper, which only permits the isolated localhost database.
func TestAttendanceIntegration(t *testing.T) {
	database := checklistTestDB(t)
	t.Setenv("ATTENDANCE_OFFICE_LATITUDE", "0")
	t.Setenv("ATTENDANCE_OFFICE_LONGITUDE", "0")
	var now atomic.Int64
	now.Store(time.Date(2026, 9, 30, 16, 59, 0, 0, time.UTC).Unix())
	database.Now = func() time.Time { return time.Unix(now.Load(), 0) }
	server := httptest.NewServer(newHandler(database, "http://test.local"))
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	request := func(method, path string, body any, want int) map[string]json.RawMessage {
		t.Helper()
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(method, server.URL+path, bytes.NewReader(raw))
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		response, _ := io.ReadAll(res.Body)
		if res.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, res.StatusCode, want, response)
		}
		var result map[string]json.RawMessage
		if err := json.Unmarshal(response, &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	decode := func(response map[string]json.RawMessage) model.Attendance {
		t.Helper()
		var item model.Attendance
		if err := json.Unmarshal(response["attendance"], &item); err != nil {
			t.Fatal(err)
		}
		return item
	}
	request("GET", "/attendance", nil, 401)
	suffix := db.NewID()
	var userID string
	for i, role := range []string{"Admin", "Management", "HR", "Project Engineer", "Finance", "IT", "Business Development", "PMO"} {
		username := fmt.Sprintf("attendance-%d-%s", i, suffix)
		checklistTestUser(t, database, role, username)
		response := request("POST", "/auth/login", map[string]string{"username": username, "password": "Checklist-test-only-2026"}, 200)
		var user struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(response["user"], &user)
		userID = user.ID
		history := request("GET", "/attendance?month=2026-09", nil, 200)
		if string(history["records"]) != "[]" || string(history["officeConfigured"]) != "true" {
			t.Fatal("unexpected initial history", history)
		}
	}
	t.Cleanup(func() { _, _ = database.Conn.Exec(`DELETE FROM users WHERE username LIKE $1`, "attendance-%-"+suffix) })
	request("GET", "/attendance?month=2026-13", nil, 400)
	request("POST", "/attendance/clock-in", map[string]any{"mode": "WFA"}, 400)
	request("POST", "/attendance/clock-in", map[string]any{"mode": "WFA", "latitude": 0, "longitude": 0, "accuracy": 5, "userId": userID}, 400)
	request("POST", "/attendance/clock-in", map[string]any{"mode": "WFO", "latitude": 1, "longitude": 0, "accuracy": 5}, 403)
	request("POST", "/attendance/clock-in", map[string]any{"mode": "WFO", "latitude": 0, "longitude": 0, "accuracy": 101}, 400)
	zero, accuracy := 0.0, 5.0
	input := model.AttendanceInput{Mode: "WFO", Latitude: &zero, Longitude: &zero, Accuracy: &accuracy}
	first := decode(request("POST", "/attendance/clock-in", input, 200))
	if first.Date != "2026-09-30" || first.Latitude != nil || first.Longitude != nil || first.ClockOut != nil {
		t.Fatalf("WFO privacy/state: %+v", first)
	}
	// Simultaneous retries must converge on the same row and timestamp.
	results := make(chan model.Attendance, 8)
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() {
			item, err := database.ClockIn(context.Background(), userID, input, db.AttendanceOffice{Configured: true})
			results <- item
			failures <- err
		}()
	}
	for i := 0; i < 8; i++ {
		item, err := <-results, <-failures
		if err != nil || item.ID != first.ID || !item.ClockIn.Equal(first.ClockIn) {
			t.Fatalf("duplicate: %+v %v", item, err)
		}
	}
	// Another employee cannot read or close this record.
	ownerClient := client
	jar, _ = cookiejar.New(nil)
	client = &http.Client{Jar: jar}
	request("POST", "/auth/login", map[string]string{"username": "attendance-0-" + suffix, "password": "Checklist-test-only-2026"}, 200)
	history := request("GET", "/attendance?month=2026-09", nil, 200)
	if string(history["records"]) != "[]" || string(history["open"]) != "null" {
		t.Fatal("cross-user read")
	}
	request("POST", "/attendance/"+first.ID+"/clock-out", nil, 404)
	client = ownerClient
	now.Store(time.Date(2026, 9, 30, 17, 0, 0, 0, time.UTC).Unix()) // October 1, 00:00 WIB.
	request("POST", "/attendance/clock-in", input, 409)
	history = request("GET", "/attendance?month=2026-10", nil, 200)
	if string(history["records"]) != "[]" || string(history["today"]) != "null" || string(history["open"]) == "null" {
		t.Fatal("overnight state")
	}
	closed := decode(request("POST", "/attendance/"+first.ID+"/clock-out", nil, 200))
	if closed.ClockOut == nil || closed.Date != first.Date {
		t.Fatal("overnight close changed date")
	}
	now.Store(time.Date(2026, 10, 1, 1, 0, 0, 0, time.UTC).Unix())
	input.Mode = "WFA"
	remote := 10.0
	input.Latitude = &remote
	second := decode(request("POST", "/attendance/clock-in", input, 200))
	if second.Date != "2026-10-01" || second.Latitude == nil || *second.Latitude != 10 || second.Longitude == nil {
		t.Fatal("WFA location missing")
	}
	now.Store(time.Date(2026, 10, 1, 14, 0, 0, 0, time.UTC).Unix()) // 21:00 WIB, ordinary clock-out.
	second = decode(request("POST", "/attendance/"+second.ID+"/clock-out", nil, 200))
	if second.ClockOut == nil || second.ClockOut.Unix() != now.Load() {
		t.Fatal("late clock-out changed actual time")
	}
	again := decode(request("POST", "/attendance/"+first.ID+"/clock-out", nil, 200))
	if !again.ClockOut.Equal(*closed.ClockOut) {
		t.Fatal("retry overwrote clock-out")
	}
	again = decode(request("POST", "/attendance/clock-in", input, 200))
	if again.ID != second.ID || !again.ClockOut.Equal(*second.ClockOut) {
		t.Fatal("second workday allowed")
	}
	history = request("GET", "/attendance?month=2026-09", nil, 200)
	var records []model.Attendance
	_ = json.Unmarshal(history["records"], &records)
	if len(records) != 1 || records[0].ID != first.ID {
		t.Fatal("monthly history")
	}
	var today model.Attendance
	_ = json.Unmarshal(history["today"], &today)
	if today.ID != second.ID {
		t.Fatal("history filter changed today's action")
	}
	// Missing office configuration affects WFO only, even for direct API calls.
	t.Setenv("ATTENDANCE_OFFICE_LATITUDE", "")
	unconfigured := httptest.NewServer(newHandler(database, "http://test.local"))
	defer unconfigured.Close()
	oldURL := server.URL
	server.URL = unconfigured.URL
	request("POST", "/attendance/clock-in", model.AttendanceInput{Mode: "WFO", Latitude: &zero, Longitude: &zero, Accuracy: &accuracy}, 503)
	request("POST", "/attendance/clock-in", input, 200)
	server.URL = oldURL
}

func TestAttendanceBrowserServer(t *testing.T) {
	if os.Getenv("ATTENDANCE_BROWSER_SERVER") != "1" {
		t.Skip("browser fixture server disabled")
	}
	database := checklistTestDB(t)
	t.Setenv("ATTENDANCE_OFFICE_LATITUDE", "0")
	t.Setenv("ATTENDANCE_OFFICE_LONGITUDE", "0")
	for _, role := range []string{"Admin", "Finance", "HR"} {
		username := "attendance-browser-" + role
		checklistTestUser(t, database, role, username)
		if _, err := database.Conn.Exec(`DELETE FROM attendance WHERE user_id=(SELECT id FROM users WHERE username=$1)`, username); err != nil {
			t.Fatal(err)
		}
	}
	server := &http.Server{Addr: "127.0.0.1:3001", Handler: newHandler(database, "http://127.0.0.1:5189"), ReadHeaderTimeout: 5 * time.Second}
	t.Log("Attendance browser fixture ready on 3001")
	if err := server.ListenAndServe(); err != nil {
		t.Fatal(err)
	}
}
