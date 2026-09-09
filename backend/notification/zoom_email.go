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
	SendAssignment func(context.Context, string, string, model.WorkAssignmentEmail) error
}

func (s *ZoomEmailWorker) Run(ctx context.Context) error {
	if err := s.DB.RetryLeaveNotifications(ctx); err != nil {
		log.Printf("leave notification category=retry_failed")
	}
	if !email.ZoomEnabled() {
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
	// ponytail: 50 sequential sends per poll; increase batch size if backlog grows.
	for i := 0; i < 50; i++ {
		var id, bookingID, recipient, kind, taskID, userID, leaveID string
		var payload []byte
		var creator, eligible bool
		var attempt int
		err := s.DB.Conn.QueryRowContext(ctx, `UPDATE zoom_booking_email_deliveries SET status='sending',attempt_count=attempt_count+1
			WHERE id=(SELECT id FROM zoom_booking_email_deliveries WHERE status IN ('pending','failed') AND attempt_count<3 AND next_attempt_at<=NOW() ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1)
			RETURNING id,COALESCE(booking_id::text,''),recipient_email,is_creator,payload,attempt_count,notification_type,COALESCE(work_item_id::text,''),COALESCE(recipient_user_id::text,''),COALESCE(leave_request_id::text,'')`).Scan(&id, &bookingID, &recipient, &creator, &payload, &attempt, &kind, &taskID, &userID, &leaveID)
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
		if isLeave {
			err = s.DB.Conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN roles r ON r.id=u.role_id JOIN leave_requests l ON l.id=$3
 WHERE u.is_active AND lower(btrim(u.email))=$2 AND (( $4='leave_request_created' AND r.name IN ('Admin','Management') AND u.id<>l.user_id) OR ($4<>'leave_request_created' AND u.id=l.user_id AND u.id=$1)))`, userID, recipient, leaveID, kind).Scan(&eligible)
		} else if kind == "work_item_assignment" {
			err = s.DB.Conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN workspace_members m ON m.user_id=u.id
			JOIN work_items t ON t.division_id=m.division_id JOIN work_item_assignees a ON a.work_item_id=t.id AND a.user_id=u.id
			WHERE u.id=$1 AND lower(btrim(u.email))=$2 AND t.id=$3 AND t.archived_at IS NULL)`, userID, recipient, taskID).Scan(&eligible)
		} else {
			err = s.DB.Conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE is_active AND lower(btrim(email))=$1)`, recipient).Scan(&eligible)
		}
		var booking model.ZoomBooking
		var assignment model.WorkAssignmentEmail
		var leave model.LeaveNotification
		if err != nil {
			status, category = "failed", "recipient_lookup_failed"
		} else if !eligible || !email.ValidAddress(recipient) {
			status, category = "skipped", "recipient_no_longer_eligible"
		} else if (kind == "zoom_booking" && json.Unmarshal(payload, &booking) != nil) || (kind == "work_item_assignment" && json.Unmarshal(payload, &assignment) != nil) || (isLeave && json.Unmarshal(payload, &leave) != nil) || (!isLeave && kind != "zoom_booking" && kind != "work_item_assignment") {
			status, category = "skipped", "invalid_payload"
		} else {
			if isLeave {
				err = sendLeave(ctx, recipient, id, leave)
			} else if kind == "work_item_assignment" {
				err = sendAssignment(ctx, recipient, id, assignment)
			} else {
				err = send(ctx, recipient, id, booking, creator)
			}
			if err != nil {
				status, category = "failed", "smtp_delivery_failed"
				if errors.Is(err, email.ErrUncertain) {
					status, category = "uncertain", "smtp_acceptance_uncertain"
				}
				// The production sender returns sanitized phase errors. Injected
				// senders may contain credentials, so never log arbitrary errors.
				if ((kind == "zoom_booking" && s.Send == nil) || (kind == "work_item_assignment" && s.SendAssignment == nil)) && status != "uncertain" {
					category = err.Error()
				}
			}
		}
		log.Printf("outbox type=%s booking=%s task=%s status=%s category=%s attempt=%d", kind, bookingID, taskID, status, category, attempt)
		_, err = s.DB.Conn.ExecContext(ctx, `UPDATE zoom_booking_email_deliveries SET status=$2,last_error=$3,sent_at=CASE WHEN $2='sent' THEN NOW() ELSE NULL END,next_attempt_at=NOW()+($4 * interval '1 minute') WHERE id=$1`, id, status, category, attempt*attempt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ZoomEmailWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.Run(ctx); err != nil {
			log.Printf("zoom email worker category=delivery_processing_failed")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
