package db

import (
	"my-ple-backend/model"
	"testing"
)

func TestReportDisplayNumbersAfterDeletion(t *testing.T) {
	reports := []model.WeeklyReport{{ID: "b", Role: "PE", Work: "A", Number: 2}, {ID: "d", Role: "PE", Work: "A", Number: 4}, {ID: "e", Role: "PE", Work: "B", Number: 8}}
	SortReports(reports)
	for i, want := range []int{1, 2, 1} {
		if reports[i].DisplayNumber != want {
			t.Fatalf("row %d: %+v", i, reports[i])
		}
	}
	if reports[0].ID != "b" || reports[0].Number != 2 {
		t.Fatal("identity or stored sequence changed")
	}
}
