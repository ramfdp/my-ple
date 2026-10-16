package db

import (
	"context"
	"database/sql"
	"my-ple-backend/model"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHermesQuery(t *testing.T) {
	if got := hermesQuery("Bagaimana cara mengatasi masalah sambungan kabel sambungan?"); got != "mengatasi sambungan kabel" {
		t.Fatalf("terms=%q", got)
	}
	if got := hermesQuery("how can this problem be solved?"); got != "solved" {
		t.Fatalf("terms=%q", got)
	}
}

func TestHermesSearchPostgres(t *testing.T) {
	dsn := os.Getenv("LESSONS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("LESSONS_TEST_DATABASE_URL absent")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Hostname() != "127.0.0.1" || !strings.Contains(u.Path, "lessons_test") {
		t.Fatal("isolated local lessons_test database required")
	}
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	database := NewDatabase(conn)
	ctx := context.Background()
	if err := database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	user := NewID()
	finance := NewID()
	admin := NewID()
	task := NewID()
	archive := NewID()
	report := NewID()
	_, err = conn.ExecContext(ctx, `INSERT INTO users(id,name,email,role_id,username,password_hash) VALUES($1,'Chat fixture','chat-fixture@example.test',(SELECT id FROM roles WHERE name='Project Engineer'),'chat-fixture','x')`, user)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.ExecContext(ctx, `INSERT INTO users(id,name,email,role_id,username,password_hash) VALUES($1,'Finance fixture','finance-fixture@example.test',(SELECT id FROM roles WHERE name='Finance'),'finance-fixture','x')`, finance)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.ExecContext(ctx, `INSERT INTO users(id,name,email,role_id,username,password_hash) VALUES($1,'Admin fixture','admin-fixture@example.test',(SELECT id FROM roles WHERE name='Admin'),'admin-fixture','x')`, admin)
	if err != nil {
		t.Fatal(err)
	}
	var division int
	if err := conn.QueryRowContext(ctx, `SELECT id FROM divisions WHERE name='Project Engineer'`).Scan(&division); err != nil {
		t.Fatal(err)
	}
	_, err = conn.ExecContext(ctx, `INSERT INTO work_items(id,division_id,title,description,created_by,archived_at) VALUES($1,$2,'Zirconiumfix task','Repair cable', $3,now())`, task, division, user)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.ExecContext(ctx, `INSERT INTO reports(id,user_id,primary_division,work_division,report_role,work,number,description,completed_at) VALUES($1,$2,'Project Engineer','MK supervisi','Project Engineer','Zirconiumfix report',1,'Repair cable',$3)`, report, user, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.ExecContext(ctx, `INSERT INTO weekly_report_archives(id,primary_division,week_start,week_end,snapshot,status) VALUES($1,'Project Engineer','2025-01-06T00:00:00+07:00','2025-01-13T00:00:00+07:00',$2,'finalized')`, archive, `[{"id":"archived-report","work":"Zirconiumfix archive","description":"Repair cable","workDivision":"MK supervisi"}]`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.ExecContext(ctx, `INSERT INTO lessons_learned(kind,title,action) VALUES('failure','Zirconiumfix lesson','Repair cable')`)
	if err != nil {
		t.Fatal(err)
	}
	claim := model.Claims{ID: user, RoleName: "Project Engineer"}
	sources, err := database.HermesSearch(ctx, claim, "Zirconiumfix")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	archivedReport := false
	currentReport := false
	for _, source := range sources {
		seen[source.Kind] = true
		if source.Kind == "lesson" && !strings.HasPrefix(source.URL, "/lessons-learned#lesson-") {
			t.Fatalf("lesson link=%q", source.URL)
		}
		if source.Kind == "task" && !strings.Contains(source.URL, "archived=1") {
			t.Fatalf("archived link=%q", source.URL)
		}
		if strings.HasPrefix(source.ID, "archive:") {
			archivedReport = true
			if !strings.Contains(source.URL, "/weekly-progress/history?archive=") {
				t.Fatalf("archive link=%q", source.URL)
			}
		}
		if source.ID == "report:"+report {
			currentReport = true
			if !strings.Contains(source.URL, "/weekly-progress?division=Project%20Engineer#report-") {
				t.Fatalf("current report link=%q", source.URL)
			}
		}
	}
	for _, kind := range []string{"lesson", "task", "report"} {
		if !seen[kind] {
			t.Fatalf("missing %s: %+v", kind, sources)
		}
	}
	if !archivedReport || !currentReport {
		t.Fatalf("missing current/archive reports: %+v", sources)
	}
	claim.ID = admin
	claim.RoleName = "Admin"
	sources, err = database.HermesSearch(ctx, claim, "Zirconiumfix")
	if err != nil || len(sources) < 4 {
		t.Fatalf("admin results=%+v err=%v", sources, err)
	}
	claim.ID = finance
	claim.RoleName = "Finance"
	sources, err = database.HermesSearch(ctx, claim, "Zirconiumfix")
	if err != nil || len(sources) != 0 {
		t.Fatalf("cross-role results=%+v err=%v", sources, err)
	}
	sources, err = database.HermesSearch(ctx, claim, "nomatchwordqzx")
	if err != nil || len(sources) != 0 {
		t.Fatalf("no-match results=%+v err=%v", sources, err)
	}
}
