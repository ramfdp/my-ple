package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"my-ple-backend/model"
)

//go:embed schema.sql
var schemaSQL string

type Database struct {
	Conn *sql.DB
}

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
	result, err := db.Conn.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
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
	err := db.Conn.QueryRowContext(ctx, `
		WITH next_number AS (
			INSERT INTO report_sequences (primary_division, work_division, last_number)
			VALUES ($1, $2, 1)
			ON CONFLICT (primary_division, work_division)
			DO UPDATE SET last_number = report_sequences.last_number + 1
			RETURNING last_number
		)
		INSERT INTO reports (id, user_id, primary_division, work_division, number, description, completed_at)
		SELECT $3, $4, $1, $2, last_number, $5, NOW() FROM next_number
		RETURNING id, primary_division, work_division, number, description, completed_at`,
		input.PrimaryDivision, input.WorkDivision, NewID(), userID, input.Description,
	).Scan(&result.ID, &result.PrimaryDivision, &result.WorkDivision, &result.Number, &result.Description, &result.CompletedAt)
	return result, err
}

func (db *Database) ListReports(ctx context.Context, primaryDivision string) ([]model.WeeklyReport, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT id, primary_division, work_division, number, description, completed_at FROM reports WHERE primary_division = $1 ORDER BY work_division, number`, primaryDivision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reports []model.WeeklyReport
	for rows.Next() {
		var item model.WeeklyReport
		if err := rows.Scan(&item.ID, &item.PrimaryDivision, &item.WorkDivision, &item.Number, &item.Description, &item.CompletedAt); err != nil {
			return nil, err
		}
		reports = append(reports, item)
	}
	return reports, rows.Err()
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
