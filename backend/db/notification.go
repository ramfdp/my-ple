package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"my-ple-backend/model"
)

func scanNotification(row interface{ Scan(...any) error }) (model.Notification, error) {
	var n model.Notification
	var data []byte
	err := row.Scan(&n.ID, &n.Type, &n.Actor.ID, &n.Actor.Name, &n.EntityType, &n.EntityID, &data, &n.CreatedAt, &n.ReadAt)
	if err == nil && n.Type == model.NotificationZoomBookingCreated {
		n.Booking = &model.NotificationBooking{}
		err = json.Unmarshal(data, n.Booking)
	}
	if err == nil && (n.Type == "workspace_task" || n.Type == model.NotificationWorkItemAssigned) {
		n.Workspace = &model.WorkspaceNotification{}
		err = json.Unmarshal(data, n.Workspace)
	}
	return n, err
}

func (db *Database) CreateZoomBookingWithNotification(ctx context.Context, userID string, input model.ZoomBookingInput, queueEmail ...bool) (model.ZoomBooking, model.Notification, error) {
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return model.ZoomBooking{}, model.Notification{}, err
	}
	defer tx.Rollback()
	booking, err := scanZoomBooking(tx.QueryRowContext(ctx, `
		WITH inserted AS (
			INSERT INTO zoom_bookings (id, user_id, zoom_resource_id, title, description, start_at, end_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *
		)
		SELECT `+zoomBookingSelect()+` FROM inserted z JOIN users u ON u.id=z.user_id JOIN zoom_resources r ON r.id=z.zoom_resource_id`,
		NewID(), userID, input.ZoomResourceID, input.Title, input.Description, input.StartAt, input.EndAt))
	if err != nil {
		return model.ZoomBooking{}, model.Notification{}, zoomConflict(err)
	}
	data, err := json.Marshal(model.NotificationBooking{ID: booking.ID, ZoomName: booking.ZoomResource.Name + " - " + booking.ZoomResource.Code, StartAt: booking.StartAt, EndAt: booking.EndAt, Description: booking.Description})
	if err != nil {
		return model.ZoomBooking{}, model.Notification{}, err
	}
	notification, err := scanNotification(tx.QueryRowContext(ctx, `
		WITH inserted AS (
			INSERT INTO notifications (id,type,actor_user_id,entity_type,entity_id,data)
			VALUES ($1,$2,$3,'zoom_booking',$4,$5) RETURNING *
		)
		SELECT n.id,n.type,n.actor_user_id,u.name,n.entity_type,n.entity_id,n.data,n.created_at,NULL::timestamptz
		FROM inserted n JOIN users u ON u.id=n.actor_user_id`, NewID(), model.NotificationZoomBookingCreated, userID, booking.ID, string(data)))
	if err != nil {
		return model.ZoomBooking{}, model.Notification{}, err
	}
	if len(queueEmail) > 0 && queueEmail[0] {
		if _, err = tx.ExecContext(ctx, "SAVEPOINT zoom_email"); err != nil {
			return model.ZoomBooking{}, model.Notification{}, err
		}
		if err = queueZoomEmail(ctx, tx, booking); err != nil {
			log.Printf("zoom email booking=%s category=enqueue_failed attempt=0", booking.ID)
			if _, err = tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT zoom_email"); err != nil {
				return model.ZoomBooking{}, model.Notification{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return model.ZoomBooking{}, model.Notification{}, err
	}
	return booking, notification, nil
}

func (db *Database) ListNotifications(ctx context.Context, userID string) ([]model.Notification, error) {
	rows, err := db.Conn.QueryContext(ctx, `
		SELECT n.id,n.type,n.actor_user_id,u.name,n.entity_type,n.entity_id,n.data,n.created_at,s.read_at
		FROM notifications n JOIN users u ON u.id=n.actor_user_id
		LEFT JOIN notification_user_states s ON s.notification_id=n.id AND s.user_id=$1
		WHERE notification_visible(n.id,$1) AND n.created_at>=NOW()-INTERVAL '72 hours' AND s.dismissed_at IS NULL
		ORDER BY n.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.Notification{}
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	return items, rows.Err()
}

func (db *Database) UnreadNotificationCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := db.Conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications n
		LEFT JOIN notification_user_states s ON s.notification_id=n.id AND s.user_id=$1
		WHERE notification_visible(n.id,$1) AND n.created_at>=NOW()-INTERVAL '72 hours' AND s.read_at IS NULL AND s.dismissed_at IS NULL`, userID).Scan(&count)
	return count, err
}

func (db *Database) MarkNotificationRead(ctx context.Context, notificationID, userID string) error {
	result, err := db.Conn.ExecContext(ctx, `INSERT INTO notification_user_states(notification_id,user_id,read_at)
		SELECT id,$2,NOW() FROM notifications WHERE notification_visible(id,$2) AND id=$1 AND created_at>=NOW()-INTERVAL '72 hours'
		ON CONFLICT(notification_id,user_id) DO UPDATE SET read_at=COALESCE(notification_user_states.read_at,EXCLUDED.read_at)`, notificationID, userID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (db *Database) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	_, err := db.Conn.ExecContext(ctx, `INSERT INTO notification_user_states(notification_id,user_id,read_at)
		SELECT id,$1,NOW() FROM notifications WHERE notification_visible(id,$1) AND created_at>=NOW()-INTERVAL '72 hours'
		ON CONFLICT(notification_id,user_id) DO UPDATE SET read_at=COALESCE(notification_user_states.read_at,EXCLUDED.read_at)
		WHERE notification_user_states.dismissed_at IS NULL`, userID)
	return err
}

func (db *Database) DismissNotification(ctx context.Context, notificationID, userID string) error {
	result, err := db.Conn.ExecContext(ctx, `INSERT INTO notification_user_states(notification_id,user_id,dismissed_at)
		SELECT id,$2,NOW() FROM notifications WHERE notification_visible(id,$2) AND id=$1 AND created_at>=NOW()-INTERVAL '72 hours'
		ON CONFLICT(notification_id,user_id) DO UPDATE SET dismissed_at=EXCLUDED.dismissed_at`, notificationID, userID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (db *Database) DeleteExpiredNotifications(ctx context.Context) (int64, error) {
	result, err := db.Conn.ExecContext(ctx, `DELETE FROM notifications WHERE created_at<NOW()-INTERVAL '72 hours'`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
