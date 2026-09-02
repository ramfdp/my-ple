package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"my-ple-backend/model"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestProjectEngineerReportPostgres(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is required")
	}
	conn, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	database := NewDatabase(conn)
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	role, err := database.FindRoleByName(context.Background(), "Project Engineer")
	if err != nil {
		t.Fatal(err)
	}
	user := model.User{ID: NewID(), Name: "PE Report Test", Email: NewID() + "@example.test", Username: "pe-" + NewID()[:8], RoleID: role.ID, PasswordHash: "test", EmploymentStatus: "Permanent Employee", IsActive: true}
	if _, err := database.CreateUser(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(`DELETE FROM reports WHERE user_id = $1`, user.ID)
		_, _ = conn.Exec(`DELETE FROM users WHERE id = $1`, user.ID)
	})
	input := model.WeeklyReportInput{PrimaryDivision: "Project Engineer", WorkDivision: "JAKG", Role: "CIHPK", Work: "JAKG Supervising Consultant (SPMK Date: March 12, 2026)", Description: "created"}
	created, err := database.CreateReport(context.Background(), user.ID, input)
	if err != nil || created.Role != input.Role || created.Work != input.Work {
		t.Fatalf("create = %#v, %v", created, err)
	}
	updated, err := database.UpdateReport(context.Background(), created.ID, user.ID, model.WeeklyReportUpdateInput{PrimaryDivision: input.PrimaryDivision, WorkDivision: "Non-MK supervisi", Role: "PE", Work: "MPS PIT", Description: "updated"})
	if err != nil || updated.Role != "PE" || updated.Work != "MPS PIT" || updated.Description != "updated" {
		t.Fatalf("update = %#v, %v", updated, err)
	}
	reports, err := database.ListReports(context.Background(), "Project Engineer")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, report := range reports {
		found = found || report.ID == created.ID && report.Role == "PE" && report.Work == "MPS PIT"
	}
	if !found {
		t.Fatal("updated role and work were not loaded")
	}
	if err := database.DeleteUser(context.Background(), user.ID); err != nil {
		t.Fatalf("delete user with reports: %v", err)
	}
	var remaining int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM reports WHERE user_id = $1`, user.ID).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("remaining reports = %d, %v", remaining, err)
	}
}
