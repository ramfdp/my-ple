package db

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"my-ple-backend/model"
	"time"
)

var (
	ErrAttendanceInvalid  = errors.New("Data absensi tidak valid. Periksa mode dan lokasi perangkat.")
	ErrAttendanceOffice   = errors.New("Lokasi kantor belum dikonfigurasi. Hubungi administrator untuk absensi WFO.")
	ErrAttendanceOutside  = errors.New("Anda berada di luar radius 100 meter dari kantor.")
	ErrAttendanceAccuracy = errors.New("Lokasi belum cukup akurat. Aktifkan lokasi presisi dan coba lagi di area terbuka.")
	ErrAttendanceOpen     = errors.New("Selesaikan Clock Out absensi sebelumnya sebelum Clock In hari ini.")
	ErrAttendanceDate     = errors.New("Tanggal absensi harus valid dan tidak boleh melebihi hari ini.")
)

type AttendanceOffice struct {
	Latitude, Longitude float64
	Configured          bool
}

func attendanceCoordinate(value float64, limit float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Abs(value) <= limit
}

func (office AttendanceOffice) Ready() bool {
	return office.Configured && attendanceCoordinate(office.Latitude, 90) && attendanceCoordinate(office.Longitude, 180)
}

func (office AttendanceOffice) Validate(in model.AttendanceInput) error {
	if (in.Mode != "WFO" && in.Mode != "WFA") || in.Latitude == nil || in.Longitude == nil || in.Accuracy == nil {
		return ErrAttendanceInvalid
	}
	if !attendanceCoordinate(*in.Latitude, 90) || !attendanceCoordinate(*in.Longitude, 180) || math.IsNaN(*in.Accuracy) || math.IsInf(*in.Accuracy, 0) || *in.Accuracy < 0 {
		return ErrAttendanceInvalid
	}
	if in.Mode == "WFA" {
		return nil
	}
	if !office.Ready() {
		return ErrAttendanceOffice
	}
	if *in.Accuracy > 100 {
		return ErrAttendanceAccuracy
	}
	const radians = math.Pi / 180
	dlat, dlng := (*in.Latitude-office.Latitude)*radians, (*in.Longitude-office.Longitude)*radians
	a := math.Pow(math.Sin(dlat/2), 2) + math.Cos(office.Latitude*radians)*math.Cos(*in.Latitude*radians)*math.Pow(math.Sin(dlng/2), 2)
	distance := 2 * 6371000 * math.Asin(math.Sqrt(math.Min(1, a)))
	if distance > 100 {
		return ErrAttendanceOutside
	}
	return nil
}

func (db *Database) attendanceNow() time.Time {
	now := time.Now()
	if db.Now != nil {
		now = db.Now()
	}
	location, _ := time.LoadLocation(model.ReportingTimezone)
	return now.In(location)
}

const attendanceColumns = `id,attendance_date::text,mode,clock_in,clock_out,latitude,longitude`

func scanAttendance(row interface{ Scan(...any) error }) (model.Attendance, error) {
	var item model.Attendance
	err := row.Scan(&item.ID, &item.Date, &item.Mode, &item.ClockIn, &item.ClockOut, &item.Latitude, &item.Longitude)
	return item, err
}

func (db *Database) AttendanceOverview(ctx context.Context, date string) (model.AttendanceOverview, error) {
	today := db.attendanceNow().Format("2006-01-02")
	if date == "" {
		date = today
	}
	result := model.AttendanceOverview{Date: date, TodayDate: today, Employees: []model.AttendanceEmployee{}}
	day, err := time.Parse("2006-01-02", date)
	if err != nil || day.Year() < 1 || date > today {
		return result, ErrAttendanceDate
	}
	rows, err := db.Conn.QueryContext(ctx, `SELECT u.id,u.name,r.name,
 CASE WHEN a.id IS NOT NULL THEN 'PRESENT' WHEN l.types IS NOT NULL THEN 'LEAVE' ELSE 'MISSING' END,
 CASE WHEN a.id IS NULL THEN COALESCE(l.types,'') ELSE '' END,
 COALESCE(a.mode,''),a.clock_in,a.clock_out
 FROM users u JOIN roles r ON r.id=u.role_id
 LEFT JOIN attendance a ON a.user_id=u.id AND a.attendance_date=$1::date
 LEFT JOIN LATERAL (
   SELECT string_agg(DISTINCT request_type, ', ' ORDER BY request_type) AS types
   FROM leave_requests WHERE user_id=u.id AND status='APPROVED' AND $1::date BETWEEN start_date AND end_date
 ) l ON true
 WHERE u.is_active AND (u.created_at AT TIME ZONE $2)::date <= $1::date
 ORDER BY lower(u.name),u.id`, date, model.ReportingTimezone)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var employee model.AttendanceEmployee
		if err := rows.Scan(&employee.UserID, &employee.Name, &employee.Role, &employee.Status, &employee.LeaveTypes, &employee.Mode, &employee.ClockIn, &employee.ClockOut); err != nil {
			return result, err
		}
		result.Employees = append(result.Employees, employee)
	}
	return result, rows.Err()
}

