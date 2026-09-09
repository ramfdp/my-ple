package realtime

import (
	"encoding/json"
	"net/http"
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

func (h *Hub) ServeHTTP(database middleware.SessionStore, w http.ResponseWriter, r *http.Request) {
	claims, expiresAt, ok := middleware.Authenticated(r, database)
	if !ok {
		http.Error(w, "Autentikasi diperlukan.", http.StatusUnauthorized)
		return
	}
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == h.allowedOrigin }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	h.add(claims.ID, conn)
	expires := time.AfterFunc(time.Until(expiresAt), func() { _ = conn.Close() })
	defer expires.Stop()
	defer func() { h.remove(claims.ID, conn); _ = conn.Close() }()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) CloseUser(userID string) {
	h.mu.RLock()
	connections := make([]*websocket.Conn, 0, len(h.connections[userID]))
	for conn := range h.connections[userID] {
		connections = append(connections, conn)
	}
	h.mu.RUnlock()
	for _, conn := range connections {
		_ = conn.Close()
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
	if notification.Leave != nil || notification.EntityType == "leave_request" || notification.Workspace != nil || notification.Type == "workspace_task" || notification.Type == model.NotificationWorkItemAssigned {
		return
	}
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

// Workspace events only invalidate authorized clients. No task or note content is sent.
func (h *Hub) SendWorkspace(userIDs []string, divisionID int, excludeUserID string) {
	h.SendEvent(userIDs, map[string]any{"event": "workspace.changed", "workspaceId": divisionID}, excludeUserID)
}

// Private invalidations contain no request content; the subsequent API read enforces current access.
func (h *Hub) SendEvent(userIDs []string, event any, excludeUserID string) {
	h.broadcastMu.Lock()
	defer h.broadcastMu.Unlock()
	payload, _ := json.Marshal(event)
	h.mu.RLock()
	targets := []*websocket.Conn{}
	for _, id := range userIDs {
		if id != excludeUserID {
			for c := range h.connections[id] {
				targets = append(targets, c)
			}
		}
	}
	h.mu.RUnlock()
	for _, c := range targets {
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, payload); err != nil {
			_ = c.Close()
		}
		_ = c.SetWriteDeadline(time.Time{})
	}
}
