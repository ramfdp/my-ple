package controller

import (
	"context"
	"my-ple-backend/db"
	"testing"
	"time"

	"my-ple-backend/model"
)

func reportValidator(t *testing.T) func(model.WeeklyReportInput) bool {
	database := archiveTestDB(t)
	ctx := context.Background()
	start, _ := model.ReportingWeek(time.Now())
	database.Now = func() time.Time { return start.AddDate(0, 0, 2) }
	role, _ := database.FindRoleByName(ctx, "Admin")
	user, err := database.CreateUser(ctx, model.User{ID: db.NewID(), Name: "Validation", Username: "validation", Email: "validation@example.test", RoleID: role.ID, PasswordHash: "test", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	return func(input model.WeeklyReportInput) bool {
		_, err := database.CreateReport(ctx, user.ID, input)
		return err == nil
	}
}
func TestProjectEngineerRoleWorkValidation(t *testing.T) {
	validReportInput := reportValidator(t)
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

func TestProjectEngineerIndonesianWorkValidation(t *testing.T) {
	validReportInput := reportValidator(t)
	for _, input := range []model.WeeklyReportInput{
		{PrimaryDivision: "Project Engineer", WorkDivision: "MK supervisi", Role: "PE", Work: "Penataan Kawasan Stasiun Jember", Description: "selesai"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "QA", Role: "QA", Work: "SOP Tata Kelola", Description: "selesai"},
	} {
		if !validReportInput(input) {
			t.Fatalf("valid Indonesian work was rejected: %s", input.Work)
		}
	}
}
