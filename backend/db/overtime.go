package db

import (
	"context"
	"database/sql"
	"errors"
	"my-ple-backend/model"
	"strings"
	"time"
)

var ErrOvertimeInvalid = errors.New("Data lembur tidak valid.")
var ErrOvertimeForbidden = errors.New("Anda tidak berhak mengakses atau meninjau lembur ini.")
var ErrOvertimeConflict = errors.New("Data lembur sudah berubah atau diproses. Muat ulang data.")
var ErrOvertimeWindow = errors.New("Pengajuan hanya tersedia tanggal 26 sampai 5 bulan berikutnya.")

type overtimeQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

const overtimeSheetSelect = `SELECT s.id,s.user_id,u.name,r.name,to_char(s.period,'YYYY-MM'),s.status,s.version,
 COALESCE(t.actual,0),COALESCE(t.weighted,0),s.submitted_at,s.reviewed_by,COALESCE(rv.name,''),s.reviewed_at,s.reviewer_note,s.created_at,s.updated_at
 FROM overtime_sheets s JOIN users u ON u.id=s.user_id JOIN roles r ON r.id=u.role_id LEFT JOIN users rv ON rv.id=s.reviewed_by
 LEFT JOIN (SELECT sheet_id,SUM(actual_minutes) actual,SUM(weighted_minutes) weighted FROM overtime_entries GROUP BY sheet_id) t ON t.sheet_id=s.id `

func scanOvertimeSheet(row interface{ Scan(...any) error }) (model.OvertimeSheet, error) {
	var item model.OvertimeSheet
	err := row.Scan(&item.ID, &item.UserID, &item.EmployeeName, &item.EmployeeRole, &item.Period, &item.Status, &item.Version,
		&item.ActualMinutes, &item.WeightedMinutes, &item.SubmittedAt, &item.ReviewedBy, &item.ReviewerName, &item.ReviewedAt,
		&item.ReviewerNote, &item.CreatedAt, &item.UpdatedAt)
	item.Entries = []model.OvertimeEntry{}
	return item, err
}

