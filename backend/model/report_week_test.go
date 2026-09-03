package model

import (
	"testing"
	"time"
)

func TestReportingWeekUsesJakartaMondayBoundary(t *testing.T) {
	loc, err := time.LoadLocation(ReportingTimezone)
	if err != nil {
		t.Fatal(err)
	}

	sunday := time.Date(2026, 8, 30, 23, 59, 59, 0, loc)
	start, next := ReportingWeek(sunday)
	if got := start.Format(time.RFC3339); got != "2026-08-24T00:00:00+07:00" {
		t.Fatalf("Sunday start = %s", got)
	}
	if got := next.Format(time.RFC3339); got != "2026-08-31T00:00:00+07:00" {
		t.Fatalf("Sunday next = %s", got)
	}

	monday := next
	start, next = ReportingWeek(monday)
	if got := start.Format(time.RFC3339); got != "2026-08-31T00:00:00+07:00" {
		t.Fatalf("Monday start = %s", got)
	}
	if got := next.Format(time.RFC3339); got != "2026-09-07T00:00:00+07:00" {
		t.Fatalf("Monday next = %s", got)
	}
	if got := IndonesianWeekLabel(start, next); got != "31 Agustus - 6 September 2026" {
		t.Fatalf("label = %q", got)
	}
}

func TestIndonesianWeekLabelWithinOneMonth(t *testing.T) {
	loc, _ := time.LoadLocation(ReportingTimezone)
	start := time.Date(2026, 9, 7, 0, 0, 0, 0, loc)
	if got := IndonesianWeekLabel(start, start.AddDate(0, 0, 7)); got != "7 - 13 September 2026" {
		t.Fatalf("label = %q", got)
	}
}
