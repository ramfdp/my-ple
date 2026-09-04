package realtime

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

type Hub struct {
	mu            sync.RWMutex
	broadcastMu   sync.Mutex
	connections   map[string]map[*websocket.Conn]struct{}
	allowedOrigin string
}

func NewHub(allowedOrigin string) *Hub {
	return &Hub{connections: make(map[string]map[*websocket.Conn]struct{}), allowedOrigin: allowedOrigin}
}

func (h *Hub) ServeHTTP(secret string, w http.ResponseWriter, r *http.Request) {
	protocol := r.Header.Get("Sec-WebSocket-Protocol")
	if !strings.HasPrefix(protocol, "auth.") {
		http.Error(w, "Autentikasi diperlukan.", http.StatusUnauthorized)
		return
	}
	claims, ok := middleware.AuthenticateToken(strings.TrimPrefix(protocol, "auth."), secret)
	if !ok {
		http.Error(w, "Autentikasi diperlukan.", http.StatusUnauthorized)
		return
	}
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == h.allowedOrigin }, Subprotocols: []string{protocol}}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	h.add(claims.ID, conn)
	expires := time.AfterFunc(time.Until(time.UnixMilli(claims.ExpiresAt)), func() { _ = conn.Close() })
	defer expires.Stop()
	defer func() { h.remove(claims.ID, conn); _ = conn.Close() }()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) add(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.connections[userID] == nil {
		h.connections[userID] = make(map[*websocket.Conn]struct{})
	}
	h.connections[userID][conn] = struct{}{}
}

func (h *Hub) remove(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.connections[userID], conn)
	if len(h.connections[userID]) == 0 {
		delete(h.connections, userID)
	}
}

func (h *Hub) Broadcast(notification model.Notification, excludeUserID string) {
	// ponytail: serialize writes; use per-connection writers if notification throughput grows.
	h.broadcastMu.Lock()
	defer h.broadcastMu.Unlock()
	payload, err := json.Marshal(model.NotificationEvent{Event: "notification.created", Notification: notification})
	if err != nil {
		return
	}
	h.mu.RLock()
	targets := make(map[string][]*websocket.Conn)
	for userID, connections := range h.connections {
		if userID == excludeUserID {
			continue
		}
		for conn := range connections {
			targets[userID] = append(targets[userID], conn)
		}
	}
	h.mu.RUnlock()
	for userID, connections := range targets {
		for _, conn := range connections {
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				h.remove(userID, conn)
				_ = conn.Close()
			}
		}
	}
}
