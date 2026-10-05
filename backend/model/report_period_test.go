package model

import (
	"testing"
	"time"
)

func TestReportingPeriodAndFinalizationBoundary(t *testing.T) {
	location, _ := time.LoadLocation(ReportingTimezone)
	monday := time.Date(2026, 9, 28, 10, 0, 0, 0, location)
	start, next := ReportingPeriodWeek(monday)
	if want := "2026-09-21/2026-09-28"; start.Format("2006-01-02")+"/"+next.Format("2006-01-02") != want {
		t.Fatalf("period = %s/%s, want %s", start.Format("2006-01-02"), next.Format("2006-01-02"), want)
	}
	if got := IndonesianWeekLabel(start, next); got != "21 - 27 September 2026" {
		t.Fatalf("label = %q, want Monday-Sunday range", got)
	}
	if got := WeeklyFinalizationBoundary(monday).Format("2006-01-02 15:04"); got != "2026-09-21 00:00" {
		t.Fatalf("Monday boundary = %s, want previous week", got)
	}
	tuesday := monday.AddDate(0, 0, 1)
	if got := WeeklyFinalizationBoundary(tuesday).Format("2006-01-02 15:04"); got != "2026-09-28 00:00" {
		t.Fatalf("Tuesday boundary = %s, want current Monday", got)
	}
}

func TestMondayRemainsInClosingPeriod(t *testing.T) {
	for _, stamp := range []string{"2026-10-04T23:59:59+07:00", "2026-10-05T00:00:00+07:00", "2026-10-05T23:59:59+07:00", "2026-10-05T16:59:59Z"} {
		now, _ := time.Parse(time.RFC3339, stamp)
		start, _ := ReportingPeriodWeek(now)
		if got := start.Format("2006-01-02"); got != "2026-09-28" {
			t.Fatalf("%s: got %s", stamp, got)
		}
	}
	now, _ := time.Parse(time.RFC3339, "2026-10-06T00:00:00+07:00")
	start, _ := ReportingPeriodWeek(now)
	if start.Format("2006-01-02") != "2026-10-05" || !WeeklyFinalizationBoundary(now).Equal(start) {
		t.Fatal("Tuesday must close the old period and open the new one")
	}
}
