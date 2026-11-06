package model

import (
	"testing"
	"time"
)

func TestReportingPeriodSaturdayBoundary(t *testing.T) {
	for _, tt := range []struct{ instant, start, next, label string }{
		{"2026-10-30T16:59:59Z", "2026-10-23", "2026-10-30", "23 - 29 Oktober 2026"},
		{"2026-10-30T17:00:00Z", "2026-10-30", "2026-11-06", "30 Oktober - 5 November 2026"},
		{"2026-11-01T05:00:00Z", "2026-10-30", "2026-11-06", "30 Oktober - 5 November 2026"},
		{"2026-11-06T16:59:59Z", "2026-10-30", "2026-11-06", "30 Oktober - 5 November 2026"},
		{"2026-11-06T17:00:00Z", "2026-11-06", "2026-11-13", "6 - 12 November 2026"},
	} {
		t.Run(tt.instant, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tt.instant)
			if err != nil {
				t.Fatal(err)
			}
			start, next := ReportingPeriodWeek(now)
			if start.Format("2006-01-02") != tt.start || next.Format("2006-01-02") != tt.next {
				t.Fatalf("period = %v to %v", start, next)
			}
			if start.Weekday() != time.Friday || start.Hour() != 0 || start.Location().String() != ReportingTimezone {
				t.Fatalf("invalid Friday key %v", start)
			}
			if !WeeklyFinalizationBoundary(now).Equal(start) {
				t.Fatal("finalization boundary differs from period key")
			}
			end, label := ReportingPeriodDates(start, next)
			if !end.Equal(next.AddDate(0, 0, -1)) || end.Weekday() != time.Thursday || label != tt.label {
				t.Fatalf("display = %v, %q", end, label)
			}
		})
	}
}

func TestWeeklyArchivePreservesHistoricalMondayLabel(t *testing.T) {
	location, err := time.LoadLocation(ReportingTimezone)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 10, 19, 0, 0, 0, 0, location)
	period := (WeeklyArchive{WeekStart: start, WeekEnd: start.AddDate(0, 0, 7)}).Period()
	if period.Label != "19 - 25 Oktober 2026" || period.End.Format("2006-01-02") != "2026-10-25" || !period.ReadOnly {
		t.Fatalf("historical period changed: %+v", period)
	}
	friday := time.Date(2026, 10, 30, 0, 0, 0, 0, location)
	period = (WeeklyArchive{WeekStart: friday, WeekEnd: friday.AddDate(0, 0, 7)}).Period()
	if period.Label != "30 Oktober - 5 November 2026" || period.End.Format("2006-01-02") != "2026-11-05" {
		t.Fatalf("Friday period: %+v", period)
	}
}

func TestReportingWeekRemainsMondayForWorkload(t *testing.T) {
	now := time.Date(2026, 11, 1, 5, 0, 0, 0, time.UTC)
	start, next := ReportingWeek(now)
	if start.Weekday() != time.Monday || start.Format("2006-01-02") != "2026-10-26" || next.Format("2006-01-02") != "2026-11-02" {
		t.Fatalf("workload week changed: %v to %v", start, next)
	}
}
