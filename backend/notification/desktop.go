package notification

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"my-ple-backend/db"
)

func DesktopPublicKey() string {
	private, err := base64.RawURLEncoding.DecodeString(os.Getenv("VAPID_PRIVATE_KEY"))
	if err != nil {
		return ""
	}
	key, err := ecdh.P256().NewPrivateKey(private)
	if err != nil || os.Getenv("VAPID_SUBJECT") == "" {
		return ""
	}
	public, err := base64.RawURLEncoding.DecodeString(os.Getenv("VAPID_PUBLIC_KEY"))
	if err != nil || !bytes.Equal(public, key.PublicKey().Bytes()) {
		return ""
	}
	return os.Getenv("VAPID_PUBLIC_KEY")
}

func ValidDesktopSubscription(sub webpush.Subscription) bool {
	u, err := url.Parse(sub.Endpoint)
	if err != nil || len(sub.Endpoint) > 2048 || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.Fragment != "" || len(u.Path) < 2 {
		return false
	}
	host := u.Hostname()
	if host != "fcm.googleapis.com" && host != "updates.push.services.mozilla.com" && host != "web.push.apple.com" && !strings.HasSuffix(host, ".notify.windows.com") {
		return false
	}
	auth, err := base64.RawURLEncoding.DecodeString(sub.Keys.Auth)
	if err != nil || len(auth) != 16 {
		return false
	}
	key, err := base64.RawURLEncoding.DecodeString(sub.Keys.P256dh)
	if err != nil {
		return false
	}
	_, err = ecdh.P256().NewPublicKey(key)
	return err == nil
}

var pushHTTP = &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

// RunDesktopPush leases one delivery at a time so another instance can recover a stopped worker.
func RunDesktopPush(ctx context.Context, database *db.Database) {
	public := DesktopPublicKey()
	if public == "" {
		log.Print("desktop push disabled: configure VAPID keys and subject")
		return
	}
	opts := webpush.Options{Subscriber: os.Getenv("VAPID_SUBJECT"), VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"), VAPIDPublicKey: public, HTTPClient: pushHTTP, TTL: 300}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		err := sendNextDesktop(ctx, database, &opts)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) && ctx.Err() == nil {
			log.Printf("desktop push queue: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func sendNextDesktop(ctx context.Context, database *db.Database, opts *webpush.Options) error {
	var id, event, kind, entity, workspace, language string
	var attempts int
	var allowed bool
	var sub webpush.Subscription
	err := database.Conn.QueryRowContext(ctx, `WITH due AS (
 SELECT subscription_id,event_key FROM desktop_deliveries WHERE next_attempt<=now()
 ORDER BY next_attempt LIMIT 1 FOR UPDATE SKIP LOCKED
 ), claim AS (
 UPDATE desktop_deliveries d SET next_attempt=now()+interval '1 minute',attempts=attempts+1
 FROM due WHERE (d.subscription_id,d.event_key)=(due.subscription_id,due.event_key) RETURNING d.*
 )
 SELECT p.id,c.event_key,c.attempts,p.endpoint,p.p256dh,p.auth,u.language,
 COALESCE(n.type,'attendance'),COALESCE(n.entity_id::text,''),COALESCE(n.data->>'workspaceId',''),
 u.is_active AND s.revoked_at IS NULL AND s.expires_at>now() AND
 CASE WHEN c.notification_id IS NOT NULL THEN notification_visible(n.id,u.id) AND n.created_at>=now()-interval '72 hours'
 ELSE c.reminder_date=(now() AT TIME ZONE 'Asia/Jakarta')::date
 AND NOT EXISTS(SELECT 1 FROM attendance WHERE user_id=u.id AND attendance_date=c.reminder_date)
 AND NOT EXISTS(SELECT 1 FROM leave_requests WHERE user_id=u.id AND status='APPROVED' AND c.reminder_date BETWEEN start_date AND end_date) END
 FROM claim c JOIN desktop_subscriptions p ON p.id=c.subscription_id JOIN sessions s ON s.id=p.session_id
 JOIN users u ON u.id=s.user_id LEFT JOIN notifications n ON n.id=c.notification_id`).Scan(&id, &event, &attempts, &sub.Endpoint, &sub.Keys.P256dh, &sub.Keys.Auth, &language, &kind, &entity, &workspace, &allowed)
	if err != nil {
		return err
	}
	if allowed {
		payload, _ := json.Marshal(map[string]string{"id": event, "kind": kind, "entity": entity, "workspace": workspace, "language": language})
		resp, sendErr := webpush.SendNotificationWithContext(ctx, payload, &sub, opts)
		status := 0
		if resp != nil {
			status = resp.StatusCode
			resp.Body.Close()
		}
		if status == 404 || status == 410 {
			_, err = database.Conn.ExecContext(ctx, "DELETE FROM desktop_subscriptions WHERE id=$1 AND endpoint=$2", id, sub.Endpoint)
			return err
		}
		if sendErr != nil || status < 200 || status >= 300 {
			if attempts < 6 && (sendErr != nil || status == 429 || status == 408 || status >= 500) {
				_, err = database.Conn.ExecContext(ctx, `UPDATE desktop_deliveries SET next_attempt=now()+$4*interval '10 seconds' WHERE subscription_id=$1 AND event_key=$2 AND attempts=$3`, id, event, attempts, 1<<attempts)
				return err
			}
			log.Printf("desktop push delivery stopped: status=%d attempts=%d", status, attempts)
		}
	}
	_, err = database.Conn.ExecContext(ctx, "DELETE FROM desktop_deliveries WHERE subscription_id=$1 AND event_key=$2 AND attempts=$3", id, event, attempts)
	return err
}
