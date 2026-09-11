package controller

import (
	"testing"

	"my-ple-backend/model"
)

func TestWeeklyReportsStayInsideUsersDivision(t *testing.T) {
	pe := model.Claims{RoleName: "Project Engineer"}
	if !canViewWeekly(pe, "Project Engineer") || canViewWeekly(pe, "HR") {
		t.Fatal("Project Engineer weekly access escaped its division")
	}
	if !canViewWeekly(model.Claims{RoleName: "Finance"}, "Keuangan") {
		t.Fatal("Finance did not map to Keuangan")
	}
	if !canViewWeekly(model.Claims{RoleName: "Admin"}, "HR") {
		t.Fatal("Admin must retain cross-division access")
	}
}
