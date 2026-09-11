package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"my-ple-backend/model"
	"strings"
	"time"
)

var ErrReportClosed = errors.New("report period is read-only")

// Serialize snapshot creation with all working mutations, including requests
// arriving across the Monday boundary. The worker has a separate session lock.
func (db *Database) writableReport(ctx context.Context, tx *sql.Tx, id string) (time.Time, error) {
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71260402)`); err != nil {
		return time.Time{}, err
	}
	now := time.Now()
	if db.Now != nil {
		now = db.Now()
	}
	start, _ := model.ReportingWeek(now)
	if now.Before(start.AddDate(0, 0, 1)) {
		return now, ErrReportClosed
	}
	if id == "" {
		return now, nil
	}
	var completed time.Time
	var archived bool
	err := tx.QueryRowContext(ctx, `SELECT completed_at, EXISTS(SELECT 1 FROM weekly_report_archives a WHERE a.primary_division=r.primary_division AND r.completed_at>=a.week_start AND r.completed_at<a.week_end) FROM reports r WHERE id=$1 FOR UPDATE`, id).Scan(&completed, &archived)
	if err != nil {
		return now, err
	}
	if completed.Before(start) || archived {
		return now, ErrReportClosed
	}
	return now, nil
}

func (db *Database) WorkingPeriod(ctx context.Context, division string, now time.Time) (model.ReportingPeriod, error) {
	start, next := model.ReportingWeek(now)
	var oldest sql.NullTime
	err := db.Conn.QueryRowContext(ctx, `SELECT MIN(completed_at) FROM reports WHERE primary_division=$1 AND completed_at<$2`, division, start).Scan(&oldest)
	if err != nil {
		return model.ReportingPeriod{}, err
	}
	readOnly := false
	if oldest.Valid {
		start, next = model.ReportingWeek(oldest.Time)
		readOnly = true
	} else if now.Before(start.AddDate(0, 0, 1)) {
		next = start
		start = start.AddDate(0, 0, -7)
		readOnly = true
	}
	status := "active"
	if readOnly {
		status = "pending"
		err := db.Conn.QueryRowContext(ctx, `SELECT status FROM weekly_report_archives WHERE primary_division=$1 AND week_start=$2`, division, start).Scan(&status)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return model.ReportingPeriod{}, err
		}
	}
	return model.ReportingPeriod{Start: start, End: next.AddDate(0, 0, -1), Label: model.IndonesianWeekLabel(start, next), ReadOnly: readOnly, Status: status}, nil
}

const archiveColumns = `id,primary_division,week_start,week_end,snapshot,status,finalized_at,pdf_filename,pdf_sha256,pdf_generated_at,reset_at,created_at,pdf_storage_key`

func scanArchive(row interface{ Scan(...any) error }) (model.WeeklyArchive, error) {
	var a model.WeeklyArchive
	var snapshot []byte
	err := row.Scan(&a.ID, &a.Division, &a.WeekStart, &a.WeekEnd, &snapshot, &a.Status, &a.FinalizedAt, &a.Filename, &a.SHA256, &a.PDFGeneratedAt, &a.ResetAt, &a.CreatedAt, &a.StorageKey)
	if err == nil {
		err = json.Unmarshal(snapshot, &a.Items)
		for i := range a.Items {
			model.NormalizeWeeklyReport(&a.Items[i].WeeklyReport)
		}
	}
	return a, err
}

func (db *Database) Archive(ctx context.Context, id string) (model.WeeklyArchive, error) {
	return scanArchive(db.Conn.QueryRowContext(ctx, `SELECT `+archiveColumns+` FROM weekly_report_archives WHERE id::text=$1`, id))
}

func (db *Database) PeriodReports(ctx context.Context, division string, period model.ReportingPeriod) ([]model.WeeklyReport, error) {
	if period.ReadOnly {
		a, err := scanArchive(db.Conn.QueryRowContext(ctx, `SELECT `+archiveColumns+` FROM weekly_report_archives WHERE primary_division=$1 AND week_start=$2`, division, period.Start))
		if err == nil {
			reports := a.Reports()
			SortReports(reports)
			return reports, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	return db.ListReports(ctx, division, period.Start, period.End.AddDate(0, 0, 1))
}

func (db *Database) ArchiveHistory(ctx context.Context, division, date string, offset int) ([]model.WeeklyArchive, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT `+strings.Replace(archiveColumns, "snapshot", "'[]'::jsonb", 1)+` FROM weekly_report_archives WHERE ($1='' OR primary_division=$1) AND ($2='' OR (week_start <= NULLIF($2,'')::date::timestamp AT TIME ZONE 'Asia/Jakarta' AND week_end > NULLIF($2,'')::date::timestamp AT TIME ZONE 'Asia/Jakarta')) ORDER BY week_start DESC,id LIMIT 21 OFFSET $3`, division, date, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.WeeklyArchive{}
	for rows.Next() {
		a, err := scanArchive(rows)
		if err != nil {
			return nil, err
		}
		a.Items = nil
		result = append(result, a)
	}
	return result, rows.Err()
}

type DueWeek struct {
	Division string
	Start    time.Time
}

