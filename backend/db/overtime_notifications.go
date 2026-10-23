package db

import (
	"context"
	"encoding/json"
	"log"
	"my-ple-backend/email"
	"my-ple-backend/model"
)

func (db *Database) NotifyOvertime(ctx context.Context, id string) ([]string, error) {
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var created, reviewed bool
	if err = tx.QueryRowContext(ctx, `SELECT created_notified,reviewed_notified FROM overtime_sheets WHERE id=$1 FOR UPDATE`, id).Scan(&created, &reviewed); err != nil {
		return nil, err
	}
	item, err := scanOvertimeSheet(tx.QueryRowContext(ctx, overtimeSheetSelect+`WHERE s.id=$1`, id))
	if err != nil {
		return nil, err
	}
	if item.Status == "DRAFT" {
		return nil, tx.Commit()
	}
	stageReviewed := item.Status == "APPROVED" || item.Status == "REJECTED"
	if (!stageReviewed && created) || (stageReviewed && reviewed) {
		return nil, tx.Commit()
	}
	kind, actor := "overtime_submitted", item.UserID
	payload := model.OvertimeNotification{SheetID: item.ID, EmployeeName: item.EmployeeName, Period: item.Period, Status: item.Status, Title: "Pengajuan Lembur Baru", Message: item.EmployeeName + " mengajukan lembur periode " + item.Period, Path: "/overtime/approvals"}
	if stageReviewed {
		if item.ReviewedBy == nil {
			return nil, ErrOvertimeInvalid
		}
		actor = *item.ReviewedBy
		kind = "overtime_approved"
		label := "disetujui"
		if item.Status == "REJECTED" {
			kind, label = "overtime_rejected", "dikembalikan untuk diperbaiki"
		}
		payload.Title = "Pengajuan Lembur " + label
		payload.Message = "Lembur periode " + item.Period + " " + label + " oleh " + item.ReviewerName
		payload.Path = "/overtime"
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	event := NewID()
	if _, err = tx.ExecContext(ctx, `INSERT INTO notifications(id,type,actor_user_id,entity_type,entity_id,data) VALUES($1,$2,$3,'overtime_sheet',$4,$5)`, event, kind, actor, id, string(data)); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT u.id,COALESCE(lower(btrim(u.email)),'') FROM users u JOIN roles r ON r.id=u.role_id WHERE u.is_active AND (($2=false AND r.name IN ('Finance','Admin') AND u.id<>$1) OR ($2=true AND u.id=$1))`, item.UserID, stageReviewed)
	if err != nil {
		return nil, err
	}
	type recipient struct{ id, address string }
	targets := []recipient{}
	for rows.Next() {
		var target recipient
		if err = rows.Scan(&target.id, &target.address); err != nil {
			rows.Close()
			return nil, err
		}
		targets = append(targets, target)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	recipients := []string{}
	for _, target := range targets {
		if _, err = tx.ExecContext(ctx, `INSERT INTO notification_recipients(notification_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, event, target.id); err != nil {
			return nil, err
		}
		recipients = append(recipients, target.id)
		if !email.ValidAddress(target.address) {
			continue
		}
		outbox := NewID()
		if _, err = tx.ExecContext(ctx, `INSERT INTO zoom_booking_email_deliveries(id,recipient_email,is_creator,payload,notification_type,overtime_sheet_id,recipient_user_id,notification_id) VALUES($1,$2,false,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, outbox, target.address, string(data), kind, id, target.id, event); err != nil {
			log.Printf("overtime email event_id=%s recipient_user_id=%s category=enqueue_failed", event, target.id)
			return nil, err
		}
	}
	if stageReviewed {
		created = true
		reviewed = true
	} else {
		created = true
	}
	if _, err = tx.ExecContext(ctx, `UPDATE overtime_sheets SET created_notified=$2,reviewed_notified=$3 WHERE id=$1`, id, created, reviewed); err != nil {
		return nil, err
	}
	return recipients, tx.Commit()
}

func (db *Database) RetryOvertimeNotifications(ctx context.Context) error {
	rows, err := db.Conn.QueryContext(ctx, `SELECT id FROM overtime_sheets WHERE status<>'DRAFT' AND (NOT created_notified OR (status IN ('APPROVED','REJECTED') AND NOT reviewed_notified)) ORDER BY updated_at LIMIT 50`)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err = db.NotifyOvertime(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
