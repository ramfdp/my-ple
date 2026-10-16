package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"my-ple-backend/model"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestRekapSource(t *testing.T) {
	sections, err := rekapSections()
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) != 13 {
		t.Fatalf("sections=%d", len(sections))
	}
	pairs, failures, successes := 0, 0, 0
	foundEmpty, blank101, blank102, duplicate4 := false, false, false, 0
	for _, section := range sections {
		if section.Project == "MPS SGR" && section.Group == "NONPENGAWASAN" && len(section.Rows) == 0 {
			foundEmpty = true
		}
		for _, pair := range section.Rows {
			pairs++
			if pair.Failure != nil {
				failures++
				if pair.Row == 101 && pair.Failure.Number == "" {
					blank101 = true
				}
				if pair.Row == 102 && pair.Failure.Number == "" {
					blank102 = true
				}
			}
			if pair.Success != nil {
				successes++
				if section.Project == "KAI - PMO IBP" && pair.Success.Number == "4" {
					duplicate4++
				}
			}
		}
	}
	if pairs != 34 || failures != 27 || successes != 21 || !foundEmpty || !blank101 || !blank102 || duplicate4 != 2 {
		t.Fatalf("pairs=%d failures=%d successes=%d empty=%v blanks=%v,%v duplicate4=%d", pairs, failures, successes, foundEmpty, blank101, blank102, duplicate4)
	}
}

func TestRekapMigrationPostgres(t *testing.T) {
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
	for i := 0; i < 2; i++ {
		if err = database.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var total int
	if err = conn.QueryRowContext(ctx, `SELECT count(*) FROM lessons_learned WHERE source_row IS NOT NULL`).Scan(&total); err != nil || total != 48 {
		t.Fatalf("imported=%d err=%v", total, err)
	}
	var id, version int
	if err = conn.QueryRowContext(ctx, `SELECT id,version FROM lessons_learned WHERE source_row=8 AND kind='failure'`).Scan(&id, &version); err != nil {
		t.Fatal(err)
	}
	var sourceProject, sourceNumber string
	if err = conn.QueryRowContext(ctx, `SELECT source_project,source_number FROM lessons_learned WHERE id=$1`, id).Scan(&sourceProject, &sourceNumber); err != nil || sourceProject != "MK Merapi-Banjarsari" || sourceNumber != "1" {
		t.Fatalf("provenance=%q/%q err=%v", sourceProject, sourceNumber, err)
	}
	var jobID, existingID int
	if err = conn.QueryRowContext(ctx, `SELECT id FROM job_context WHERE division_name='Project Engineer' ORDER BY id LIMIT 1`).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	if err = conn.QueryRowContext(ctx, `INSERT INTO lessons_learned(job_id,kind,title) VALUES($1,'failure','Existing MyHIS finding') RETURNING id`, jobID).Scan(&existingID); err != nil {
		t.Fatal(err)
	}
	lessons, err := database.LessonsLearned(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bySource := map[string]model.LessonLearned{}
	for _, l := range lessons {
		if l.SourceRow > 0 {
			bySource[fmt.Sprintf("%d:%s", l.SourceRow, l.Kind)] = l
		}
	}
	sections, err := rekapSections()
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range sections {
		for _, pair := range section.Rows {
			for _, item := range []struct {
				kind  string
				value *rekapFinding
			}{{"failure", pair.Failure}, {"success", pair.Success}} {
				if item.value == nil {
					continue
				}
				got, ok := bySource[fmt.Sprintf("%d:%s", pair.Row, item.kind)]
				want := item.value
				if !ok || got.SourceGroup != section.Group || got.SourceProject != section.Project || got.SourceNumber != want.Number || got.Title != want.Title || got.InternalFactor != want.InternalFactor || got.ExternalFactor != want.ExternalFactor || got.Impact != want.Impact || got.Action != want.Action || got.Notes != want.Notes {
					t.Fatalf("REKAP row %d %s differs after import", pair.Row, item.kind)
				}
			}
		}
	}
	for _, l := range lessons {
		if l.ID == id {
			l.Title = "Edited in app"
			updated, e := database.SaveLessonLearned(ctx, id, l)
			if e != nil || updated.Version != version+1 {
				t.Fatalf("update=%+v err=%v", updated, e)
			}
			if _, e = database.SaveLessonLearned(ctx, id, l); !errors.Is(e, model.ErrLessonConflict) {
				t.Fatalf("stale edit: %v", e)
			}
			break
		}
	}
	if err = database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var title string
	if err = conn.QueryRowContext(ctx, `SELECT title FROM lessons_learned WHERE id=$1`, id).Scan(&title); err != nil || title != "Edited in app" {
		t.Fatalf("rerun overwrote edit: %q %v", title, err)
	}
	if err = conn.QueryRowContext(ctx, `SELECT title FROM lessons_learned WHERE id=$1`, existingID).Scan(&title); err != nil || title != "Existing MyHIS finding" {
		t.Fatalf("existing finding lost: %q %v", title, err)
	}
}
