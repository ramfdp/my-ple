package model

import (
	"testing"
	"time"
)

func TestReportingPeriodAndFinalizationBoundary(t *testing.T) {
	location, _ := time.LoadLocation(ReportingTimezone)
	monday := time.Date(2026, 9, 28, 10, 0, 0, 0, location)
	start, next := ReportingPeriodWeek(monday)
	if want := "2026-09-28/2026-10-05"; start.Format("2006-01-02")+"/"+next.Format("2006-01-02") != want {
		t.Fatalf("period = %s/%s, want %s", start.Format("2006-01-02"), next.Format("2006-01-02"), want)
	}
	if got := IndonesianWeekLabel(start, next); got != "28 September - 4 Oktober 2026" {
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
