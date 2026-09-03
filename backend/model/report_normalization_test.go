package model

import "testing"

func TestNormalizeWeeklyRoleCategory(t *testing.T) {
	for _, legacy := range []string{"QA 1", "QA 2"} {
		role, category := NormalizeWeeklyRoleCategory(legacy, legacy)
		if role != "QA" || category != "QA" {
			t.Fatalf("%s normalized to role=%q category=%q", legacy, role, category)
		}
	}
	if !ValidWeeklyReport("Project Engineer", "QA", "QA", "Internal", "selesai") {
		t.Fatal("canonical QA input is not valid")
	}
}
