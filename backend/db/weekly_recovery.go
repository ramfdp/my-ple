package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

const weeklyRecoveryName = "2026-10-31-missed-week-recovery-v1"
const weeklyRecoverySource = "2026-10-30"
const weeklyRecoveryTarget = "2026-10-23"

type WeeklyRecoveryDivision struct {
	Division            string `json:"division"`
	Count               int    `json:"count"`
	ExistingTargetCount int    `json:"existingTargetCount"`
}
type WeeklyRecoveryPreview struct {
	Count       int                      `json:"count"`
	Fingerprint string                   `json:"fingerprint"`
	Applied     bool                     `json:"applied"`
	Divisions   []WeeklyRecoveryDivision `json:"divisions"`
}

func recoveryRecorded(ctx context.Context, tx *sql.Tx) (WeeklyRecoveryPreview, bool, error) {
	var p WeeklyRecoveryPreview
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT to_regclass('weekly_report_recovery_runs') IS NOT NULL`).Scan(&exists); err != nil {
		return p, false, err
	}
	if !exists {
		return p, false, nil
	}
	var data []byte
	err := tx.QueryRowContext(ctx, `SELECT preview FROM weekly_report_recovery_runs WHERE name=$1`, weeklyRecoveryName).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return p, false, nil
	}
	if err != nil {
		return p, false, err
	}
	if err = json.Unmarshal(data, &p); err != nil {
		return p, false, err
	}
	p.Applied = true
	return p, true, nil
}

func recoveryPreview(ctx context.Context, tx *sql.Tx) (WeeklyRecoveryPreview, []fridayReport, error) {
	p := WeeklyRecoveryPreview{Divisions: []WeeklyRecoveryDivision{}}
	rows, err := tx.QueryContext(ctx, `SELECT r.id::text,r.report_week::text,r.primary_division,r.work_division,r.job_id,r.number,to_jsonb(r)::text FROM reports r WHERE report_week=$1::date ORDER BY completed_at,id`, weeklyRecoverySource)
	if err != nil {
		return p, nil, err
	}
	candidates := []fridayReport{}
	hash := sha256.New()
	counts := map[string]int{}
	// Full row JSON covers content, timestamps, identity, ownership, numbering and scope.
	encoder := json.NewEncoder(hash)
	for rows.Next() {
		var r fridayReport
		var raw string
		if err = rows.Scan(&r.id, &r.week, &r.division, &r.category, &r.jobID, &r.number, &raw); err != nil {
			rows.Close()
			return p, nil, err
		}
		if err = encoder.Encode(raw); err != nil {
			rows.Close()
			return p, nil, err
		}
		candidates = append(candidates, r)
		counts[r.division]++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return p, nil, err
	}
	p.Count = len(candidates)
	p.Fingerprint = fmt.Sprintf("%x", hash.Sum(nil))
	for division, count := range counts {
		p.Divisions = append(p.Divisions, WeeklyRecoveryDivision{Division: division, Count: count})
	}
	sort.Slice(p.Divisions, func(i, j int) bool { return p.Divisions[i].Division < p.Divisions[j].Division })
	for i := range p.Divisions {
		d := &p.Divisions[i]
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM reports WHERE primary_division=$1 AND report_week=$2::date`, d.Division, weeklyRecoveryTarget).Scan(&d.ExistingTargetCount); err != nil {
			return p, nil, err
		}
		if d.ExistingTargetCount > 0 {
			return p, nil, fmt.Errorf("weekly recovery: division %q already has target reports for %s; reconcile manually", d.Division, weeklyRecoveryTarget)
		}
		var archived bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM weekly_report_archives WHERE primary_division=$1 AND (week_start AT TIME ZONE 'Asia/Jakarta')::date IN ($2::date,$3::date))`, d.Division, weeklyRecoverySource, weeklyRecoveryTarget).Scan(&archived); err != nil {
			return p, nil, err
		}
		if archived {
			return p, nil, fmt.Errorf("weekly recovery: division %q has a source/target archive; immutable archives cannot be merged", d.Division)
		}
	}
	var represented bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM reports r JOIN weekly_report_archives a ON EXISTS(SELECT 1 FROM jsonb_array_elements(a.snapshot) item WHERE item->>'id'=r.id::text) WHERE r.report_week=$1::date)`, weeklyRecoverySource).Scan(&represented); err != nil {
		return p, nil, err
	}
	if represented {
		return p, nil, errors.New("weekly recovery: a source report is already represented in an archive snapshot; reconcile manually")
	}
	return p, candidates, nil
}

