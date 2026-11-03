package model

import (
	"testing"
	"time"
)

func TestCalculateOvertime(t *testing.T) {
	tests := []struct {
		name, start, end, dayType string
		actual, weighted          int
	}{
		{"workday after hours", "08:00", "21:00", "WORKDAY", 180, 180},
		{"workday before and after", "07:00", "19:00", "WORKDAY", 120, 120},
		{"workday excludes time before eighteen", "17:00", "18:00", "WORKDAY", 0, 0},
		{"workday starts at eighteen", "18:00", "19:00", "WORKDAY", 60, 60},
		{"workday morning", "06:00", "09:00", "WORKDAY", 120, 120},
		{"workday office hours", "08:00", "17:00", "WORKDAY", 0, 0},
		{"workday overnight excludes next office hours", "22:00", "09:00", "WORKDAY", 600, 600},
		{"workday overnight before eight", "22:00", "02:00", "WORKDAY", 240, 240},
		{"workday overnight reaches next evening", "19:00", "18:30", "WORKDAY", 810, 810},
		{"holiday full duration", "08:00", "10:00", "HOLIDAY", 120, 240},
		{"rounds fifteen minutes up", "18:00", "18:15", "WORKDAY", 30, 30},
		{"rounds fourteen minutes down", "18:00", "18:14", "WORKDAY", 0, 0},
		{"overnight", "22:00", "02:00", "HOLIDAY", 240, 480},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, weighted, err := CalculateOvertime(tt.start, tt.end, tt.dayType)
			if err != nil || actual != tt.actual || weighted != tt.weighted {
				t.Fatalf("got actual=%d weighted=%d err=%v", actual, weighted, err)
			}
		})
	}
}

func TestCalculateOvertimeRejectsEqualTimes(t *testing.T) {
	if _, _, err := CalculateOvertime("08:00", "08:00", "WORKDAY"); err == nil {
		t.Fatal("expected equal times to fail")
	}
}

func TestOvertimeSubmissionWindow(t *testing.T) {
	jakarta, _ := time.LoadLocation(ReportingTimezone)
	tests := []struct {
		at      time.Time
		period  string
		allowed bool
	}{
		{time.Date(2026, 10, 25, 23, 59, 59, 0, jakarta), "2026-10", false},
		{time.Date(2026, 10, 26, 0, 0, 0, 0, jakarta), "2026-10", true},
		{time.Date(2026, 11, 5, 23, 59, 59, 0, jakarta), "2026-10", true},
		{time.Date(2026, 11, 6, 0, 0, 0, 0, jakarta), "2026-10", false},
		{time.Date(2027, 1, 3, 12, 0, 0, 0, jakarta), "2026-12", true},
	}
	for _, tt := range tests {
		if got := OvertimeSubmissionOpen(tt.at, tt.period); got != tt.allowed {
			t.Errorf("%s period %s: got %v", tt.at, tt.period, got)
		}
	}
}
