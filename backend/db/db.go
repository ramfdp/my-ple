package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"my-ple-backend/model"
)

type Database struct {
	Conn *sql.DB
}

func NewDatabase(conn *sql.DB) *Database {
	return &Database{Conn: conn}
}

func (db *Database) FindUser(ctx context.Context, username string) (model.User, error) {
	var u model.User
	err := db.Conn.QueryRowContext(ctx, `SELECT id, username, display_name, password_hash FROM users WHERE username = $1`, username).Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash)
	return u, err
}

func (db *Database) CreateUser(ctx context.Context, u model.User) (model.User, error) {
	err := db.Conn.QueryRowContext(ctx, `INSERT INTO users (id, username, display_name, password_hash) VALUES ($1, $2, $3, $4) RETURNING id, username, display_name`, u.ID, u.Username, u.DisplayName, u.PasswordHash).Scan(&u.ID, &u.Username, &u.DisplayName)
	return u, err
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
