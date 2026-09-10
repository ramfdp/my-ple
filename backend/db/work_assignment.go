package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"my-ple-backend/email"
	"my-ple-backend/model"
)

func workAssignmentNotification(ctx context.Context, tx *sql.Tx, id, actor string, added []string) error {
	var payload model.WorkAssignmentEmail
	var version int
	err := tx.QueryRowContext(ctx, `SELECT t.title,t.description,t.division_id,COALESCE(j.name,'Tanpa project'),u.name,
 COALESCE(t.start_date::text,''),COALESCE(t.due_date::text,''),t.priority,t.version
 FROM work_items t LEFT JOIN jobs j ON j.id=t.job_id JOIN users u ON u.id=$2 WHERE t.id=$1`, id, actor).
		Scan(&payload.Title, &payload.Description, &payload.WorkspaceID, &payload.ProjectName, &payload.AssignedByName, &payload.StartDate, &payload.DueDate, &payload.Priority, &version)
	if err != nil {
		return err
	}
	payload.TaskID = id
	if err = tx.QueryRowContext(ctx, `SELECT name FROM divisions WHERE id=$1`, payload.WorkspaceID).Scan(&payload.WorkspaceName); err != nil {
		return err
	}
	payload.Message = "Anda ditugaskan ke pekerjaan baru"
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	event := NewID()
	if _, err = tx.ExecContext(ctx, `INSERT INTO notifications(id,type,actor_user_id,entity_type,entity_id,workspace_division_id,data) VALUES($1,$2,$3,'work_item',$4,$5,$6)`, event, model.NotificationWorkItemAssigned, actor, id, payload.WorkspaceID, string(data)); err != nil {
		return err
	}
	for _, recipient := range added {
		if _, err = tx.ExecContext(ctx, `INSERT INTO notification_recipients(notification_id,user_id) VALUES($1,$2)`, event, recipient); err != nil {
			return err
		}
	}
	for _, recipient := range added {
		var address string
		err = tx.QueryRowContext(ctx, `SELECT COALESCE(lower(btrim(email)),'') FROM users WHERE id=$1`, recipient).Scan(&address)
		if err != nil {
			return err
		}
		status, reason := "pending", ""
		if !email.Enabled() {
			reason = "email_disabled"
		}
		if !email.ValidAddress(address) {
			status, reason = "skipped", "invalid_or_empty_email"
		}
		outbox := NewID()
		_, err = tx.ExecContext(ctx, `INSERT INTO zoom_booking_email_deliveries(id,recipient_email,is_creator,payload,notification_type,work_item_id,recipient_user_id,assignment_version,notification_id,status,last_error)
  VALUES($1,$2,false,$3,'work_item_assignment',$4,$5,$6,$7,$8,$9) ON CONFLICT(notification_id,recipient_user_id) DO NOTHING`, outbox, address, string(data), id, recipient, version, event, status, reason)
		if err != nil {
			return err
		}
		log.Printf("email enqueue event_id=%s recipient_user_id=%s outbox_id=%s awaiting_commit=true", event, recipient, outbox)
	}
	return err
}
