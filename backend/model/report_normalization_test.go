package model

import "testing"

func TestNormalizeWeeklyRoleCategory(t *testing.T) {
	for legacy, wantCategory := range map[string]string{"QA 1": "QA1", "QA 2": "QA2"} {
		role, category := NormalizeWeeklyRoleCategory(legacy, legacy)
		if role != "QA" || category != wantCategory {
			t.Fatalf("%s normalized to role=%q category=%q", legacy, role, category)
		}
	}
	report := WeeklyReport{Role: "QA 1", WorkDivision: "QA 1", Work: "Internal"}
	NormalizeWeeklyReport(&report)
	if report.Role != "QA" || report.WorkDivision != "QA1" || report.Work != "Internal" {
		t.Fatal(report)
	}
}
