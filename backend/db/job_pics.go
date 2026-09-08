package db

import (
	"context"
	"database/sql"
	"my-ple-backend/model"
	"strings"
)

func (db *Database) PICUsers(ctx context.Context, search string) ([]model.JobPIC, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT id,name,is_active,username FROM users
 WHERE is_active AND (strpos(lower(name),$1)>0 OR strpos(lower(username),$1)>0 OR strpos(lower(email),$1)>0)
 ORDER BY lower(name),id LIMIT 50`, strings.ToLower(strings.TrimSpace(search)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []model.JobPIC{}
	for rows.Next() {
		var user model.JobPIC
		if err = rows.Scan(&user.UserID, &user.Name, &user.IsActive, &user.Username); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (db *Database) AddJobPIC(ctx context.Context, jobID int, userID, actorID string) error {
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var active bool
	if err = tx.QueryRowContext(ctx, `SELECT is_active FROM users WHERE id=$1 FOR SHARE`, userID).Scan(&active); err != nil {
		return err
	}
	if !active {
		return ErrInvalidReport
	}
	// Incremental insert preserves concurrent assignments of different users.
	// The composite primary key makes simultaneous/retried additions idempotent.
	if _, err = tx.ExecContext(ctx, `INSERT INTO job_pics(job_id,user_id,created_by) VALUES($1,$2,$3) ON CONFLICT(job_id,user_id) DO NOTHING`, jobID, userID, actorID); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *Database) RemoveJobPIC(ctx context.Context, jobID int, userID string) error {
	result, err := db.Conn.ExecContext(ctx, `DELETE FROM job_pics WHERE job_id=$1 AND user_id=$2`, jobID, userID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return sql.ErrNoRows
	}
	return err
}
