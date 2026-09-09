package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"my-ple-backend/email"
	"my-ple-backend/model"
)

func queueZoomEmail(ctx context.Context, tx *sql.Tx, booking model.ZoomBooking, eventID string) error {
	rows, err := tx.QueryContext(ctx, `SELECT lower(btrim(email)), bool_or(id=$1), (array_agg(id ORDER BY (id=$1) DESC,id))[1] FROM users WHERE is_active GROUP BY lower(btrim(email))`, booking.UserID)
	if err != nil {
		return err
	}
	type recipient struct {
		address string
		id      string
		creator bool
	}
	recipients := []recipient{}
	for rows.Next() {
		var r recipient
		var address sql.NullString
		if err = rows.Scan(&address, &r.creator, &r.id); err != nil {
			rows.Close()
			return err
		}
		r.address = address.String
		if !email.ValidAddress(r.address) {
			log.Printf("email event_id=%s recipient_user_id=%s category=invalid_or_empty_email", eventID, r.id)
			continue
		}
		recipients = append(recipients, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	// Snapshot only fields needed by the email, never the resource host key.
	snapshot := model.ZoomBooking{CreatedBy: booking.CreatedBy, Title: booking.Title, Description: booking.Description, StartAt: booking.StartAt, EndAt: booking.EndAt, ZoomResource: model.ZoomResource{Name: booking.ZoomResource.Name, Code: booking.ZoomResource.Code, MeetingURL: booking.ZoomResource.MeetingURL, Passcode: booking.ZoomResource.Passcode}}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	for _, r := range recipients {
		outbox := NewID()
		if _, err = tx.ExecContext(ctx, `INSERT INTO zoom_booking_email_deliveries(id,booking_id,recipient_email,is_creator,payload,recipient_user_id,notification_id) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(booking_id,recipient_email) DO NOTHING`, outbox, booking.ID, r.address, r.creator, string(payload), r.id, eventID); err != nil {
			return err
		}
		log.Printf("email enqueue event_id=%s recipient_user_id=%s outbox_id=%s awaiting_commit=true", eventID, r.id, outbox)
	}
	return nil
}
