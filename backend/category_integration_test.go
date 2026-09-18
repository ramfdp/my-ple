package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"my-ple-backend/db"
	"my-ple-backend/model"
	"strings"
	"testing"
)

func TestCreateJobCategoryIntegration(t *testing.T) {
	database := checklistTestDB(t)
	ctx := context.Background()
	var division, role int
	if err := database.Conn.QueryRow(`SELECT division_id,id FROM report_roles WHERE name='CIHPK' LIMIT 1`).Scan(&division, &role); err != nil {
		t.Fatal(err)
	}
	var otherDivision int
	if err := database.Conn.QueryRow(`SELECT id FROM divisions WHERE id<>$1 LIMIT 1`, division).Scan(&otherDivision); err != nil {
		t.Fatal(err)
	}
	name := "Category test " + db.NewID()
	category, err := database.CreateJobCategory(ctx, model.JobMaster{DivisionID: &division, Name: "  " + name + "  "})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Conn.Exec(`DELETE FROM job_categories WHERE id=$1`, category.ID) })
	if category.Name != name || category.ID <= 0 || category.RoleID != nil {
		t.Fatalf("unexpected category: %+v", category)
	}
	_, err = database.CreateJobCategory(ctx, model.JobMaster{DivisionID: &division, Name: strings.ToUpper(name)})
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "23505" {
		t.Fatalf("duplicate accepted: %v", err)
	}
	for _, input := range []model.JobMaster{
		{DivisionID: &division, Name: " "}, {DivisionID: &division, Name: strings.Repeat("a", 101)},
		{Name: name}, {DivisionID: &division, RoleID: &role, Name: name},
		{DivisionID: &otherDivision, RoleID: &role, Name: name},
	} {
		if _, err := database.CreateJobCategory(ctx, input); !errors.Is(err, db.ErrInvalidReport) {
			t.Fatalf("invalid accepted: %+v, %v", input, err)
		}
	}
	masters, err := database.JobMasters(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range masters["categories"] {
		if item.ID == category.ID && item.Name == name {
			return
		}
	}
	t.Fatal("new category missing from masters")
}
