package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"my-ple-backend/model"
)

//go:embed schema.sql
var schemaSQL string

type Database struct {
	Conn *sql.DB
}

var ErrInvalidReport = errors.New("invalid report")
var ErrZoomBookingConflict = errors.New("zoom booking conflict")

func NewDatabase(conn *sql.DB) *Database {
	return &Database{Conn: conn}
}

// ponytail: auto-migrate standard embedded schema without external migration tooling
func (db *Database) Migrate(ctx context.Context) error {
	_, err := db.Conn.ExecContext(ctx, schemaSQL)
	return err
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
		SET name = COALESCE(NULLIF($2, ''), name),
		    email = COALESCE(NULLIF($3, ''), email),
		    job_title = COALESCE(NULLIF($4, ''), job_title),
		    salary = CASE WHEN $5 > 0 THEN $5 ELSE salary END,
		    residence = COALESCE(NULLIF($6, ''), residence),
		    phone_number = COALESCE(NULLIF($7, ''), phone_number),
		    employment_status = COALESCE(NULLIF($8, ''), employment_status),
		    avatar_url = COALESCE(NULLIF($9, ''), avatar_url),
		    role_id = CASE WHEN $10 > 0 THEN $10 ELSE role_id END,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, email, role_id, username, password_hash, job_title, salary, residence, phone_number, employment_status, avatar_url, is_active, created_at, updated_at`,
		id, input.Name, input.Email, input.JobTitle, input.Salary, input.Residence, input.PhoneNumber, input.EmploymentStatus, input.AvatarURL, input.RoleID,
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
	if _, err := tx.ExecContext(ctx, `DELETE FROM reports WHERE user_id = $1`, id); err != nil {
		return err
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
		&booking.ZoomResource.ID, &booking.ZoomResource.Name, &booking.ZoomResource.Code, &booking.ZoomResource.Description, &booking.ZoomResource.MeetingURL, &booking.ZoomResource.MeetingID, &booking.ZoomResource.HostKey, &booking.ZoomResource.CreatedAt, &booking.ZoomResource.UpdatedAt)
	return booking, err
}

func zoomBookingSelect() string {
	return `z.id, z.user_id, u.name, z.title, z.description, z.start_at, z.end_at, z.created_at, z.updated_at,
		r.id, r.name, r.code, r.description, r.meeting_url, r.meeting_id, r.host_key, r.created_at, r.updated_at`
}

func zoomConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23P01" {
		return ErrZoomBookingConflict
	}
	return err
}

func (db *Database) CreateZoomBooking(ctx context.Context, userID string, input model.ZoomBookingInput) (model.ZoomBooking, error) {
	booking, err := scanZoomBooking(db.Conn.QueryRowContext(ctx, `
		WITH inserted AS (
			INSERT INTO zoom_bookings (id, user_id, zoom_resource_id, title, description, start_at, end_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *
		)
		SELECT `+zoomBookingSelect()+` FROM inserted z JOIN users u ON u.id=z.user_id JOIN zoom_resources r ON r.id=z.zoom_resource_id`,
		NewID(), userID, input.ZoomResourceID, input.Title, input.Description, input.StartAt, input.EndAt))
	return booking, zoomConflict(err)
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
	err := row.Scan(&resource.ID, &resource.Name, &resource.Code, &resource.Description, &resource.MeetingURL, &resource.MeetingID, &resource.HostKey, &resource.CreatedAt, &resource.UpdatedAt)
	return resource, err
}

func (db *Database) ListZoomResources(ctx context.Context) ([]model.ZoomResource, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT id, name, code, description, meeting_url, meeting_id, host_key, created_at, updated_at FROM zoom_resources ORDER BY id`)
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
	return scanZoomResource(db.Conn.QueryRowContext(ctx, `UPDATE zoom_resources SET description=$2, meeting_url=$3, meeting_id=$4, host_key=$5, updated_at=NOW() WHERE id=$1 RETURNING id, name, code, description, meeting_url, meeting_id, host_key, created_at, updated_at`, id, input.Description, input.MeetingURL, input.MeetingID, input.HostKey))
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

func (db *Database) CreateReport(ctx context.Context, userID string, input model.WeeklyReportInput) (model.WeeklyReport, error) {
	var result model.WeeklyReport
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(ctx, `
		WITH next_number AS (
			INSERT INTO report_sequences (primary_division, work_division, last_number)
			VALUES ($1, $2, 1)
			ON CONFLICT (primary_division, work_division)
			DO UPDATE SET last_number = report_sequences.last_number + 1
			RETURNING last_number
		)
		INSERT INTO reports (id, user_id, primary_division, work_division, number, report_role, work, description, completed_at)
		SELECT $3, $4, $1, $2, last_number, $5, $6, $7, NOW() FROM next_number
		RETURNING id, user_id, primary_division, work_division, number, report_role, work, description, completed_at`,
		input.PrimaryDivision, input.WorkDivision, NewID(), userID, input.Role, input.Work, input.Description,
	).Scan(&result.ID, &result.UserID, &result.PrimaryDivision, &result.WorkDivision, &result.Number, &result.Role, &result.Work, &result.Description, &result.CompletedAt)
	if err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

func (db *Database) CleanupExpiredReports(ctx context.Context, weekStart time.Time) error {
	_, err := db.Conn.ExecContext(ctx, `DELETE FROM reports WHERE completed_at < $1`, weekStart)
	return err
}

func (db *Database) ListReports(ctx context.Context, primaryDivision string, weekStart, nextWeekStart time.Time) ([]model.WeeklyReport, error) {
	rows, err := db.Conn.QueryContext(ctx, `
		SELECT id, user_id, primary_division, work_division, number, report_role, work, description, completed_at
		FROM reports WHERE primary_division = $1 AND completed_at >= $2 AND completed_at < $3
		ORDER BY CASE work_division WHEN 'MK supervisi' THEN 1 WHEN 'Non-MK supervisi' THEN 2 WHEN 'QA 1' THEN 3 WHEN 'QA 2' THEN 4 ELSE 5 END,
		work_division, report_role, MIN(number) OVER (PARTITION BY primary_division, work_division, report_role, work), number, completed_at, id`, primaryDivision, weekStart, nextWeekStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reports := []model.WeeklyReport{}
	for rows.Next() {
		var item model.WeeklyReport
		if err := rows.Scan(&item.ID, &item.UserID, &item.PrimaryDivision, &item.WorkDivision, &item.Number, &item.Role, &item.Work, &item.Description, &item.CompletedAt); err != nil {
			return nil, err
		}
		reports = append(reports, item)
	}
	return reports, rows.Err()
}

func (db *Database) UpdateReport(ctx context.Context, id, userID string, input model.WeeklyReportUpdateInput) (model.WeeklyReport, error) {
	var result model.WeeklyReport
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	var primaryDivision, oldWorkDivision string
	if err := tx.QueryRowContext(ctx, `SELECT primary_division, work_division FROM reports WHERE id = $1 AND user_id = $2 FOR UPDATE`, id, userID).Scan(&primaryDivision, &oldWorkDivision); err != nil {
		return result, err
	}
	if input.PrimaryDivision != primaryDivision || !model.ValidWeeklyReport(input.PrimaryDivision, input.WorkDivision, input.Role, input.Work, input.Description) {
		return result, ErrInvalidReport
	}
	query := `UPDATE reports SET report_role = $3, work = $4, description = $5 WHERE id = $1 AND user_id = $2 RETURNING id, user_id, primary_division, work_division, number, report_role, work, description, completed_at`
	args := []any{id, userID, input.Role, input.Work, input.Description}
	if input.WorkDivision != oldWorkDivision {
		query = `WITH next_number AS (
			INSERT INTO report_sequences (primary_division, work_division, last_number) VALUES ($3, $4, 1)
			ON CONFLICT (primary_division, work_division) DO UPDATE SET last_number = report_sequences.last_number + 1
			RETURNING last_number
		)
		UPDATE reports SET work_division = $4, number = (SELECT last_number FROM next_number), report_role = $5, work = $6, description = $7
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, primary_division, work_division, number, report_role, work, description, completed_at`
		args = []any{id, userID, input.PrimaryDivision, input.WorkDivision, input.Role, input.Work, input.Description}
	}
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&result.ID, &result.UserID, &result.PrimaryDivision, &result.WorkDivision, &result.Number, &result.Role, &result.Work, &result.Description, &result.CompletedAt); err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

func (db *Database) DeleteReport(ctx context.Context, id, userID string) error {
	result, err := db.Conn.ExecContext(ctx, `DELETE FROM reports WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
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
