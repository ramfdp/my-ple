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
