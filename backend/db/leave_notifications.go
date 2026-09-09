package db

import (
	"context"
	"encoding/json"
	"log"
	"my-ple-backend/email"
	"my-ple-backend/model"
)

// Runs only after the business statement commits. Flags survive notification retention,
// and the existing outbox worker retries incomplete notification transactions.
func (db *Database) NotifyLeave(ctx context.Context, id string) ([]string, error) {
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var created, reviewed bool
	if err = tx.QueryRowContext(ctx, `SELECT created_notified,reviewed_notified FROM leave_requests WHERE id=$1 FOR UPDATE`, id).Scan(&created, &reviewed); err != nil {
		return nil, err
	}
	item, err := scanLeave(tx.QueryRowContext(ctx, leaveSelect+`WHERE l.id=$1`, id))
	if err != nil {
		return nil, err
	}
	recipients := []string{}
	stages := []string{"PENDING"}
	if item.Status != "PENDING" {
		stages = append(stages, item.Status)
	}
	for _, stage := range stages {
		stageQueued := true
		if (stage == "PENDING" && created) || (stage != "PENDING" && reviewed) {
			continue
		}
		actor, kind := item.UserID, "leave_request_created"
		payload := model.LeaveNotification{LeaveRequest: item, Path: "/leave/approvals?request=" + id}
		label := "Cuti"
		if item.RequestType == "IZIN" {
			label = "Izin"
		}
		payload.Title = "Pengajuan " + label + " Baru"
		payload.Message = item.EmployeeName + " mengajukan " + label
		payload.Status = stage
		if stage == "PENDING" {
			payload.ReviewedBy = nil
			payload.ReviewedAt = nil
			payload.ReviewerName = ""
			payload.ReviewerNote = ""
		} else {
			actor = *item.ReviewedBy
			kind = "leave_request_approved"
			result := "Disetujui"
			if stage == "REJECTED" {
				kind = "leave_request_rejected"
				result = "Ditolak"
			}
			payload.Title = "Pengajuan " + label + " " + result
			payload.Message = "Pengajuan Anda " + result + " oleh " + item.ReviewerName
			payload.Path = "/leave?request=" + id
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		event := NewID()
		if err = tx.QueryRowContext(ctx, `INSERT INTO notifications(id,type,actor_user_id,entity_type,entity_id,data) VALUES($1,$2,$3,'leave_request',$4,$5) ON CONFLICT(entity_id,type) WHERE entity_type='leave_request' DO UPDATE SET entity_id=EXCLUDED.entity_id RETURNING id`, event, kind, actor, id, string(data)).Scan(&event); err != nil {
			return nil, err
		}
		rows, err := tx.QueryContext(ctx, `SELECT u.id,COALESCE(lower(btrim(u.email)),'') FROM users u JOIN roles r ON r.id=u.role_id
 WHERE u.is_active AND (($2='PENDING' AND r.name IN ('Admin','Management') AND u.id<>$1) OR ($2<>'PENDING' AND u.id=$1))`, item.UserID, stage)
		if err != nil {
			return nil, err
		}
		type recipient struct{ id, address string }
		targets := []recipient{}
		for rows.Next() {
			var r recipient
			if err = rows.Scan(&r.id, &r.address); err != nil {
				rows.Close()
				return nil, err
			}
			targets = append(targets, r)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		addresses := map[string]bool{}
		for _, r := range targets {
			if _, err = tx.ExecContext(ctx, `INSERT INTO notification_recipients(notification_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, event, r.id); err != nil {
				return nil, err
			}
			recipients = append(recipients, r.id)
			if !email.ValidAddress(r.address) {
				log.Printf("email event_id=%s recipient_user_id=%s category=invalid_or_empty_email", event, r.id)
				continue
			}
			if addresses[r.address] {
				continue
			}
			addresses[r.address] = true
			if _, err = tx.ExecContext(ctx, `SAVEPOINT leave_email`); err != nil {
				return nil, err
			}
			outbox := NewID()
			if _, err = tx.ExecContext(ctx, `INSERT INTO zoom_booking_email_deliveries(id,recipient_email,is_creator,payload,notification_type,leave_request_id,recipient_user_id,notification_id)
 VALUES($1,$2,false,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, outbox, r.address, string(data), kind, id, r.id, event); err != nil {
				stageQueued = false
				log.Printf("email event_id=%s recipient_user_id=%s category=enqueue_failed", event, r.id)
				if _, err = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT leave_email`); err != nil {
					return nil, err
				}
			} else {
				log.Printf("email enqueue_attempt event_id=%s recipient_user_id=%s outbox_id=%s awaiting_commit=true", event, r.id, outbox)
			}
			if _, err = tx.ExecContext(ctx, `RELEASE SAVEPOINT leave_email`); err != nil {
				return nil, err
			}
		}
		if stage == "PENDING" {
			created = stageQueued
		} else {
			reviewed = stageQueued
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE leave_requests SET created_notified=$2,reviewed_notified=$3 WHERE id=$1`, id, created, reviewed); err != nil {
		return nil, err
	}
	return recipients, tx.Commit()
}
func (db *Database) RetryLeaveNotifications(ctx context.Context) error {
	rows, err := db.Conn.QueryContext(ctx, `SELECT id FROM leave_requests WHERE NOT created_notified OR (status<>'PENDING' AND NOT reviewed_notified) ORDER BY created_at LIMIT 50`)
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
		if _, err = db.NotifyLeave(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// Re-read recipients even on an idempotent retry: the worker may have persisted
// the notification between the request commit and the controller's enqueue call.
func (db *Database) LeaveRecipients(ctx context.Context, id, status string) ([]string, error) {
	kind := map[string]string{"PENDING": "leave_request_created", "APPROVED": "leave_request_approved", "REJECTED": "leave_request_rejected"}[status]
	rows, err := db.Conn.QueryContext(ctx, `SELECT nr.user_id FROM notification_recipients nr JOIN notifications n ON n.id=nr.notification_id WHERE n.entity_id=$1 AND n.type=$2 AND notification_visible(n.id,nr.user_id)`, id, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