func (db *Database) DueWeeks(ctx context.Context, now time.Time) ([]DueWeek, error) {
	start, _ := model.ReportingWeek(now)
	rows, err := db.Conn.QueryContext(ctx, `SELECT primary_division, date_trunc('week',completed_at AT TIME ZONE 'Asia/Jakarta') AT TIME ZONE 'Asia/Jakarta' AS week_start FROM reports WHERE completed_at<$1 AND $2 GROUP BY 1,2 UNION SELECT primary_division,week_start FROM weekly_report_archives WHERE reset_at IS NULL OR status<>'finalized' ORDER BY 2,1`, start, weeklyRolloverReady(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []DueWeek{}
	for rows.Next() {
		var w DueWeek
		if err := rows.Scan(&w.Division, &w.Start); err != nil {
			return nil, err
		}
		result = append(result, w)
	}
	return result, rows.Err()
}

func weeklyRolloverReady(now time.Time) bool {
	start, _ := model.ReportingWeek(now)
	return !now.Before(start.AddDate(0, 0, 1))
}

func (db *Database) SnapshotWeek(ctx context.Context, w DueWeek) (model.WeeklyArchive, error) {
	start, next := model.ReportingWeek(w.Start)
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return model.WeeklyArchive{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71260402)`); err != nil {
		return model.WeeklyArchive{}, err
	}
	existing, err := scanArchive(tx.QueryRowContext(ctx, `SELECT `+archiveColumns+` FROM weekly_report_archives WHERE primary_division=$1 AND week_start=$2 AND week_end=$3`, w.Division, start, next))
	if err == nil {
		return existing, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return existing, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT r.id,r.user_id,r.primary_division,r.work_division,r.number,r.report_role,COALESCE((SELECT name FROM jobs WHERE id=r.job_id),r.work),r.description,r.completed_at,r.job_id,u.name,u.role_id,roles.name,p.pics FROM reports r LEFT JOIN job_pic_summary p ON p.job_id=r.job_id JOIN users u ON u.id=r.user_id JOIN roles ON roles.id=u.role_id WHERE r.primary_division=$1 AND r.completed_at>=$2 AND r.completed_at<$3 ORDER BY r.report_role,r.work_division,r.work,r.number,r.completed_at,r.id`, w.Division, start, next)
	if err != nil {
		return existing, err
	}
	items := []model.ArchiveItem{}
	for rows.Next() {
		var item model.ArchiveItem
		err = rows.Scan(&item.ID, &item.UserID, &item.PrimaryDivision, &item.WorkDivision, &item.Number, &item.Role, &item.Work, &item.Description, &item.CompletedAt, &item.JobID, &item.EmployeeName, &item.EmployeeRoleID, &item.EmployeeRole, &item.PICs)
		if err != nil {
			rows.Close()
			return existing, err
		}
		model.NormalizeWeeklyReport(&item.WeeklyReport)
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return existing, err
	}
	if len(items) == 0 {
		return existing, sql.ErrNoRows
	}
	snapshot, err := json.Marshal(items)
	if err != nil {
		return existing, err
	}
	a, err := scanArchive(tx.QueryRowContext(ctx, `INSERT INTO weekly_report_archives(id,primary_division,week_start,week_end,snapshot) VALUES($1,$2,$3,$4,$5) RETURNING `+archiveColumns, NewID(), w.Division, start, next, snapshot))
	if err != nil {
		return a, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO weekly_archive_jobs(archive_id,report_id,job_id) SELECT $1,item->>'id',(item->>'jobId')::int FROM jsonb_array_elements($2::jsonb) item WHERE item->>'jobId' IS NOT NULL ON CONFLICT DO NOTHING`, a.ID, snapshot); err != nil {
		return a, err
	}
	return a, tx.Commit()
}

// Called only after the worker has verified the archive's stored PDF checksum.
func (db *Database) ResetArchivedWeek(ctx context.Context, a model.WeeklyArchive, now time.Time) (int64, error) {
	if a.Status != "finalized" || a.SHA256 == "" || now.Before(a.WeekEnd.Add(24*time.Hour)) {
		return 0, ErrReportClosed
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71260402)`); err != nil {
		return 0, err
	}
	// Delete exactly the snapshotted IDs, never an unrelated/new report.
	result, err := tx.ExecContext(ctx, `DELETE FROM reports r USING weekly_report_archives a WHERE a.id=$1 AND a.status='finalized' AND a.pdf_sha256=$2 AND r.primary_division=a.primary_division AND r.completed_at>=a.week_start AND r.completed_at<a.week_end AND EXISTS(SELECT 1 FROM jsonb_array_elements(a.snapshot) item WHERE item->>'id'=r.id::text)`, a.ID, a.SHA256)
	if err != nil {
		return 0, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE weekly_report_archives SET reset_at=COALESCE(reset_at,$2) WHERE id=$1`, a.ID, now); err != nil {
		return 0, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM report_sequences s WHERE primary_division=$1 AND week_start=($2::timestamptz AT TIME ZONE 'Asia/Jakarta')::date AND NOT EXISTS(SELECT 1 FROM reports r WHERE r.primary_division=s.primary_division AND r.report_week=s.week_start)`, a.Division, a.WeekStart); err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return count, tx.Commit()
}
