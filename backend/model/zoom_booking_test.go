package model

import (
	"strings"
	"testing"
	"time"
)

func TestValidZoomBooking(t *testing.T) {
	start := time.Date(2026, 9, 10, 9, 0, 0, 0, time.UTC)
	if !ValidZoomBooking(ZoomBookingInput{Title: "Project meeting", ZoomResourceID: 1, StartAt: start, EndAt: start.Add(time.Hour)}) {
		t.Fatal("valid booking was rejected")
	}
	for _, input := range []ZoomBookingInput{
		{Title: "", ZoomResourceID: 1, StartAt: start, EndAt: start.Add(time.Hour)},
		{Title: "Meeting", ZoomResourceID: 0, StartAt: start, EndAt: start.Add(time.Hour)},
		{Title: "Meeting", ZoomResourceID: 1, StartAt: start, EndAt: start},
		{Title: "Meeting", ZoomResourceID: 1, StartAt: start, EndAt: start.Add(-time.Minute)},
	} {
		if ValidZoomBooking(input) {
			t.Fatalf("invalid booking was accepted: %#v", input)
		}
	}
}

func TestValidZoomResourceUpdate(t *testing.T) {
	if !ValidZoomResourceUpdate(ZoomResourceUpdateInput{MeetingID: "123 456", HostKey: "654321", MeetingURL: "https://zoom.us/j/123", Description: "NP account"}) {
		t.Fatal("valid resource information was rejected")
	}
	if ValidZoomResourceUpdate(ZoomResourceUpdateInput{MeetingURL: "javascript:alert(1)"}) {
		t.Fatal("unsafe meeting URL was accepted")
	}
	if ValidZoomResourceUpdate(ZoomResourceUpdateInput{HostKey: strings.Repeat("1", 101)}) {
		t.Fatal("host key longer than 100 characters was accepted")
	}
}

func TestZoomBookingStatus(t *testing.T) {
	start := time.Date(2026, 9, 10, 9, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	for _, test := range []struct {
		now  time.Time
		want string
	}{
		{start.Add(-time.Second), "upcoming"},
		{start, "ongoing"},
		{end.Add(-time.Second), "ongoing"},
		{end, "completed"},
	} {
		if got := ZoomBookingStatus(start, end, test.now); got != test.want {
			t.Fatalf("status at %v = %q, want %q", test.now, got, test.want)
		}
	}
}