// Preview is read-only and never runs schema migrations or creates recovery tables.
func (db *Database) PreviewWeeklyRecovery(ctx context.Context) (WeeklyRecoveryPreview, error) {
	tx, err := db.Conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return WeeklyRecoveryPreview{}, err
	}
	defer tx.Rollback()
	p, done, err := recoveryRecorded(ctx, tx)
	if err != nil {
		return p, err
	}
	if done {
		return p, tx.Commit()
	}
	p, _, err = recoveryPreview(ctx, tx)
	if err != nil {
		return p, err
	}
	return p, tx.Commit()
}

func recoveryCohorts(p WeeklyRecoveryPreview) []DueWeek {
	start := time.Date(2026, 10, 23, 0, 0, 0, 0, time.FixedZone("WIB", 7*60*60))
	weeks := make([]DueWeek, 0, len(p.Divisions))
	for _, d := range p.Divisions {
		weeks = append(weeks, DueWeek{Division: d.Division, Start: start})
	}
	return weeks
}

// Caller must hold lifecycle session lock 71260401 until these cohorts finish.
// All backend writers must be stopped; retries select the recorded cohort only.
func (db *Database) ApplyWeeklyRecovery(ctx context.Context, expectedCount int, fingerprint string) ([]DueWeek, error) {
	if expectedCount <= 0 || len(fingerprint) != 64 {
		return nil, errors.New("weekly recovery: positive expected count and exact preview fingerprint required")
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71260402); LOCK TABLE reports IN SHARE ROW EXCLUSIVE MODE; LOCK TABLE weekly_report_archives IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return nil, err
	}
	p, done, err := recoveryRecorded(ctx, tx)
	if err != nil {
		return nil, err
	}
	var candidates []fridayReport
	if !done {
		p, candidates, err = recoveryPreview(ctx, tx)
		if err != nil {
			return nil, err
		}
	}
	if p.Count != expectedCount || p.Fingerprint != fingerprint {
		return nil, errors.New("weekly recovery: preview changed; review a fresh preview before applying")
	}
	if done {
		if err = validateRecovery(ctx, tx, p, false); err != nil {
			return nil, err
		}
		return recoveryCohorts(p), tx.Commit()
	}
	numbers, err := fridayNumbers(candidates, nil)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS weekly_report_recovery_runs(name TEXT PRIMARY KEY,preview JSONB NOT NULL,applied_at TIMESTAMPTZ NOT NULL);
 CREATE TABLE IF NOT EXISTS weekly_report_recovery_rows(recovery_name TEXT NOT NULL REFERENCES weekly_report_recovery_runs(name),report_id UUID NOT NULL,primary_division TEXT NOT NULL,original_week DATE NOT NULL,original_number INTEGER NOT NULL,recovered_week DATE NOT NULL,recovered_number INTEGER NOT NULL,PRIMARY KEY(recovery_name,report_id))`); err != nil {
		return nil, err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if db.Now != nil {
		now = db.Now()
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO weekly_report_recovery_runs(name,preview,applied_at) VALUES($1,$2,$3)`, weeklyRecoveryName, data, now); err != nil {
		return nil, err
	}
	for i, r := range candidates {
		if _, err = tx.ExecContext(ctx, `INSERT INTO weekly_report_recovery_rows VALUES($1,$2,$3,$4,$5,$6,$7)`, weeklyRecoveryName, r.id, r.division, r.week, r.number, weeklyRecoveryTarget, numbers[i]); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE reports SET report_week=$2::date,number=$3 WHERE id=$1`, r.id, weeklyRecoveryTarget, numbers[i]); err != nil {
			return nil, err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO report_sequences(primary_division,work_division,week_start,job_id,last_number)
 SELECT COALESCE(j.division_name,r.primary_division),COALESCE(j.category_name,r.work_division),r.report_week,r.job_id,MAX(r.number)
 FROM reports r LEFT JOIN job_context j ON j.id=r.job_id JOIN weekly_report_recovery_rows m ON m.report_id=r.id AND m.recovery_name=$1 GROUP BY 1,2,3,4
 ON CONFLICT ON CONSTRAINT report_sequences_job_scope DO UPDATE SET last_number=GREATEST(report_sequences.last_number,EXCLUDED.last_number)`, weeklyRecoveryName); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM report_sequences s WHERE week_start=$1::date
 AND EXISTS(SELECT 1 FROM weekly_report_recovery_rows m WHERE m.recovery_name=$2 AND m.primary_division=s.primary_division)
 AND NOT EXISTS(SELECT 1 FROM reports r WHERE r.report_week=s.week_start AND ((s.job_id IS NOT NULL AND r.job_id=s.job_id) OR (s.job_id IS NULL AND r.job_id IS NULL AND r.primary_division=s.primary_division AND r.work_division=s.work_division)))`, weeklyRecoverySource, weeklyRecoveryName); err != nil {
		return nil, err
	}
	return recoveryCohorts(p), tx.Commit()
}

// ValidateWeeklyRecovery checks only the fixed recovery ledger, never a new sweep.
func (db *Database) ValidateWeeklyRecovery(ctx context.Context) error {
	return db.verifyWeeklyRecovery(ctx, false)
}

// VerifyWeeklyRecovery is the completion gate after verified PDF finalization/reset.
func (db *Database) VerifyWeeklyRecovery(ctx context.Context) error {
	return db.verifyWeeklyRecovery(ctx, true)
}
func (db *Database) verifyWeeklyRecovery(ctx context.Context, complete bool) error {
	tx, err := db.Conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	p, done, err := recoveryRecorded(ctx, tx)
	if err != nil {
		return err
	}
	if !done {
		return errors.New("weekly recovery: no applied recovery ledger")
	}
	if err = validateRecovery(ctx, tx, p, complete); err != nil {
		return err
	}
	return tx.Commit()
}
func validateRecovery(ctx context.Context, tx *sql.Tx, p WeeklyRecoveryPreview, complete bool) error {
	for _, d := range p.Divisions {
		rows, err := tx.QueryContext(ctx, `SELECT report_id::text FROM weekly_report_recovery_rows WHERE recovery_name=$1 AND primary_division=$2`, weeklyRecoveryName, d.Division)
		if err != nil {
			return err
		}
		ids := map[string]bool{}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids[id] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(ids) != d.Count {
			return errors.New("weekly recovery: ledger count mismatch")
		}
		var polluted bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM reports r WHERE r.primary_division=$1 AND r.report_week=$2::date AND NOT EXISTS(SELECT 1 FROM weekly_report_recovery_rows m WHERE m.recovery_name=$3 AND m.report_id=r.id AND m.primary_division=r.primary_division))`, d.Division, weeklyRecoveryTarget, weeklyRecoveryName).Scan(&polluted); err != nil {
			return err
		}
		if polluted {
			return errors.New("weekly recovery: target contains reports outside the confirmed ledger")
		}
		var live, correct int
		if err = tx.QueryRowContext(ctx, `SELECT count(*),count(*) FILTER(WHERE r.primary_division=m.primary_division AND r.report_week=m.recovered_week AND r.number=m.recovered_number) FROM reports r JOIN weekly_report_recovery_rows m ON r.id=m.report_id WHERE m.recovery_name=$1 AND m.primary_division=$2`, weeklyRecoveryName, d.Division).Scan(&live, &correct); err != nil {
			return err
		}
		if live != correct {
			return errors.New("weekly recovery: selected report assignment changed")
		}
		var archiveCount int
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM weekly_report_archives WHERE primary_division=$1 AND (week_start AT TIME ZONE 'Asia/Jakarta')::date=$2::date`, d.Division, weeklyRecoveryTarget).Scan(&archiveCount); err != nil {
			return err
		}
		if archiveCount > 1 {
			return errors.New("weekly recovery: multiple target archives require manual reconciliation")
		}
		var snapshot []byte
		var status, digest, end string
		var reset sql.NullTime
		err = tx.QueryRowContext(ctx, `SELECT snapshot,status,pdf_sha256,reset_at,(week_end AT TIME ZONE 'Asia/Jakarta')::date::text FROM weekly_report_archives WHERE primary_division=$1 AND (week_start AT TIME ZONE 'Asia/Jakarta')::date=$2::date`, d.Division, weeklyRecoveryTarget).Scan(&snapshot, &status, &digest, &reset, &end)
		if errors.Is(err, sql.ErrNoRows) {
			if complete || live != d.Count {
				return errors.New("weekly recovery: selected reports missing before archival")
			}
			continue
		}
		if err != nil {
			return err
		}
		var items []struct {
			ID string `json:"id"`
		}
		if err = json.Unmarshal(snapshot, &items); err != nil {
			return err
		}
		if end != weeklyRecoverySource || len(items) != d.Count {
			return errors.New("weekly recovery: archive period or snapshot count differs from confirmed ledger")
		}
		for _, item := range items {
			if !ids[item.ID] {
				return errors.New("weekly recovery: archive snapshot IDs differ from confirmed ledger")
			}
			delete(ids, item.ID)
		}
		if len(ids) != 0 {
			return errors.New("weekly recovery: archive is missing confirmed reports")
		}
		if complete && (live != 0 || status != "finalized" || digest == "" || !reset.Valid) {
			return errors.New("weekly recovery: archival/reset is incomplete")
		}
	}
	return nil
}
