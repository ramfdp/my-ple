package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const weeklyFridayMigration = "2026-10-30-friday-report-period-v1"

type fridayScope struct {
	jobID              int64
	division, category string
}
type fridayReport struct {
	id, week           string
	division, category string
	jobID              sql.NullInt64
	number             int
}

func (r fridayReport) scope() fridayScope {
	if r.jobID.Valid {
		return fridayScope{jobID: r.jobID.Int64}
	}
	return fridayScope{division: r.division, category: r.category}
}

// Keep available numbers; collisions append above every existing/candidate number.
func fridayNumbers(candidates, existing []fridayReport) ([]int, error) {
	used := map[fridayScope]map[int]bool{}
	maximum := map[fridayScope]int{}
	for _, r := range append(append([]fridayReport{}, existing...), candidates...) {
		key := r.scope()
		if used[key] == nil {
			used[key] = map[int]bool{}
		}
		if r.number > maximum[key] {
			maximum[key] = r.number
		}
	}
	for _, r := range existing {
		used[r.scope()][r.number] = true
	}
	numbers := make([]int, len(candidates))
	for i, r := range candidates {
		key := r.scope()
		number := r.number
		if used[key][number] {
			if maximum[key] >= 2147483647 {
				return nil, fmt.Errorf("Friday report migration: report number exhausted")
			}
			maximum[key]++
			number = maximum[key]
		}
		numbers[i] = number
		used[key][number] = true
	}
	return numbers, nil
}

// Run inside startup's schema transaction, before any lifecycle worker starts.
func migrateWeeklyFriday(ctx context.Context, tx *sql.Tx, now time.Time) error {
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71260402);
 CREATE TABLE IF NOT EXISTS weekly_period_migrations(name TEXT PRIMARY KEY,applied_at TIMESTAMPTZ NOT NULL);
 CREATE TABLE IF NOT EXISTS weekly_period_migration_reports(
 migration_name TEXT NOT NULL,report_id UUID NOT NULL,original_week DATE NOT NULL,original_number INTEGER NOT NULL,
 migrated_week DATE NOT NULL,migrated_number INTEGER NOT NULL,PRIMARY KEY(migration_name,report_id));`); err != nil {
		return err
	}
	var done bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM weekly_period_migrations WHERE name=$1)`, weeklyFridayMigration).Scan(&done); err != nil {
		return err
	}
	if done {
		return nil
	}
	// Any archive status protects its period and IDs, including pending/failed snapshots.
	rows, err := tx.QueryContext(ctx, `SELECT r.id::text,r.report_week::text,r.primary_division,r.work_division,r.job_id,r.number
 FROM reports r WHERE report_week IN ('2026-10-26','2026-11-02') AND NOT EXISTS(
 SELECT 1 FROM weekly_report_archives a WHERE
 (a.primary_division=r.primary_division AND (a.week_start AT TIME ZONE 'Asia/Jakarta')::date=r.report_week)
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(a.snapshot) item WHERE item->>'id'=r.id::text))
 ORDER BY r.completed_at,r.id FOR UPDATE OF r`)
	if err != nil {
		return err
	}
	candidates := []fridayReport{}
	for rows.Next() {
		var r fridayReport
		if err = rows.Scan(&r.id, &r.week, &r.division, &r.category, &r.jobID, &r.number); err != nil {
			rows.Close()
			return err
		}
		candidates = append(candidates, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if len(candidates) > 0 {
		cutoff := time.Date(2026, 11, 7, 0, 0, 0, 0, time.FixedZone("WIB", 7*60*60))
		if !now.Before(cutoff) {
			return fmt.Errorf("Friday report migration: unarchived transition rows remain after 2026-11-07 00:00 WIB; stop workers and reconcile manually")
		}
		for _, r := range candidates {
			var archived bool
			if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM weekly_report_archives WHERE primary_division=$1 AND (week_start AT TIME ZONE 'Asia/Jakarta')::date='2026-10-30')`, r.division).Scan(&archived); err != nil {
				return err
			}
			if archived {
				return fmt.Errorf("Friday report migration: target 2026-10-30 already archived; reconcile manually")
			}
		}
		targetRows, err := tx.QueryContext(ctx, `SELECT id::text,report_week::text,primary_division,work_division,job_id,number FROM reports WHERE report_week='2026-10-30' FOR UPDATE`)
		if err != nil {
			return err
		}
		existing := []fridayReport{}
		for targetRows.Next() {
			var r fridayReport
			if err = targetRows.Scan(&r.id, &r.week, &r.division, &r.category, &r.jobID, &r.number); err != nil {
				targetRows.Close()
				return err
			}
			existing = append(existing, r)
		}
		err = targetRows.Err()
		targetRows.Close()
		if err != nil {
			return err
		}
		numbers, err := fridayNumbers(candidates, existing)
		if err != nil {
			return err
		}
		for i, r := range candidates {
			if _, err = tx.ExecContext(ctx, `INSERT INTO weekly_period_migration_reports VALUES($1,$2,$3,$4,'2026-10-30',$5)`, weeklyFridayMigration, r.id, r.week, r.number, numbers[i]); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `UPDATE reports SET report_week='2026-10-30',number=$2 WHERE id=$1`, r.id, numbers[i]); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO report_sequences(primary_division,work_division,week_start,job_id,last_number)
 SELECT COALESCE(j.division_name,r.primary_division),COALESCE(j.category_name,r.work_division),r.report_week,r.job_id,MAX(r.number)
 FROM reports r LEFT JOIN job_context j ON j.id=r.job_id WHERE r.report_week='2026-10-30' GROUP BY 1,2,3,4
 ON CONFLICT ON CONSTRAINT report_sequences_job_scope DO UPDATE SET last_number=GREATEST(report_sequences.last_number,EXCLUDED.last_number)`); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM report_sequences s WHERE s.week_start IN ('2026-10-26','2026-11-02')
 AND EXISTS(SELECT 1 FROM weekly_period_migration_reports m JOIN reports moved ON moved.id=m.report_id
 WHERE m.migration_name=$1 AND m.original_week=s.week_start AND
 ((s.job_id IS NOT NULL AND moved.job_id=s.job_id) OR (s.job_id IS NULL AND moved.job_id IS NULL AND moved.primary_division=s.primary_division AND moved.work_division=s.work_division)))
 AND NOT EXISTS(SELECT 1 FROM reports r WHERE r.report_week=s.week_start AND
 ((s.job_id IS NOT NULL AND r.job_id=s.job_id) OR (s.job_id IS NULL AND r.job_id IS NULL AND r.primary_division=s.primary_division AND r.work_division=s.work_division)))`, weeklyFridayMigration); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO weekly_period_migrations(name,applied_at) VALUES($1,$2)`, weeklyFridayMigration, now)
	return err
}
