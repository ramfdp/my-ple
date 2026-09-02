package controller

import (
	"testing"

	"my-ple-backend/model"
)

func TestProjectEngineerRoleWorkValidation(t *testing.T) {
	base := model.WeeklyReportInput{PrimaryDivision: "Project Engineer", WorkDivision: "JAKG", Description: "completed"}
	base.Role, base.Work = "CIHPK", "JAKG Supervising Consultant (SPMK Date: March 12, 2026)"
	if !validReportInput(base) {
		t.Fatal("valid CIHPK work was rejected")
	}
	base.Work = "MPS PIT"
	if validReportInput(base) {
		t.Fatal("CIHPK accepted a general work option")
	}
	base.Role, base.WorkDivision, base.Work = "PE", "Non-MK supervisi", "MPS PIT"
	if !validReportInput(base) {
		t.Fatal("PE work was rejected")
	}
}