func (db *Database) AttendanceHistory(ctx context.Context, userID, month string) (model.AttendanceHistory, error) {
	today := db.attendanceNow().Format("2006-01-02")
	result := model.AttendanceHistory{Records: []model.Attendance{}, TodayDate: today}
	if month == "" {
		month = today[:7]
	}
	start, err := time.Parse("2006-01", month)
	if err != nil || start.Year() < 1 || start.Year() > 9999 {
		return result, ErrAttendanceInvalid
	}
	rows, err := db.Conn.QueryContext(ctx, `SELECT `+attendanceColumns+` FROM attendance WHERE user_id=$1 AND ((attendance_date >= $2::date AND attendance_date < $3::date) OR attendance_date=$4::date OR clock_out IS NULL) ORDER BY attendance_date DESC`, userID, start.Format("2006-01-02"), start.AddDate(0, 1, 0).Format("2006-01-02"), today)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanAttendance(rows)
		if err != nil {
			return result, err
		}
		if item.Date[:7] == month {
			result.Records = append(result.Records, item)
		}
		if item.Date == today {
			result.Today = &item
		}
		if item.ClockOut == nil {
			result.Open = &item
		}
	}
	return result, rows.Err()
}

func (db *Database) ClockIn(ctx context.Context, userID string, in model.AttendanceInput, office AttendanceOffice) (model.Attendance, error) {
	var item model.Attendance
	if err := office.Validate(in); err != nil {
		return item, err
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return item, err
	}
	defer tx.Rollback()
	// Serialize attendance actions per employee, including requests crossing midnight.
	if err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&userID); err != nil {
		return item, err
	}
	now := db.attendanceNow()
	today := now.Format("2006-01-02")
	item, err = scanAttendance(tx.QueryRowContext(ctx, `SELECT `+attendanceColumns+` FROM attendance WHERE user_id=$1 AND (attendance_date=$2::date OR clock_out IS NULL) ORDER BY attendance_date DESC LIMIT 1`, userID, today))
	if err == nil {
		if item.Date != today {
			return item, ErrAttendanceOpen
		}
		return item, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return item, err
	}
	var lat, lng *float64
	if in.Mode == "WFA" {
		lat, lng = in.Latitude, in.Longitude
	}
	item, err = scanAttendance(tx.QueryRowContext(ctx, `INSERT INTO attendance(id,user_id,attendance_date,mode,clock_in,latitude,longitude) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+attendanceColumns, NewID(), userID, today, in.Mode, now, lat, lng))
	if err != nil {
		return item, err
	}
	return item, tx.Commit()
}

func (db *Database) ClockOut(ctx context.Context, userID, id string) (model.Attendance, error) {
	var item model.Attendance
	if !workspaceUUID(id) {
		return item, ErrAttendanceInvalid
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return item, err
	}
	defer tx.Rollback()
	if err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&userID); err != nil {
		return item, err
	}
	item, err = scanAttendance(tx.QueryRowContext(ctx, `UPDATE attendance SET clock_out=COALESCE(clock_out,GREATEST(clock_in,$3)) WHERE id=$1 AND user_id=$2 RETURNING `+attendanceColumns, id, userID, db.attendanceNow()))
	if err != nil {
		return item, err
	}
	return item, tx.Commit()
}
