package model

import (
	"testing"
	"time"
)

func TestReportingPeriodResetsTuesdayAtMidnightWIB(t *testing.T) {
	location, _ := time.LoadLocation(ReportingTimezone)
	start, next := ReportingPeriodWeek(time.Date(2026, 9, 14, 23, 59, 59, 0, location))
	if !start.Equal(time.Date(2026, 9, 8, 0, 0, 0, 0, location)) || !next.Equal(time.Date(2026, 9, 15, 0, 0, 0, 0, location)) {
		t.Fatalf("Monday period = %s to %s", start, next)
	}
	start, next = ReportingPeriodWeek(time.Date(2026, 9, 15, 0, 0, 0, 0, location))
	if !start.Equal(time.Date(2026, 9, 15, 0, 0, 0, 0, location)) || !next.Equal(time.Date(2026, 9, 22, 0, 0, 0, 0, location)) {
		t.Fatalf("Tuesday period = %s to %s", start, next)
	}
}
