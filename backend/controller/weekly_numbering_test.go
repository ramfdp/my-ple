package controller

import (
	"context"
	"fmt"
	"my-ple-backend/db"
	"my-ple-backend/model"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestConcurrentWeeklyJobNumbersPostgres(t *testing.T) {
	database := archiveTestDB(t)
	ctx := context.Background()
	start, _ := model.ReportingWeek(time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC))
	database.Now = func() time.Time { return start.AddDate(0, 0, 2) }
	role, _ := database.FindRoleByName(ctx, "Project Engineer")
	users := []string{}
	for i := 0; i < 3; i++ {
		u, err := database.CreateUser(ctx, model.User{ID: db.NewID(), Name: "Concurrent", Username: fmt.Sprint("concurrent", i), Email: fmt.Sprintf("concurrent%d@example.test", i), RoleID: role.ID, PasswordHash: "test", IsActive: true})
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, u.ID)
	}
	jobs, err := database.ListJobs(ctx, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	scopes := []model.Job{}
	for _, j := range jobs {
		if j.Role == "PE" && j.Category == "MK supervisi" && len(scopes) < 2 {
			scopes = append(scopes, j)
		}
	}
	if len(scopes) != 2 {
		t.Fatal("missing two jobs in the same category")
	}
	input := func(j model.Job) model.WeeklyReportInput {
		return model.WeeklyReportInput{JobID: &j.ID, PrimaryDivision: j.Division, Role: j.Role, WorkDivision: j.Category, Description: "Concurrent entry"}
	}
	const count = 12
	type outcome struct {
		report model.WeeklyReport
		err    error
	}
	run := func(database *db.Database) {
		t.Helper()
		results := make(chan outcome, 2*count)
		gate := make(chan struct{})
		var wg sync.WaitGroup
		for _, j := range scopes {
			for i := 0; i < count; i++ {
				wg.Add(1)
				go func(j model.Job, i int) {
					defer wg.Done()
					<-gate
					r, e := database.CreateReport(ctx, users[i%len(users)], input(j))
					results <- outcome{r, e}
				}(j, i)
			}
		}
		close(gate)
		wg.Wait()
		close(results)
		numbers := map[int][]int{}
		for result := range results {
			if result.err != nil {
				t.Fatal(result.err)
			}
			numbers[*result.report.JobID] = append(numbers[*result.report.JobID], result.report.Number)
		}
		for _, j := range scopes {
			sort.Ints(numbers[j.ID])
			if len(numbers[j.ID]) != count {
				t.Fatal(numbers)
			}
			for i, n := range numbers[j.ID] {
				if n != i+1 {
					t.Fatalf("job %d: got %v, want independent 1..%d", j.ID, numbers[j.ID], count)
				}
			}
		}
	}
	run(database)
	// A failing report INSERT must roll back the increment, not consume a number.
	if _, err = database.CreateReport(ctx, db.NewID(), input(scopes[0])); err == nil {
		t.Fatal("invalid user accepted")
	}
	next, err := database.CreateReport(ctx, users[0], input(scopes[0]))
	if err != nil || next.Number != count+1 {
		t.Fatalf("rollback: %+v %v", next, err)
	}
	// Moving a report to another job in the SAME category allocates in that job.
	if _, err = database.CreateReport(ctx, users[0], input(scopes[1])); err != nil {
		t.Fatal(err)
	}
	moved, err := database.UpdateReport(ctx, next.ID, model.WeeklyReportUpdateInput{JobID: &scopes[1].ID, PrimaryDivision: scopes[1].Division, Role: scopes[1].Role, WorkDivision: scopes[1].Category, Description: "Moved"})
	if err != nil || moved.Number != count+2 {
		t.Fatalf("move: %+v %v", moved, err)
	}
	unchanged, err := database.UpdateReport(ctx, moved.ID, model.WeeklyReportUpdateInput{JobID: moved.JobID, PrimaryDivision: moved.PrimaryDivision, Role: moved.Role, WorkDivision: moved.WorkDivision, Description: "Edited"})
	if err != nil || unchanged.Number != moved.Number {
		t.Fatalf("description edit renumbered: %v", err)
	}
	if err = database.DeleteReport(ctx, moved.ID); err != nil {
		t.Fatal(err)
	}
	afterDelete, err := database.CreateReport(ctx, users[0], input(scopes[1]))
	if err != nil || afterDelete.Number != count+3 {
		t.Fatalf("deleted number reused: %+v %v", afterDelete, err)
	}
	if _, err = database.Conn.Exec(`INSERT INTO reports(id,user_id,primary_division,work_division,report_role,job_id,work,number,description,completed_at)
 SELECT $2,user_id,primary_division,work_division,report_role,job_id,work,number,description,completed_at FROM reports WHERE id=$1`, afterDelete.ID, db.NewID()); !db.UniqueViolation(err) {
		t.Fatalf("duplicate report number allowed: %v", err)
	}
	if _, err = database.Conn.Exec(`INSERT INTO report_sequences SELECT * FROM report_sequences WHERE job_id=$1 LIMIT 1`, scopes[0].ID); !db.UniqueViolation(err) {
		t.Fatalf("duplicate counter scope allowed: %v", err)
	}
	// Exercise the existing verified-archive reset contract, without changing it.
	archive, err := database.SnapshotWeek(ctx, db.DueWeek{Division: scopes[0].Division, Start: start})
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Conn.Exec(`UPDATE weekly_report_archives SET status='finalized',pdf_sha256='verified-test-pdf' WHERE id=$1`, archive.ID)
	if err != nil {
		t.Fatal(err)
	}
	archive.Status = "finalized"
	archive.SHA256 = "verified-test-pdf"
	if _, err = database.ResetArchivedWeek(ctx, archive, start.AddDate(0, 0, 8)); err != nil {
		t.Fatal(err)
	}
	nextWeek := db.NewDatabase(database.Conn)
	nextWeek.Now = func() time.Time { return start.AddDate(0, 0, 9) }
	// Retaining an older counter must not influence the next week's numbers.
	if _, err = database.Conn.Exec(`INSERT INTO report_sequences(primary_division,work_division,week_start,job_id,last_number) VALUES($1,$2,$3,$4,99)`, scopes[0].Division, scopes[0].Category, start.Format("2006-01-02"), scopes[0].ID); err != nil {
		t.Fatal(err)
	}
	run(nextWeek)
	// Reapplying schema must not restore the old category-wide unique constraint.
	if err = database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, j := range scopes {
		r, e := nextWeek.CreateReport(ctx, users[0], input(j))
		if e != nil || r.Number != count+1 {
			t.Fatalf("repeat migration scope: %+v %v", r, e)
		}
	}
}

