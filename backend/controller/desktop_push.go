package controller

import (
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/notification"
)

func DesktopPush(database *db.Database, w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method == http.MethodGet {
		w.Header().Set("Cache-Control", "no-store")
		WriteJSON(w, 200, map[string]string{"publicKey": notification.DesktopPublicKey()})
		return
	}
	cookie, err := r.Cookie(middleware.SessionCookie)
	if err != nil {
		http.Error(w, "Unauthorized", 401)
		return
	}
	var sub webpush.Subscription
	switch r.Method {
	case http.MethodPut:
		if notification.DesktopPublicKey() == "" {
			http.Error(w, "Desktop push unavailable", 503)
			return
		}
		input := struct {
			webpush.Subscription
			UserID string `json:"userId"`
		}{}
		if !DecodeJSON(w, r, &input) {
			return
		}
		if input.UserID != userID {
			http.Error(w, "Account changed", 409)
			return
		}
		sub = input.Subscription
		if !notification.ValidDesktopSubscription(sub) {
			http.Error(w, "Invalid push subscription", 400)
			return
		}
		result, err := database.Conn.ExecContext(r.Context(), `INSERT INTO desktop_subscriptions(id,session_id,endpoint,p256dh,auth)
   SELECT $1,id,$2,$3,$4 FROM sessions WHERE user_id=$5 AND token_hash=$6 AND revoked_at IS NULL AND expires_at>now()
   ON CONFLICT(endpoint) DO UPDATE SET session_id=EXCLUDED.session_id
   WHERE desktop_subscriptions.auth=EXCLUDED.auth AND desktop_subscriptions.p256dh=EXCLUDED.p256dh
   AND EXISTS (SELECT 1 FROM sessions old WHERE old.id=desktop_subscriptions.session_id AND old.user_id=$5)`,
			db.NewID(), sub.Endpoint, sub.Keys.P256dh, sub.Keys.Auth, userID, middleware.HashToken(cookie.Value))
		if err != nil {
			http.Error(w, "Subscription unavailable", 500)
			return
		}
		if n, _ := result.RowsAffected(); n == 0 {
			http.Error(w, "Subscription conflict", 409)
			return
		}
	case http.MethodDelete:
		if !DecodeJSON(w, r, &sub) {
			return
		}
		_, err = database.Conn.ExecContext(r.Context(), `DELETE FROM desktop_subscriptions d USING sessions s
   WHERE d.session_id=s.id AND s.user_id=$1 AND d.endpoint=$2`, userID, sub.Endpoint)
		if err != nil {
			http.Error(w, "Subscription unavailable", 500)
			return
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
