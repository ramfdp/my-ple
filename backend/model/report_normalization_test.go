package model

import "testing"

func TestNormalizeWeeklyRoleCategory(t *testing.T) {
	for legacy, wantCategory := range map[string]string{"QA 1": "QA1", "QA 2": "QA2"} {
		role, category := NormalizeWeeklyRoleCategory(legacy, legacy)
		if role != "QA" || category != wantCategory {
			t.Fatalf("%s normalized to role=%q category=%q", legacy, role, category)
		}
	}
	if !ValidWeeklyReport("Project Engineer", "QA1", "QA", "Internal", "selesai") {
		t.Fatal("canonical QA input is not valid")
	}
}