func loadOvertimeEntries(ctx context.Context, q overtimeQuerier, sheetID string) ([]model.OvertimeEntry, error) {
	rows, err := q.QueryContext(ctx, `SELECT work_date::text,to_char(start_time,'HH24:MI'),to_char(end_time,'HH24:MI'),day_type,is_override,override_reason,project,description,actual_minutes,weighted_minutes FROM overtime_entries WHERE sheet_id=$1 ORDER BY work_date`, sheetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.OvertimeEntry{}
	for rows.Next() {
		var item model.OvertimeEntry
		if err = rows.Scan(&item.WorkDate, &item.StartTime, &item.EndTime, &item.DayType, &item.IsOverride, &item.OverrideReason, &item.Project, &item.Description, &item.ActualMinutes, &item.WeightedMinutes); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func overtimeReviewerRole(role string) bool {
	return strings.EqualFold(role, "Finance") || strings.EqualFold(role, "Admin")
}

func (db *Database) overtimeRole(ctx context.Context, userID string) (string, error) {
	var role string
	err := db.Conn.QueryRowContext(ctx, `SELECT r.name FROM users u JOIN roles r ON r.id=u.role_id WHERE u.id=$1 AND u.is_active`, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrOvertimeForbidden
	}
	return role, err
}

func normalizeOvertimeEntries(period string, inputs []model.OvertimeEntryInput) ([]model.OvertimeEntry, error) {
	month, err := model.ParseOvertimePeriod(period)
	if err != nil || len(inputs) > 31 {
		return nil, ErrOvertimeInvalid
	}
	seen := map[string]bool{}
	items := make([]model.OvertimeEntry, 0, len(inputs))
	for _, input := range inputs {
		date, parseErr := time.ParseInLocation("2006-01-02", input.WorkDate, month.Location())
		input.Project = strings.TrimSpace(input.Project)
		input.Description = strings.TrimSpace(input.Description)
		input.OverrideReason = strings.TrimSpace(input.OverrideReason)
		if parseErr != nil || date.Format("2006-01") != period || seen[input.WorkDate] || input.Project == "" || len(input.Project) > 255 || input.Description == "" || len(input.Description) > 4000 || len(input.OverrideReason) > 500 {
			return nil, ErrOvertimeInvalid
		}
		seen[input.WorkDate] = true
		automatic := model.OvertimeWorkday
		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			automatic = model.OvertimeHoliday
		}
		if input.DayType != model.OvertimeWorkday && input.DayType != model.OvertimeHoliday {
			return nil, ErrOvertimeInvalid
		}
		override := input.DayType != automatic
		if override && input.OverrideReason == "" {
			return nil, ErrOvertimeInvalid
		}
		if !override {
			input.OverrideReason = ""
		}
		actual, weighted, calcErr := model.CalculateOvertime(input.StartTime, input.EndTime, input.DayType)
		if calcErr != nil || actual == 0 {
			return nil, ErrOvertimeInvalid
		}
		items = append(items, model.OvertimeEntry{WorkDate: input.WorkDate, StartTime: input.StartTime, EndTime: input.EndTime, DayType: input.DayType, IsOverride: override, OverrideReason: input.OverrideReason, Project: input.Project, Description: input.Description, ActualMinutes: actual, WeightedMinutes: weighted})
	}
	return items, nil
}

func (db *Database) OvertimeSheet(ctx context.Context, userID, period string) (model.OvertimeSheet, error) {
	if _, err := model.ParseOvertimePeriod(period); err != nil {
		return model.OvertimeSheet{}, ErrOvertimeInvalid
	}
	item, err := scanOvertimeSheet(db.Conn.QueryRowContext(ctx, overtimeSheetSelect+`WHERE s.user_id=$1 AND s.period=$2::date`, userID, period+"-01"))
	if errors.Is(err, sql.ErrNoRows) {
		role, roleErr := db.overtimeRole(ctx, userID)
		if roleErr != nil {
			return item, roleErr
		}
		var name string
		if err = db.Conn.QueryRowContext(ctx, `SELECT name FROM users WHERE id=$1`, userID).Scan(&name); err != nil {
			return item, err
		}
		return model.OvertimeSheet{UserID: userID, EmployeeName: name, EmployeeRole: role, Period: period, Status: "DRAFT", Entries: []model.OvertimeEntry{}}, nil
	}
	if err != nil {
		return item, err
	}
	item.Entries, err = loadOvertimeEntries(ctx, db.Conn, item.ID)
	return item, err
}

func (db *Database) SaveOvertimeSheet(ctx context.Context, userID, period string, input model.OvertimeSaveInput) (model.OvertimeSheet, error) {
	entries, err := normalizeOvertimeEntries(period, input.Entries)
	if err != nil || input.Version < 0 {
		return model.OvertimeSheet{}, ErrOvertimeInvalid
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return model.OvertimeSheet{}, err
	}
	defer tx.Rollback()
	id := NewID()
	if _, err = tx.ExecContext(ctx, `INSERT INTO overtime_sheets(id,user_id,period) VALUES($1,$2,$3::date) ON CONFLICT(user_id,period) DO NOTHING`, id, userID, period+"-01"); err != nil {
		return model.OvertimeSheet{}, err
	}
	var status string
	var version int
	if err = tx.QueryRowContext(ctx, `SELECT id,status,version FROM overtime_sheets WHERE user_id=$1 AND period=$2::date FOR UPDATE`, userID, period+"-01").Scan(&id, &status, &version); err != nil {
		return model.OvertimeSheet{}, err
	}
	if version != input.Version || (status != "DRAFT" && status != "REJECTED") {
		return model.OvertimeSheet{}, ErrOvertimeConflict
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM overtime_entries WHERE sheet_id=$1`, id); err != nil {
		return model.OvertimeSheet{}, err
	}
	for _, entry := range entries {
		if _, err = tx.ExecContext(ctx, `INSERT INTO overtime_entries(sheet_id,work_date,start_time,end_time,day_type,is_override,override_reason,project,description,actual_minutes,weighted_minutes) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, entry.WorkDate, entry.StartTime, entry.EndTime, entry.DayType, entry.IsOverride, entry.OverrideReason, entry.Project, entry.Description, entry.ActualMinutes, entry.WeightedMinutes); err != nil {
			return model.OvertimeSheet{}, err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE overtime_sheets SET version=version+1,updated_at=NOW() WHERE id=$1`, id); err != nil {
		return model.OvertimeSheet{}, err
	}
	if err = tx.Commit(); err != nil {
		return model.OvertimeSheet{}, err
	}
	return db.OvertimeSheet(ctx, userID, period)
}

func (db *Database) SubmitOvertimeSheet(ctx context.Context, userID, period string, version int) (model.OvertimeSheet, error) {
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return model.OvertimeSheet{}, err
	}
	defer tx.Rollback()
	var id, status string
	var currentVersion, count int
	err = tx.QueryRowContext(ctx, `SELECT s.id,s.status,s.version,(SELECT count(*) FROM overtime_entries e WHERE e.sheet_id=s.id) FROM overtime_sheets s WHERE s.user_id=$1 AND s.period=$2::date FOR UPDATE`, userID, period+"-01").Scan(&id, &status, &currentVersion, &count)
	if errors.Is(err, sql.ErrNoRows) || count == 0 {
		return model.OvertimeSheet{}, ErrOvertimeInvalid
	}
	if err != nil {
		return model.OvertimeSheet{}, err
	}
	if status == "SUBMITTED" && currentVersion == version+1 {
		_ = tx.Rollback()
		return db.OvertimeSheet(ctx, userID, period)
	}
	if currentVersion != version || (status != "DRAFT" && status != "REJECTED") {
		return model.OvertimeSheet{}, ErrOvertimeConflict
	}
	if status != "REJECTED" && !model.OvertimeSubmissionOpen(db.Now(), period) {
		return model.OvertimeSheet{}, ErrOvertimeWindow
	}
	if _, err = tx.ExecContext(ctx, `UPDATE overtime_sheets SET status='SUBMITTED',version=version+1,submitted_at=NOW(),reviewed_by=NULL,reviewed_at=NULL,reviewer_note='',created_notified=false,reviewed_notified=false,updated_at=NOW() WHERE id=$1`, id); err != nil {
		return model.OvertimeSheet{}, err
	}
	if err = tx.Commit(); err != nil {
		return model.OvertimeSheet{}, err
	}
	return db.OvertimeSheet(ctx, userID, period)
}

func (db *Database) OvertimeRoster(ctx context.Context, reviewerID, period, status string, offset int) ([]model.OvertimeRosterItem, error) {
	role, err := db.overtimeRole(ctx, reviewerID)
	if err != nil || !overtimeReviewerRole(role) {
		return nil, ErrOvertimeForbidden
	}
	if _, err = model.ParseOvertimePeriod(period); err != nil || offset < 0 || offset > 1000000 || (status != "" && status != "NO_ENTRY" && status != "DRAFT" && status != "SUBMITTED" && status != "APPROVED" && status != "REJECTED") {
		return nil, ErrOvertimeInvalid
	}
	rows, err := db.Conn.QueryContext(ctx, `SELECT COALESCE(s.id::text,''),u.id,u.name,r.name,COALESCE(s.status,'NO_ENTRY'),s.submitted_at,s.reviewed_at FROM users u JOIN roles r ON r.id=u.role_id LEFT JOIN overtime_sheets s ON s.user_id=u.id AND s.period=$1::date WHERE u.is_active AND ($2='' OR COALESCE(s.status,'NO_ENTRY')=$2) ORDER BY u.name,u.id LIMIT 50 OFFSET $3`, period+"-01", status, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.OvertimeRosterItem{}
	for rows.Next() {
		var item model.OvertimeRosterItem
		if err = rows.Scan(&item.SheetID, &item.UserID, &item.EmployeeName, &item.EmployeeRole, &item.Status, &item.SubmittedAt, &item.ReviewedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *Database) OvertimeReviewSheet(ctx context.Context, reviewerID, sheetID string) (model.OvertimeSheet, error) {
	role, err := db.overtimeRole(ctx, reviewerID)
	if err != nil || !overtimeReviewerRole(role) || !workspaceUUID(sheetID) {
		return model.OvertimeSheet{}, ErrOvertimeForbidden
	}
	item, err := scanOvertimeSheet(db.Conn.QueryRowContext(ctx, overtimeSheetSelect+`WHERE s.id=$1`, sheetID))
	if err != nil {
		return item, err
	}
	if strings.EqualFold(role, "Finance") && item.Status == "DRAFT" {
		return model.OvertimeSheet{}, ErrOvertimeForbidden
	}
	item.Entries, err = loadOvertimeEntries(ctx, db.Conn, item.ID)
	return item, err
}

func (db *Database) DecideOvertime(ctx context.Context, reviewerID, sheetID string, input model.OvertimeDecision) (model.OvertimeSheet, error) {
	role, err := db.overtimeRole(ctx, reviewerID)
	input.ReviewerNote = strings.TrimSpace(input.ReviewerNote)
	if err != nil || !overtimeReviewerRole(role) || !workspaceUUID(sheetID) {
		return model.OvertimeSheet{}, ErrOvertimeForbidden
	}
	if (input.Status != "APPROVED" && input.Status != "REJECTED") || len(input.ReviewerNote) > 4000 || (input.Status == "REJECTED" && input.ReviewerNote == "") {
		return model.OvertimeSheet{}, ErrOvertimeInvalid
	}
	result, err := db.Conn.ExecContext(ctx, `UPDATE overtime_sheets SET status=$3,version=version+1,reviewed_by=$2,reviewed_at=NOW(),reviewer_note=$4,reviewed_notified=false,updated_at=NOW() WHERE id=$1 AND status='SUBMITTED' AND user_id<>$2 AND EXISTS(SELECT 1 FROM users u JOIN roles r ON r.id=u.role_id WHERE u.id=$2 AND u.is_active AND r.name IN ('Finance','Admin'))`, sheetID, reviewerID, input.Status, input.ReviewerNote)
	if err != nil {
		return model.OvertimeSheet{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return model.OvertimeSheet{}, err
	}
	if count != 1 {
		item, readErr := db.OvertimeReviewSheet(ctx, reviewerID, sheetID)
		if readErr == nil && item.Status == input.Status && item.ReviewedBy != nil && *item.ReviewedBy == reviewerID {
			return item, nil
		}
		return model.OvertimeSheet{}, ErrOvertimeConflict
	}
	return db.OvertimeReviewSheet(ctx, reviewerID, sheetID)
}
