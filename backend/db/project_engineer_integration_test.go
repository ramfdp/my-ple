package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

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
	startOfWeek, _ := model.ReportingWeek(time.Now())
	database.Now = func() time.Time { return startOfWeek.AddDate(0, 0, 1) }
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
	if _, err := conn.Exec(`UPDATE reports SET completed_at = $2 WHERE id = $1`, created.ID, time.Now().AddDate(0, 0, -8)); err != nil {
		t.Fatal(err)
	}
	weekStart, _ := model.ReportingWeek(time.Now())
	if err := database.CleanupExpiredReports(context.Background(), weekStart); err != nil {
		t.Fatal(err)
	}
	var expired int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM reports WHERE id = $1`, created.ID).Scan(&expired); err != nil || expired != 1 {
		t.Fatalf("unarchived reports must be retained: %d, %v", expired, err)
	}
	if _, err := database.CreateReport(context.Background(), user.ID, input); err != ErrReportClosed {
		t.Fatalf("pending archive must block new working cycle: %v", err)
	}
	// Remove only this test fixture before testing current-period CRUD.
	if _, err := conn.Exec(`DELETE FROM reports WHERE id=$1`, created.ID); err != nil {
		t.Fatal(err)
	}
	created, err = database.CreateReport(context.Background(), user.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := database.UpdateReport(context.Background(), created.ID, model.WeeklyReportUpdateInput{PrimaryDivision: input.PrimaryDivision, WorkDivision: "Non-MK supervisi", Role: "PE", Work: "MPS PIT", Description: "updated"})
	if err != nil || updated.Role != "PE" || updated.Work != "MPS PIT" || updated.Description != "updated" {
		t.Fatalf("update = %#v, %v", updated, err)
	}
	start, next := model.ReportingWeek(time.Now())
	reports, err := database.ListReports(context.Background(), "Project Engineer", start, next)
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
	other := model.User{ID: NewID(), Name: "Other Report Test", Email: NewID() + "@example.test", Username: "other-" + NewID()[:8], RoleID: role.ID, PasswordHash: "test", EmploymentStatus: "Permanent Employee", IsActive: true}
	if _, err := database.CreateUser(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(`DELETE FROM reports WHERE user_id = $1`, other.ID)
		_, _ = conn.Exec(`DELETE FROM users WHERE id = $1`, other.ID)
	})
	if _, err := database.UpdateReport(context.Background(), created.ID, model.WeeklyReportUpdateInput{PrimaryDivision: input.PrimaryDivision, WorkDivision: "Non-MK supervisi", Role: "PE", Work: "MPS PIT", Description: "any user"}); err != nil {
		t.Fatalf("authenticated user update: %v", err)
	}
	owned, err := database.CreateReport(context.Background(), other.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.DeleteReport(context.Background(), owned.ID); err != nil {
		t.Fatalf("authenticated user delete: %v", err)
	}
	if err := database.DeleteReport(context.Background(), created.ID); err != nil {
		t.Fatalf("authenticated user delete: %v", err)
	}
	if err := database.DeleteUser(context.Background(), user.ID); err != nil {
		t.Fatalf("delete user after working reports removed: %v", err)
	}
	var remaining int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM reports WHERE user_id = $1`, user.ID).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("remaining reports = %d, %v", remaining, err)
	}
}
