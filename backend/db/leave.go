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

var ErrLeaveInvalid = errors.New("Data pengajuan tidak valid. Periksa jenis, tanggal, dan alasan.")
var ErrLeaveForbidden = errors.New("Anda tidak berhak mengakses atau meninjau pengajuan ini.")
var ErrLeaveConflict = errors.New("Pengajuan sudah diproses atau kunci pengajuan telah digunakan. Muat ulang data.")

func ValidateLeave(in *model.LeaveInput) error {
	in.Reason = strings.TrimSpace(in.Reason)
	in.Notes = strings.TrimSpace(in.Notes)
	if !workspaceUUID(in.RequestID) || (in.RequestType != "CUTI" && in.RequestType != "IZIN") || in.Reason == "" || len(in.Reason) > 4000 || len(in.Notes) > 4000 {
		return ErrLeaveInvalid
	}
	for _, d := range []string{in.StartDate, in.EndDate} {
		if parsed, err := time.Parse("2006-01-02", d); err != nil || parsed.Year() < 1 {
			return ErrLeaveInvalid
		}
	}
	if in.EndDate < in.StartDate {
		return ErrLeaveInvalid
	}
	return nil
}
func (db *Database) leaveAccess(ctx context.Context, userID string, review bool) error {
	var allowed bool
	err := db.Conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN roles r ON r.id=u.role_id WHERE u.id=$1 AND u.is_active AND (NOT $2 OR r.name IN ('Admin','Management')))`, userID, review).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrLeaveForbidden
	}
	return nil
}

const leaveSelect = `SELECT jsonb_build_object('id',l.id,'userId',l.user_id,'employeeName',u.name,
 'division',COALESCE((SELECT d.name FROM workspace_members m JOIN divisions d ON d.id=m.division_id WHERE m.user_id=u.id LIMIT 1),''),
 'requestType',l.request_type,'startDate',l.start_date::text,'endDate',l.end_date::text,'reason',l.reason,'notes',l.notes,'status',l.status,
 'reviewedBy',l.reviewed_by,'reviewerName',COALESCE(r.name,''),'reviewedAt',l.reviewed_at,'reviewerNote',l.reviewer_note,'createdAt',l.created_at,'updatedAt',l.updated_at)
 FROM leave_requests l JOIN users u ON u.id=l.user_id LEFT JOIN users r ON r.id=l.reviewed_by `

func scanLeave(row interface{ Scan(...any) error }) (model.LeaveRequest, error) {
	var item model.LeaveRequest
	var raw []byte
	err := row.Scan(&raw)
	if err == nil {
		err = json.Unmarshal(raw, &item)
	}
	return item, err
}
func (db *Database) LeaveRequest(ctx context.Context, userID, id string, review bool) (model.LeaveRequest, error) {
	if !workspaceUUID(id) {
		return model.LeaveRequest{}, ErrLeaveInvalid
	}
	if err := db.leaveAccess(ctx, userID, review); err != nil {
		return model.LeaveRequest{}, err
	}
	return scanLeave(db.Conn.QueryRowContext(ctx, leaveSelect+`WHERE l.id=$1 AND ($3 OR l.user_id=$2)`, id, userID, review))
}
func (db *Database) LeaveRequests(ctx context.Context, userID string, review bool, status, kind string, offset int) ([]model.LeaveRequest, error) {
	if err := db.leaveAccess(ctx, userID, review); err != nil {
		return nil, err
	}
	if (status != "" && status != "PENDING" && status != "APPROVED" && status != "REJECTED") || (kind != "" && kind != "CUTI" && kind != "IZIN") || offset < 0 || offset > 1000000 {
		return nil, ErrLeaveInvalid
	}
	rows, err := db.Conn.QueryContext(ctx, leaveSelect+`WHERE ($2 OR l.user_id=$1) AND ($3='' OR l.status=$3) AND ($4='' OR l.request_type=$4) ORDER BY l.created_at DESC,l.id DESC LIMIT 50 OFFSET $5`, userID, review, status, kind, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.LeaveRequest{}
	for rows.Next() {
		item, err := scanLeave(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (db *Database) CreateLeave(ctx context.Context, userID string, in model.LeaveInput) (model.LeaveRequest, error) {
	if err := ValidateLeave(&in); err != nil {
		return model.LeaveRequest{}, err
	}
	if err := db.leaveAccess(ctx, userID, false); err != nil {
		return model.LeaveRequest{}, err
	}
	id := NewID()
	err := db.Conn.QueryRowContext(ctx, `INSERT INTO leave_requests(id,user_id,request_id,request_type,start_date,end_date,reason,notes)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(user_id,request_id) DO NOTHING RETURNING id`, id, userID, in.RequestID, in.RequestType, in.StartDate, in.EndDate, in.Reason, in.Notes).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		err = db.Conn.QueryRowContext(ctx, `SELECT id FROM leave_requests WHERE user_id=$1 AND request_id=$2 AND request_type=$3 AND start_date=$4 AND end_date=$5 AND reason=$6 AND notes=$7`, userID, in.RequestID, in.RequestType, in.StartDate, in.EndDate, in.Reason, in.Notes).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return model.LeaveRequest{}, ErrLeaveConflict
		}
	}
	if err != nil {
		return model.LeaveRequest{}, err
	}
	return db.LeaveRequest(ctx, userID, id, false)
}
func (db *Database) DecideLeave(ctx context.Context, userID, id string, in model.LeaveDecision) (model.LeaveRequest, error) {
	if err := db.leaveAccess(ctx, userID, true); err != nil {
		return model.LeaveRequest{}, err
	}
	if !workspaceUUID(id) || (in.Status != "APPROVED" && in.Status != "REJECTED") || len(in.ReviewerNote) > 4000 {
		return model.LeaveRequest{}, ErrLeaveInvalid
	}
	item, err := db.LeaveRequest(ctx, userID, id, true)
	if err != nil {
		return item, err
	}
	if item.UserID == userID {
		return item, ErrLeaveForbidden
	}
	result, err := db.Conn.ExecContext(ctx, `UPDATE leave_requests SET status=$3,reviewed_by=$2,reviewed_at=NOW(),reviewer_note=$4,updated_at=NOW()
 WHERE id=$1 AND status='PENDING' AND user_id<>$2 AND EXISTS(SELECT 1 FROM users u JOIN roles r ON r.id=u.role_id WHERE u.id=$2 AND u.is_active AND r.name IN ('Admin','Management'))`, id, userID, in.Status, strings.TrimSpace(in.ReviewerNote))
	if err != nil {
		return item, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return item, err
	}
	if n != 1 {
		return item, ErrLeaveConflict
	}
	return db.LeaveRequest(ctx, userID, id, true)
}
