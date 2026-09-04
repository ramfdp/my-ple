package realtime

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

func TestHubAuthenticatesSupportsMultipleConnectionsAndExcludesActor(t *testing.T) {
	const secret = "test-secret"
	var hub *Hub
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hub.ServeHTTP(secret, w, r) }))
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dial := func(userID string) *websocket.Conn {
		token := middleware.SignToken(model.Claims{ID: userID, ExpiresAt: time.Now().Add(time.Minute).UnixMilli()}, secret)
		dialer := websocket.Dialer{Subprotocols: []string{"auth." + token}}
		conn, _, err := dialer.Dial(wsURL, http.Header{"Origin": []string{server.URL}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = conn.Close() })
		return conn
	}
	hub = NewHub(server.URL)
	actor := dial("actor")
	recipient1 := dial("recipient")
	recipient2 := dial("recipient")
	hub.Broadcast(model.Notification{ID: "event"}, "actor")
	for _, conn := range []*websocket.Conn{recipient1, recipient2} {
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		var event model.NotificationEvent
		if err := conn.ReadJSON(&event); err != nil || event.Notification.ID != "event" {
			t.Fatalf("event = %#v, %v", event, err)
		}
	}
	_ = actor.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	if _, _, err := actor.ReadMessage(); err == nil {
		t.Fatal("actor received its own notification")
	}
}

func TestHubRejectsUnauthenticatedConnection(t *testing.T) {
	hub := NewHub("http://example.test")
	r := httptest.NewRequest(http.MethodGet, "/ws/notifications", nil)
	w := httptest.NewRecorder()
	hub.ServeHTTP("secret", w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}
