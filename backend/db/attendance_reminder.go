package db

import (
	"context"
	"database/sql"
	"my-ple-backend/model"
)

type UserMissingAttendance struct {
	ID    string
	Email string
	Name  string
}

// GetUsersNotClockInToday returns active users (non-deleted, is_active=true) who haven't clocked in today
// Excludes users on approved leave for today. Only returns on weekdays (Mon-Fri).
func (db *Database) GetUsersNotClockInToday(ctx context.Context) ([]UserMissingAttendance, error) {
	now := db.attendanceNow()
	today := now.Format("2006-01-02")
	
	// Only trigger on weekdays (1=Mon, 5=Fri)
	weekday := int(now.Weekday())
	if weekday == 0 || weekday == 6 {
		return []UserMissingAttendance{}, nil // Skip Sunday & Saturday
	}

	rows, err := db.Conn.QueryContext(ctx, `
		SELECT u.id, u.email, u.name
		FROM users u
		WHERE u.is_active
		  AND u.created_at AT TIME ZONE $2 <= CURRENT_TIMESTAMP AT TIME ZONE $2
		  AND NOT EXISTS (
		    SELECT 1 FROM attendance a 
		    WHERE a.user_id = u.id AND a.attendance_date = $1::date
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM leave_requests lr
		    WHERE lr.user_id = u.id 
		      AND lr.status = 'APPROVED'
		      AND $1::date BETWEEN lr.start_date AND lr.end_date
		  )
		ORDER BY u.name, u.id
	`, today, model.ReportingTimezone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []UserMissingAttendance
	for rows.Next() {
		var user UserMissingAttendance
		if err := rows.Scan(&user.ID, &user.Email, &user.Name); err != nil {
			return nil, err
		}
		result = append(result, user)
	}
	return result, rows.Err()
}

// HasReminderBeenSentToday checks if reminder has already been sent today
func (db *Database) HasReminderBeenSentToday(ctx context.Context) (bool, error) {
	today := db.attendanceNow().Format("2006-01-02")
	var exists bool
	err := db.Conn.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM attendance_reminders_log WHERE reminder_date = $1::date)
	`, today).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}

// RecordReminderSent marks that reminder was sent today
func (db *Database) RecordReminderSent(ctx context.Context) error {
	today := db.attendanceNow().Format("2006-01-02")
	_, err := db.Conn.ExecContext(ctx, `
		INSERT INTO attendance_reminders_log (reminder_date, sent_at) 
		VALUES ($1::date, CURRENT_TIMESTAMP)
		ON CONFLICT (reminder_date) DO UPDATE SET sent_at = CURRENT_TIMESTAMP
	`, today)
	return err
}
