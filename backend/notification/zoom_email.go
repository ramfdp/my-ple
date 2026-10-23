package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"time"

	"my-ple-backend/db"
	"my-ple-backend/email"
	"my-ple-backend/model"
)

// Reuses the existing SMTP transport and durable-claim semantics, without
// coupling Zoom delivery to Weekly Report's archive-specific queue.
type ZoomEmailWorker struct {
	DB             *db.Database
	Send           func(context.Context, string, string, model.ZoomBooking, bool) error
	SendLeave      func(context.Context, string, string, model.LeaveNotification) error
	SendOvertime   func(context.Context, string, string, model.OvertimeNotification) error
	SendAssignment func(context.Context, string, string, model.WorkAssignmentEmail) error
}

func (s *ZoomEmailWorker) Run(ctx context.Context) error {
	if err := s.DB.RetryLeaveNotifications(ctx); err != nil {
		log.Printf("leave notification category=retry_failed")
	}
	if err := s.DB.RetryOvertimeNotifications(ctx); err != nil {
		log.Printf("overtime notification category=retry_failed")
	}
	// A lost SMTP acknowledgement cannot safely be replayed automatically.
	if _, err := s.DB.Conn.ExecContext(ctx, `UPDATE zoom_booking_email_deliveries SET status='uncertain',last_error='worker_interrupted_acceptance_unknown' WHERE status='sending' AND COALESCE(claimed_at,created_at)<NOW()-interval '5 minutes'`); err != nil {
		return err
	}
	if !email.Enabled() {
		return nil
	}
	if err := email.ValidateZoomConfig(); err != nil {
		return err
	}
	send := s.Send
	if send == nil {
		send = email.SendZoom
	}
	sendAssignment := s.SendAssignment
	if sendAssignment == nil {
		sendAssignment = email.SendWorkAssignment
	}
	sendLeave := s.SendLeave
	if sendLeave == nil {
		sendLeave = email.SendLeave
	}
	sendOvertime := s.SendOvertime
	if sendOvertime == nil {
		sendOvertime = email.SendOvertime
	}
	// ponytail: 50 sequential sends per poll; increase batch size if backlog grows.
	for i := 0; i < 50; i++ {
		var id, bookingID, recipient, kind, taskID, userID, leaveID, overtimeID string
		var payload []byte
		var creator, eligible bool
		var attempt int
		err := s.DB.Conn.QueryRowContext(ctx, `UPDATE zoom_booking_email_deliveries SET status='sending',claimed_at=NOW(),attempt_count=attempt_count+1
			WHERE id=(SELECT id FROM zoom_booking_email_deliveries WHERE status IN ('pending','failed') AND attempt_count<3 AND next_attempt_at<=NOW() ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1)
			RETURNING id,COALESCE(booking_id::text,''),recipient_email,is_creator,payload,attempt_count,notification_type,COALESCE(work_item_id::text,''),COALESCE(recipient_user_id::text,''),COALESCE(leave_request_id::text,''),COALESCE(overtime_sheet_id::text,'')`).Scan(&id, &bookingID, &recipient, &creator, &payload, &attempt, &kind, &taskID, &userID, &leaveID, &overtimeID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		// The committed claim prevents another worker/restart from resending mail
		// whose acceptance may be uncertain. Such rows require operator review.
		status, category := "sent", ""
		isLeave := kind == "leave_request_created" || kind == "leave_request_approved" || kind == "leave_request_rejected"
		isOvertime := kind == "overtime_submitted" || kind == "overtime_approved" || kind == "overtime_rejected"
		// Identity is authoritative: refresh the current address just before SMTP.
		var current string
		err = s.DB.Conn.QueryRowContext(ctx, `SELECT COALESCE(lower(btrim(u.email)),''),u.id FROM users u
 WHERE u.is_active AND (u.id::text=$1 OR ($1='' AND lower(btrim(u.email))=$2 AND $3='zoom_booking' AND (SELECT count(*) FROM users legacy WHERE lower(btrim(legacy.email))=$2)=1))
 AND CASE
 WHEN $3='leave_request_created' THEN EXISTS(SELECT 1 FROM roles r,leave_requests l WHERE r.id=u.role_id AND r.name IN ('Admin','Management') AND l.id::text=$4 AND l.user_id<>u.id)
 WHEN $3 IN ('leave_request_approved','leave_request_rejected') THEN EXISTS(SELECT 1 FROM leave_requests l WHERE l.id::text=$4 AND l.user_id=u.id)
 WHEN $3 IN ('work_item_assignment','workspace_task') THEN EXISTS(SELECT 1 FROM work_items t JOIN workspace_access a ON a.division_id=t.division_id AND a.user_id=u.id WHERE t.id::text=$5 AND t.archived_at IS NULL AND
 (($3='work_item_assignment' AND EXISTS(SELECT 1 FROM work_item_assignees wa WHERE wa.work_item_id=t.id AND wa.user_id=u.id)) OR
 ($3='workspace_task')))
 WHEN $3='overtime_submitted' THEN EXISTS(SELECT 1 FROM roles r,overtime_sheets o WHERE r.id=u.role_id AND r.name IN ('Finance','Admin') AND o.id::text=$6 AND o.user_id<>u.id AND o.status='SUBMITTED')
 WHEN $3='overtime_approved' THEN EXISTS(SELECT 1 FROM overtime_sheets o WHERE o.id::text=$6 AND o.user_id=u.id AND o.status='APPROVED')
 WHEN $3='overtime_rejected' THEN EXISTS(SELECT 1 FROM overtime_sheets o WHERE o.id::text=$6 AND o.user_id=u.id AND o.status='REJECTED')
 ELSE $3='zoom_booking' END LIMIT 1`, userID, recipient, kind, leaveID, taskID, overtimeID).Scan(&current, &userID)
		eligible = err == nil
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
		}
		if eligible {
			recipient = current
			_, err = s.DB.Conn.ExecContext(ctx, `UPDATE zoom_booking_email_deliveries SET recipient_email=$2,recipient_user_id=$3 WHERE id=$1`, id, recipient, userID)
		}

		var booking model.ZoomBooking
		var assignment model.WorkAssignmentEmail
		var leave model.LeaveNotification
		var overtime model.OvertimeNotification
		if err != nil {
			status, category = "failed", "recipient_lookup_failed"
		} else if !eligible || !email.ValidAddress(recipient) {
			status, category = "skipped", "recipient_no_longer_eligible"
		} else if (kind == "zoom_booking" && json.Unmarshal(payload, &booking) != nil) || ((kind == "work_item_assignment" || kind == "workspace_task") && json.Unmarshal(payload, &assignment) != nil) || (isLeave && json.Unmarshal(payload, &leave) != nil) || (isOvertime && json.Unmarshal(payload, &overtime) != nil) || (!isLeave && !isOvertime && kind != "zoom_booking" && kind != "work_item_assignment" && kind != "workspace_task") {
			status, category = "skipped", "invalid_payload"
		} else {
			if isLeave {
				err = sendLeave(ctx, recipient, id, leave)
			} else if isOvertime {
				err = sendOvertime(ctx, recipient, id, overtime)
			} else if kind == "work_item_assignment" || kind == "workspace_task" {
				err = sendAssignment(ctx, recipient, id, assignment)
			} else {
				err = send(ctx, recipient, id, booking, creator)
			}
			if err != nil {
				status, category = email.Failure(err)
			}
		}
		log.Printf("email outbox_id=%s type=%s booking=%s task=%s recipient_user_id=%s recipient=%s status=%s category=%s attempt=%d", id, kind, bookingID, taskID, userID, email.MaskAddress(recipient), status, category, attempt)
		resultCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		_, err = s.DB.Conn.ExecContext(resultCtx, `UPDATE zoom_booking_email_deliveries SET status=$2,last_error=$3,sent_at=CASE WHEN $2='sent' THEN NOW() ELSE NULL END,next_attempt_at=NOW()+($4 * interval '1 minute') WHERE id=$1`, id, status, category, attempt*attempt)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ZoomEmailWorker) Start(ctx context.Context) {
	log.Printf("email worker started enabled=%t poll=10s", email.Enabled())
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.Run(ctx); err != nil {
			log.Printf("email worker category=delivery_processing_failed")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
