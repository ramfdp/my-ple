package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"my-ple-backend/model"
)

//go:embed schema.sql
var schemaSQL string

//go:embed weekly_archive.sql
var weeklyArchiveSQL string

//go:embed jobs.sql
var jobsSQL string

//go:embed report_numbering.sql
var reportNumberingSQL string

//go:embed job_pics.sql
var jobPICsSQL string

//go:embed workspace.sql
var workspaceSQL string

//go:embed leave.sql
var leaveSQL string

//go:embed email_delivery.sql
var emailDeliverySQL string

//go:embed collaboration.sql
var collaborationSQL string

//go:embed qa_validation.sql
var qaValidationSQL string

type Database struct {
	Conn *sql.DB
	Now  func() time.Time
}

var ErrInvalidReport = errors.New("invalid report")
var ErrReportConflict = errors.New("Laporan sudah berubah. Bandingkan dengan data terbaru sebelum menyimpan lagi.")
var ErrUserHasReports = errors.New("user has working reports awaiting archival")
var ErrZoomBookingConflict = errors.New("zoom booking conflict")

func NewDatabase(conn *sql.DB) *Database {
	return &Database{Conn: conn, Now: time.Now}
}

// ponytail: auto-migrate standard embedded schema without external migration tooling
func (db *Database) Migrate(ctx context.Context) error {
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71260400)`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71260402)`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, schemaSQL+weeklyArchiveSQL+jobsSQL+reportNumberingSQL+jobPICsSQL+workspaceSQL+leaveSQL+emailDeliverySQL+collaborationSQL+qaValidationSQL); err != nil {
		return err
	}
	var unmatched int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM unmatched_job_references`).Scan(&unmatched); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if unmatched > 0 {
		log.Printf("job migration: %d unmatched references retained; inspect unmatched_job_references", unmatched)
	}
	return nil
}

// ponytail: single SELECT query covering all core user and profile fields
func (db *Database) FindUser(ctx context.Context, identifier string) (model.User, error) {
	var u model.User
	err := db.Conn.QueryRowContext(ctx, `
		SELECT u.id, u.name, u.email, u.role_id, r.name, u.username, u.password_hash,
		       COALESCE(u.job_title, ''), COALESCE(u.salary, 0), COALESCE(u.residence, ''),
		       COALESCE(u.phone_number, ''), COALESCE(u.employment_status, 'Permanent Employee'),
		       COALESCE(u.avatar_url, ''), COALESCE(u.is_active, true), u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.username = $1 OR LOWER(u.email) = LOWER($1)`, identifier,
	).Scan(
		&u.ID, &u.Name, &u.Email, &u.RoleID, &u.RoleName, &u.Username, &u.PasswordHash,
		&u.JobTitle, &u.Salary, &u.Residence, &u.PhoneNumber, &u.EmploymentStatus,
		&u.AvatarURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}

func (db *Database) FindUserByID(ctx context.Context, id string) (model.User, error) {
	var u model.User
	err := db.Conn.QueryRowContext(ctx, `
		SELECT u.id, u.name, u.email, u.role_id, r.name, u.username, u.password_hash,
		       COALESCE(u.job_title, ''), COALESCE(u.salary, 0), COALESCE(u.residence, ''),
		       COALESCE(u.phone_number, ''), COALESCE(u.employment_status, 'Permanent Employee'),
		       COALESCE(u.avatar_url, ''), COALESCE(u.is_active, true), u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1`, id,
	).Scan(
		&u.ID, &u.Name, &u.Email, &u.RoleID, &u.RoleName, &u.Username, &u.PasswordHash,
		&u.JobTitle, &u.Salary, &u.Residence, &u.PhoneNumber, &u.EmploymentStatus,
		&u.AvatarURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}

func (db *Database) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := db.Conn.ExecContext(ctx, `INSERT INTO sessions (id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`, NewID(), userID, tokenHash, expiresAt)
	return err
}

func (db *Database) FindSessionUser(ctx context.Context, tokenHash string) (model.User, time.Time, error) {
	var u model.User
	var expires time.Time
	err := db.Conn.QueryRowContext(ctx, `UPDATE sessions s SET last_activity_at=NOW() FROM users u JOIN roles r ON u.role_id=r.id WHERE s.token_hash=$1 AND s.user_id=u.id AND s.revoked_at IS NULL AND s.expires_at>NOW() AND u.is_active RETURNING u.id,u.name,u.email,u.role_id,r.name,u.username,u.password_hash,u.job_title,u.salary,u.residence,u.phone_number,u.employment_status,u.avatar_url,u.is_active,u.created_at,u.updated_at,s.expires_at`, tokenHash).Scan(&u.ID, &u.Name, &u.Email, &u.RoleID, &u.RoleName, &u.Username, &u.PasswordHash, &u.JobTitle, &u.Salary, &u.Residence, &u.PhoneNumber, &u.EmploymentStatus, &u.AvatarURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &expires)
	return u, expires, err
}

func (db *Database) RevokeSession(ctx context.Context, tokenHash string) error {
	_, err := db.Conn.ExecContext(ctx, `UPDATE sessions SET revoked_at=NOW() WHERE token_hash=$1 AND revoked_at IS NULL`, tokenHash)
	return err
}

func (db *Database) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	result, err := db.Conn.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at<=NOW() OR revoked_at<NOW()-INTERVAL '24 hours'`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *Database) ListUsers(ctx context.Context) ([]model.User, error) {
	rows, err := db.Conn.QueryContext(ctx, `
		SELECT u.id, u.name, u.email, u.role_id, r.name, u.username, u.password_hash,
		       u.job_title, u.salary, u.residence, u.phone_number, u.employment_status,
		       u.avatar_url, u.is_active, u.created_at, u.updated_at
		FROM users u JOIN roles r ON u.role_id = r.id ORDER BY u.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.RoleID, &u.RoleName, &u.Username, &u.PasswordHash, &u.JobTitle, &u.Salary, &u.Residence, &u.PhoneNumber, &u.EmploymentStatus, &u.AvatarURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (db *Database) CreateUser(ctx context.Context, u model.User) (model.User, error) {
	if u.EmploymentStatus == "" {
		u.EmploymentStatus = "Permanent Employee"
	}
	err := db.Conn.QueryRowContext(ctx, `
		INSERT INTO users (id, name, email, role_id, username, password_hash, job_title, salary, residence, phone_number, employment_status, avatar_url, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
		RETURNING id, name, email, role_id, username, job_title, salary, residence, phone_number, employment_status, avatar_url, is_active, created_at, updated_at`,
		u.ID, u.Name, u.Email, u.RoleID, u.Username, u.PasswordHash,
		u.JobTitle, u.Salary, u.Residence, u.PhoneNumber, u.EmploymentStatus, u.AvatarURL, u.IsActive,
	).Scan(
		&u.ID, &u.Name, &u.Email, &u.RoleID, &u.Username,
		&u.JobTitle, &u.Salary, &u.Residence, &u.PhoneNumber, &u.EmploymentStatus,
		&u.AvatarURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}

func (db *Database) UpdateUserProfile(ctx context.Context, id string, input model.UpdateProfileInput) (model.User, error) {
	var u model.User
	err := db.Conn.QueryRowContext(ctx, `
		UPDATE users
		SET residence = $2, phone_number = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, email, role_id, username, password_hash, job_title, salary, residence, phone_number, employment_status, avatar_url, is_active, created_at, updated_at`,
		id, input.Residence, input.PhoneNumber,
	).Scan(
		&u.ID, &u.Name, &u.Email, &u.RoleID, &u.Username, &u.PasswordHash,
		&u.JobTitle, &u.Salary, &u.Residence, &u.PhoneNumber, &u.EmploymentStatus,
		&u.AvatarURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == nil {
		role, rErr := db.FindRoleByID(ctx, u.RoleID)
		if rErr == nil {
			u.RoleName = role.Name
		}
	}
	return u, err
}

func (db *Database) UpdateUser(ctx context.Context, id string, input model.UpdateUserInput, passwordHash string) (model.User, error) {
	var u model.User
	err := db.Conn.QueryRowContext(ctx, `
		UPDATE users SET name=$2, email=$3, username=$4, password_hash=COALESCE(NULLIF($5, ''), password_hash),
		role_id=$6, job_title=$7, salary=$8, residence=$9, phone_number=$10, employment_status=$11,
		avatar_url=$12, is_active=$13, updated_at=NOW() WHERE id=$1
		RETURNING id, name, email, role_id, username, password_hash, job_title, salary, residence, phone_number, employment_status, avatar_url, is_active, created_at, updated_at`,
		id, input.Name, input.Email, input.Username, passwordHash, input.RoleID, input.JobTitle, input.Salary, input.Residence, input.PhoneNumber, input.EmploymentStatus, input.AvatarURL, input.IsActive,
	).Scan(&u.ID, &u.Name, &u.Email, &u.RoleID, &u.Username, &u.PasswordHash, &u.JobTitle, &u.Salary, &u.Residence, &u.PhoneNumber, &u.EmploymentStatus, &u.AvatarURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, err
	}
	role, err := db.FindRoleByID(ctx, u.RoleID)
	if err == nil {
		u.RoleName = role.Name
	}
	return u, err
}

func (db *Database) UpdateAvatar(ctx context.Context, id, avatarURL string) (model.User, error) {
	var u model.User
	err := db.Conn.QueryRowContext(ctx, `
		UPDATE users SET avatar_url=$2, updated_at=NOW() WHERE id=$1
		RETURNING id, name, email, role_id, username, password_hash, job_title, salary, residence, phone_number, employment_status, avatar_url, is_active, created_at, updated_at`, id, avatarURL,
	).Scan(&u.ID, &u.Name, &u.Email, &u.RoleID, &u.Username, &u.PasswordHash, &u.JobTitle, &u.Salary, &u.Residence, &u.PhoneNumber, &u.EmploymentStatus, &u.AvatarURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, err
	}
	role, err := db.FindRoleByID(ctx, u.RoleID)
	if err == nil {
		u.RoleName = role.Name
	}
	return u, err
}

func (db *Database) DeleteUser(ctx context.Context, id string) error {
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Reports must be archived/reset before their author can be removed.
	var hasReports bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM reports WHERE user_id=$1)`, id).Scan(&hasReports); err != nil {
		return err
	}
	if hasReports {
		return ErrUserHasReports
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func scanZoomBooking(row interface{ Scan(...any) error }) (model.ZoomBooking, error) {
	var booking model.ZoomBooking
	err := row.Scan(&booking.ID, &booking.UserID, &booking.CreatedBy, &booking.Title, &booking.Description, &booking.StartAt, &booking.EndAt, &booking.CreatedAt, &booking.UpdatedAt,
		&booking.ZoomResource.ID, &booking.ZoomResource.Name, &booking.ZoomResource.Code, &booking.ZoomResource.Description, &booking.ZoomResource.MeetingURL, &booking.ZoomResource.MeetingID, &booking.ZoomResource.HostKey, &booking.ZoomResource.Passcode, &booking.ZoomResource.CreatedAt, &booking.ZoomResource.UpdatedAt)
	return booking, err
}

func zoomBookingSelect() string {
	return `z.id, z.user_id, u.name, z.title, z.description, z.start_at, z.end_at, z.created_at, z.updated_at,
		r.id, r.name, r.code, r.description, r.meeting_url, r.meeting_id, r.host_key, r.passcode, r.created_at, r.updated_at`
}

func zoomConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23P01" {
		return ErrZoomBookingConflict
	}
	return err
}

func (db *Database) ListZoomBookings(ctx context.Context, start, end time.Time) ([]model.ZoomBooking, error) {
	rows, err := db.Conn.QueryContext(ctx, `
		SELECT `+zoomBookingSelect()+`
		FROM zoom_bookings z JOIN users u ON u.id = z.user_id JOIN zoom_resources r ON r.id=z.zoom_resource_id
		WHERE z.start_at < $2 AND z.end_at > $1 ORDER BY z.start_at`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	bookings := []model.ZoomBooking{}
	for rows.Next() {
		booking, err := scanZoomBooking(rows)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, booking)
	}
	return bookings, rows.Err()
}

func (db *Database) UpdateZoomBooking(ctx context.Context, id, userID string, input model.ZoomBookingInput) (model.ZoomBooking, error) {
	booking, err := scanZoomBooking(db.Conn.QueryRowContext(ctx, `
		WITH updated AS (
		UPDATE zoom_bookings SET title=$3, description=$4, zoom_resource_id=$5, start_at=$6, end_at=$7, updated_at=NOW()
		WHERE id=$1 AND user_id=$2 AND start_at > NOW()
		RETURNING *) SELECT `+zoomBookingSelect()+` FROM updated z JOIN users u ON u.id=z.user_id JOIN zoom_resources r ON r.id=z.zoom_resource_id`,
		id, userID, input.Title, input.Description, input.ZoomResourceID, input.StartAt, input.EndAt))
	return booking, zoomConflict(err)
}

func scanZoomResource(row interface{ Scan(...any) error }) (model.ZoomResource, error) {
	var resource model.ZoomResource
	err := row.Scan(&resource.ID, &resource.Name, &resource.Code, &resource.Description, &resource.MeetingURL, &resource.MeetingID, &resource.HostKey, &resource.Passcode, &resource.CreatedAt, &resource.UpdatedAt)
	return resource, err
}

func (db *Database) ListZoomResources(ctx context.Context) ([]model.ZoomResource, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT id, name, code, description, meeting_url, meeting_id, host_key, passcode, created_at, updated_at FROM zoom_resources ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resources := []model.ZoomResource{}
	for rows.Next() {
		resource, err := scanZoomResource(rows)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, rows.Err()
}

func (db *Database) UpdateZoomResource(ctx context.Context, id int, input model.ZoomResourceUpdateInput) (model.ZoomResource, error) {
	return scanZoomResource(db.Conn.QueryRowContext(ctx, `UPDATE zoom_resources SET description=$2, meeting_url=$3, meeting_id=$4, host_key=$5, passcode=$6, updated_at=NOW() WHERE id=$1 RETURNING id, name, code, description, meeting_url, meeting_id, host_key, passcode, created_at, updated_at`, id, input.Description, input.MeetingURL, input.MeetingID, input.HostKey, input.Passcode))
}

func (db *Database) DeleteZoomBooking(ctx context.Context, id, userID string) error {
	result, err := db.Conn.ExecContext(ctx, `DELETE FROM zoom_bookings WHERE id=$1 AND user_id=$2 AND start_at > NOW()`, id, userID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (db *Database) ListRoles(ctx context.Context) ([]model.Role, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT id, name, COALESCE(description, '') FROM roles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []model.Role
	for rows.Next() {
		var r model.Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Description); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, rows.Err()
}

func (db *Database) FindRoleByName(ctx context.Context, name string) (model.Role, error) {
	var r model.Role
	err := db.Conn.QueryRowContext(ctx, `SELECT id, name, COALESCE(description, '') FROM roles WHERE LOWER(name) = LOWER($1)`, name).Scan(&r.ID, &r.Name, &r.Description)
	return r, err
}

func (db *Database) FindRoleByID(ctx context.Context, id int) (model.Role, error) {
	var r model.Role
	err := db.Conn.QueryRowContext(ctx, `SELECT id, name, COALESCE(description, '') FROM roles WHERE id = $1`, id).Scan(&r.ID, &r.Name, &r.Description)
	return r, err
}

// Both create and reassignment allocate in PostgreSQL, in the caller's transaction.
func allocateReportNumber(ctx context.Context, tx *sql.Tx, division, category string, jobID *int, completed time.Time) (int, error) {
	start, _ := model.ReportingWeek(completed)
	var number int
	err := tx.QueryRowContext(ctx, `INSERT INTO report_sequences(primary_division,work_division,week_start,job_id,last_number)
 VALUES($1,$2,$3,$4,1) ON CONFLICT ON CONSTRAINT report_sequences_job_scope
 DO UPDATE SET last_number=report_sequences.last_number+1 RETURNING last_number`, division, category, start.Format("2006-01-02"), jobID).Scan(&number)
	return number, err
}

func (db *Database) CreateReport(ctx context.Context, userID string, input model.WeeklyReportInput) (model.WeeklyReport, error) {
	var result model.WeeklyReport
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	now, err := db.writableReport(ctx, tx, "")
	if err != nil {
		return result, err
	}
	if err := resolveReportJob(ctx, tx, &input, nil); err != nil {
		return result, err
	}
	start, _ := model.ReportingWeek(now)
	var pendingReset bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM reports WHERE primary_division=$1 AND completed_at<$2)`, input.PrimaryDivision, start).Scan(&pendingReset); err != nil {
		return result, err
	}
	if pendingReset {
		return result, ErrReportClosed
	}
	number, err := allocateReportNumber(ctx, tx, input.PrimaryDivision, input.WorkDivision, input.JobID, now)
	if err != nil {
		return result, err
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO reports(id,user_id,primary_division,work_division,number,report_role,work,description,completed_at,job_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
 RETURNING id,user_id,primary_division,work_division,number,report_role,work,description,completed_at`,
		NewID(), userID, input.PrimaryDivision, input.WorkDivision, number, input.Role, input.Work, input.Description, now, input.JobID,
	).Scan(&result.ID, &result.UserID, &result.PrimaryDivision, &result.WorkDivision, &result.Number, &result.Role, &result.Work, &result.Description, &result.CompletedAt)
	if err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	result.JobID = input.JobID
	result.Version = 1
	model.NormalizeWeeklyReport(&result)
	return result, nil
}

func (db *Database) ListReports(ctx context.Context, primaryDivision string, weekStart, nextWeekStart time.Time) ([]model.WeeklyReport, error) {
	rows, err := db.Conn.QueryContext(ctx, `
		SELECT id, user_id, primary_division, work_division, number, report_role, COALESCE((SELECT name FROM jobs WHERE id=reports.job_id),work), description, completed_at, reports.job_id, p.pics, reports.version
		FROM reports LEFT JOIN job_pic_summary p ON p.job_id=reports.job_id WHERE primary_division = $1 AND completed_at >= $2 AND completed_at < $3
		ORDER BY CASE work_division WHEN 'MK supervisi' THEN 1 WHEN 'Non-MK supervisi' THEN 2 WHEN 'QA 1' THEN 3 WHEN 'QA 2' THEN 4 ELSE 5 END,
		work_division, report_role, MIN(number) OVER (PARTITION BY primary_division, work_division, report_role, work), number, completed_at, id`, primaryDivision, weekStart, nextWeekStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reports := []model.WeeklyReport{}
	for rows.Next() {
		var item model.WeeklyReport
		if err := rows.Scan(&item.ID, &item.UserID, &item.PrimaryDivision, &item.WorkDivision, &item.Number, &item.Role, &item.Work, &item.Description, &item.CompletedAt, &item.JobID, &item.PICs, &item.Version); err != nil {
			return nil, err
		}
		model.NormalizeWeeklyReport(&item)
		reports = append(reports, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	SortReports(reports)
	return reports, nil
}

// SortReports keeps manual and archived PDFs on the existing grouping order.
func SortReports(reports []model.WeeklyReport) {
	roleRank := map[string]int{"Project Engineer": 1, "Quality Assurance": 2, "CIHPK": 3}
	categoryRank := map[string]int{"MK/Supervisi": 1, "Non MK/Non Supervisi": 2, "QA 1": 3, "QA 2": 4}
	sort.SliceStable(reports, func(i, j int) bool {
		a, b := reports[i], reports[j]
		if roleRank[a.Role] != roleRank[b.Role] {
			return roleRank[a.Role] < roleRank[b.Role]
		}
		if categoryRank[a.WorkDivision] != categoryRank[b.WorkDivision] {
			return categoryRank[a.WorkDivision] < categoryRank[b.WorkDivision]
		}
		if a.WorkDivision != b.WorkDivision {
			return a.WorkDivision < b.WorkDivision
		}
		if a.Work != b.Work {
			return a.Work < b.Work
		}
		if a.Number != b.Number {
			return a.Number < b.Number
		}
		if !a.CompletedAt.Equal(b.CompletedAt) {
			return a.CompletedAt.Before(b.CompletedAt)
		}
		return a.ID < b.ID
	})
	numbers := map[string]int{}
	for i := range reports {
		r := &reports[i]
		key := r.PrimaryDivision + "\x00" + r.Role + "\x00" + r.WorkDivision + "\x00" + r.Work
		numbers[key]++
		r.DisplayNumber = numbers[key]
	}
}

func (db *Database) UpdateReport(ctx context.Context, id string, input model.WeeklyReportUpdateInput) (model.WeeklyReport, error) {
	var result model.WeeklyReport
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if _, err := db.writableReport(ctx, tx, id); err != nil {
		return result, err
	}
	var old model.WeeklyReport
	if err := tx.QueryRowContext(ctx, `SELECT id,user_id,primary_division,work_division,report_role,work,job_id,number,completed_at,description,version FROM reports WHERE id=$1 FOR UPDATE`, id).Scan(&old.ID, &old.UserID, &old.PrimaryDivision, &old.WorkDivision, &old.Role, &old.Work, &old.JobID, &old.Number, &old.CompletedAt, &old.Description, &old.Version); err != nil {
		return result, err
	}
	model.NormalizeWeeklyReport(&old)
	if input.Version != old.Version {
		return old, ErrReportConflict
	}
	if input.PrimaryDivision != nil && *input.PrimaryDivision != old.PrimaryDivision {
		return result, ErrInvalidReport
	}
	value := func(v *string, fallback string) string {
		if v != nil {
			return strings.TrimSpace(*v)
		}
		return fallback
	}
	resolved := model.WeeklyReportInput{JobID: old.JobID, PrimaryDivision: old.PrimaryDivision, WorkDivision: value(input.WorkDivision, old.WorkDivision), Role: value(input.Role, old.Role), Work: value(input.Work, old.Work), Description: value(input.Description, old.Description)}
	if input.JobID != nil {
		resolved.JobID = input.JobID
	}
	if resolved.Description == "" {
		return result, ErrInvalidReport
	}
	if err := resolveReportJob(ctx, tx, &resolved, &old); err != nil {
		return result, err
	}
	number := old.Number
	if resolved.JobID != nil && (old.JobID == nil || *resolved.JobID != *old.JobID) {
		number, err = allocateReportNumber(ctx, tx, resolved.PrimaryDivision, resolved.WorkDivision, resolved.JobID, old.CompletedAt)
		if err != nil {
			return result, err
		}
	}
	if err := tx.QueryRowContext(ctx, `UPDATE reports SET work_division=$2,report_role=$3,work=$4,description=$5,job_id=$6,number=$7,version=version+1 WHERE id=$1 AND version=$8
 RETURNING id,user_id,primary_division,work_division,number,report_role,work,description,completed_at,version`,
		id, resolved.WorkDivision, resolved.Role, resolved.Work, resolved.Description, resolved.JobID, number, input.Version,
	).Scan(&result.ID, &result.UserID, &result.PrimaryDivision, &result.WorkDivision, &result.Number, &result.Role, &result.Work, &result.Description, &result.CompletedAt, &result.Version); err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	result.JobID = resolved.JobID
	model.NormalizeWeeklyReport(&result)
	return result, nil
}

func (db *Database) DeleteReport(ctx context.Context, id string) error {
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := db.writableReport(ctx, tx, id); err != nil {
		return err
	}
	if err := tx.QueryRowContext(ctx, `SELECT id FROM reports WHERE id = $1 FOR UPDATE`, id).Scan(&id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM reports WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func UniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func NewID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}
