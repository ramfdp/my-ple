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
	DB   *db.Database
	Send func(context.Context, string, string, model.ZoomBooking, bool) error
}

func (s *ZoomEmailWorker) Run(ctx context.Context) error {
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
	// ponytail: 50 sequential sends per poll; increase batch size if backlog grows.
	for i := 0; i < 50; i++ {
		var id, bookingID, recipient string
		var payload []byte
		var creator, eligible bool
		var attempt int
		err := s.DB.Conn.QueryRowContext(ctx, `UPDATE zoom_booking_email_deliveries SET status='sending',attempt_count=attempt_count+1
			WHERE id=(SELECT id FROM zoom_booking_email_deliveries WHERE status IN ('pending','failed') AND attempt_count<3 AND next_attempt_at<=NOW() ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1)
			RETURNING id,booking_id,recipient_email,is_creator,payload,attempt_count`).Scan(&id, &bookingID, &recipient, &creator, &payload, &attempt)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		// The committed claim prevents another worker/restart from resending mail
		// whose acceptance may be uncertain. Such rows require operator review.
		status, category := "sent", ""
		err = s.DB.Conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE is_active AND lower(btrim(email))=$1)`, recipient).Scan(&eligible)
		var booking model.ZoomBooking
		if err != nil {
			status, category = "failed", "recipient_lookup_failed"
		} else if !eligible || !email.ValidAddress(recipient) {
			status, category = "skipped", "recipient_no_longer_eligible"
		} else if json.Unmarshal(payload, &booking) != nil {
			status, category = "skipped", "invalid_payload"
		} else {
			err = send(ctx, recipient, id, booking, creator)
			if err != nil {
				status, category = "failed", "smtp_delivery_failed"
				if errors.Is(err, email.ErrUncertain) {
					status, category = "uncertain", "smtp_acceptance_uncertain"
				}
				// The production sender returns sanitized phase errors. Injected
				// senders may contain credentials, so never log arbitrary errors.
				if s.Send == nil && status != "uncertain" {
					category = err.Error()
				}
			}
		}
		log.Printf("zoom email booking=%s recipient=%s status=%s category=%s attempt=%d", bookingID, recipient, status, category, attempt)
		_, err = s.DB.Conn.ExecContext(ctx, `UPDATE zoom_booking_email_deliveries SET status=$2,last_error=$3,sent_at=CASE WHEN $2='sent' THEN NOW() ELSE NULL END,next_attempt_at=NOW()+($4 * interval '1 minute') WHERE id=$1`, id, status, category, attempt*attempt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ZoomEmailWorker) Start(ctx context.Context) {
	if !email.ZoomEnabled() {
		return
	}
	if err := email.ValidateZoomConfig(); err != nil {
		log.Printf("zoom email configuration: %v", err)
		return
	}
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
