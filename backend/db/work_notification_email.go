package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"my-ple-backend/email"
	"my-ple-backend/model"
)

// Email uses exactly the persisted website recipients, never a role broadcast.
func queueWorkNotificationEmail(ctx context.Context, tx *sql.Tx, event string) error {
	if !email.Enabled() {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `SAVEPOINT work_notification_email`); err != nil {
		return err
	}
	err := insertWorkNotificationEmail(ctx, tx, event)
	if err != nil {
		log.Printf("email event_id=%s category=enqueue_failed", event)
		_, err = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT work_notification_email`)
	}
	return err
}

func insertWorkNotificationEmail(ctx context.Context, tx *sql.Tx, event string) error {
	var item string
	var payload model.WorkAssignmentEmail
	err := tx.QueryRowContext(ctx, `SELECT t.id,t.title,t.description,t.division_id,COALESCE(j.name,'Tanpa project'),u.name,COALESCE(t.start_date::text,''),COALESCE(t.due_date::text,''),t.priority,n.data->>'message'
 FROM notifications n JOIN work_items t ON t.id=n.entity_id JOIN users u ON u.id=n.actor_user_id LEFT JOIN jobs j ON j.id=t.job_id WHERE n.id=$1`, event).Scan(&item, &payload.Title, &payload.Description, &payload.WorkspaceID, &payload.ProjectName, &payload.AssignedByName, &payload.StartDate, &payload.DueDate, &payload.Priority, &payload.Message)
	if err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT u.id,COALESCE(u.email,'') FROM notification_recipients nr JOIN users u ON u.id=nr.user_id WHERE nr.notification_id=$1 AND u.is_active ORDER BY u.id`, event)
	if err != nil {
		return err
	}
	type recipient struct{ id, address string }
	targets := []recipient{}
	for rows.Next() {
		var r recipient
		if err = rows.Scan(&r.id, &r.address); err != nil {
			rows.Close()
			return err
		}
		targets = append(targets, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, r := range targets {
		address := email.NormalizeAddress(r.address)
		if !email.ValidAddress(address) {
			log.Printf("email event_id=%s recipient_user_id=%s category=invalid_or_empty_email", event, r.id)
			continue
		}
		if seen[address] {
			continue
		}
		seen[address] = true
		outbox := NewID()
		if _, err = tx.ExecContext(ctx, `INSERT INTO zoom_booking_email_deliveries(id,recipient_email,is_creator,payload,notification_type,work_item_id,recipient_user_id,notification_id) VALUES($1,$2,false,$3,'workspace_task',$4,$5,$6) ON CONFLICT(notification_id,recipient_user_id) DO NOTHING`, outbox, address, string(data), item, r.id, event); err != nil {
			return err
		}
		log.Printf("email enqueue event_id=%s recipient_user_id=%s outbox_id=%s awaiting_commit=true", event, r.id, outbox)
	}
	return nil
}