func TestWeeklyNumberingMigrationPreservesReportsPostgres(t *testing.T) {
	database := archiveTestDB(t)
	ctx := context.Background()
	start, _ := model.ReportingWeek(time.Now())
	database.Now = func() time.Time { return start.AddDate(0, 0, 2) }
	role, _ := database.FindRoleByName(ctx, "Project Engineer")
	u, err := database.CreateUser(ctx, model.User{ID: db.NewID(), Name: "Legacy numbering", Username: "legacy-numbering", Email: "legacy-numbering@example.test", RoleID: role.ID, PasswordHash: "test", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := database.ListJobs(ctx, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var job model.Job
	for _, j := range jobs {
		if j.Role == "PE" && j.Category == "MK supervisi" {
			job = j
			break
		}
	}
	// Reconstruct the previous sequence/constraint shape in this test's schema.
	_, err = database.Conn.Exec(`ALTER TABLE report_sequences DROP COLUMN job_id;
 ALTER TABLE report_sequences ADD CONSTRAINT report_sequences_week_pk PRIMARY KEY(primary_division,work_division,week_start);
 DROP INDEX reports_job_week_number_key; DROP INDEX reports_legacy_week_number_key;
 ALTER TABLE reports ADD CONSTRAINT reports_week_number_key UNIQUE(primary_division,work_division,report_week,number);`)
	if err != nil {
		t.Fatal(err)
	}
	id := db.NewID()
	_, err = database.Conn.Exec(`INSERT INTO reports(id,user_id,primary_division,work_division,report_role,work,job_id,number,description,completed_at) VALUES($1,$2,$3,$4,$5,$6,$7,9,'preserve me',$8)`, id, u.ID, job.Division, job.Category, job.Role, job.Name, job.ID, database.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Conn.Exec(`INSERT INTO report_sequences(primary_division,work_division,week_start,last_number) VALUES($1,$2,$3,25)`, job.Division, job.Category, start.Format("2006-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	before := ""
	if err = database.Conn.QueryRow(`SELECT row_to_json(r)::text FROM reports r WHERE id=$1`, id).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	after := ""
	if err = database.Conn.QueryRow(`SELECT row_to_json(r)::text FROM reports r WHERE id=$1`, id).Scan(&after); err != nil || after != before {
		t.Fatalf("historical row changed: %v", err)
	}
	counter := 0
	if err = database.Conn.QueryRow(`SELECT last_number FROM report_sequences WHERE job_id IS NULL AND primary_division=$1 AND work_division=$2`, job.Division, job.Category).Scan(&counter); err != nil || counter != 25 {
		t.Fatalf("legacy counter changed: %d %v", counter, err)
	}
	created, err := database.CreateReport(ctx, u.ID, model.WeeklyReportInput{JobID: &job.ID, PrimaryDivision: job.Division, WorkDivision: job.Category, Role: job.Role, Description: "continue"})
	if err != nil || created.Number != 10 {
		t.Fatalf("continue legacy job: %+v %v", created, err)
	}
}
