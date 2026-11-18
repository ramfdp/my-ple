package db

import (
	"my-ple-backend/model"
	"testing"
)

func TestNormalizeOvertimeEntriesClassifiesAndAuditsOverrides(t *testing.T) {
	items, err := normalizeOvertimeEntries("2026-10", []model.OvertimeEntryInput{
		{WorkDate: "2026-10-10", StartTime: "08:00", EndTime: "10:00", DayType: "HOLIDAY", Project: "internal", Description: "Testing"},
		{WorkDate: "2026-10-12", StartTime: "08:00", EndTime: "10:00", DayType: "HOLIDAY", OverrideReason: "Tanggal merah", Project: "internal", Description: "Support"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].IsOverride || !items[1].IsOverride || items[1].OverrideReason != "Tanggal merah" {
		t.Fatalf("unexpected override state: %#v", items)
	}
}

func TestNormalizeOvertimeEntriesRejectsUnexplainedOverride(t *testing.T) {
	_, err := normalizeOvertimeEntries("2026-10", []model.OvertimeEntryInput{{WorkDate: "2026-10-12", StartTime: "08:00", EndTime: "10:00", DayType: "HOLIDAY", Project: "internal", Description: "Support"}})
	if err == nil {
		t.Fatal("expected unexplained override to fail")
	}
}

func TestNormalizeOvertimeEntriesUsesRoundedActualForPayment(t *testing.T) {
	for _, end := range []string{"18:29", "18:30"} {
		items, err := normalizeOvertimeEntries("2026-10", []model.OvertimeEntryInput{
			{WorkDate: "2026-10-12", StartTime: "17:00", EndTime: end, DayType: "WORKDAY", Project: "internal", Description: "Support"},
		})
		if err != nil {
			t.Fatal(err)
		}
		want := 60
		if end == "18:30" {
			want = 120
		}
		if items[0].ActualMinutes != want || items[0].WeightedMinutes != want {
			t.Fatalf("end=%s: got actual=%d weighted=%d, want %d", end, items[0].ActualMinutes, items[0].WeightedMinutes, want)
		}
	}
}

func TestNormalizeOvertimeEntriesRejectsRoundedZero(t *testing.T) {
	_, err := normalizeOvertimeEntries("2026-10", []model.OvertimeEntryInput{
		{WorkDate: "2026-10-12", StartTime: "17:00", EndTime: "17:29", DayType: "WORKDAY", Project: "internal", Description: "Support"},
	})
	if err == nil {
		t.Fatal("expected overtime rounded to zero to fail")
	}
}
