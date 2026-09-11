package db

import (
	"testing"
	"time"
)

func TestWeeklyRolloverStartsTuesdayAtMidnightWIB(t *testing.T) {
	location, _ := time.LoadLocation("Asia/Jakarta")
	if weeklyRolloverReady(time.Date(2026, 9, 14, 23, 59, 59, 0, location)) {
		t.Fatal("weekly report rolled over before Tuesday")
	}
	if !weeklyRolloverReady(time.Date(2026, 9, 15, 0, 0, 0, 0, location)) {
		t.Fatal("weekly report did not roll over at Tuesday 00:00 WIB")
	}
}
