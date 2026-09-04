package controller

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"my-ple-backend/db"
	"my-ple-backend/middleware"
)

type NotificationController struct{ DB *db.Database }

func (c *NotificationController) List(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.GetClaims(r.Context())
	items, err := c.DB.ListNotifications(r.Context(), current.ID)
	if err != nil {
		log.Printf("list notifications: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Notifikasi tidak dapat dimuat."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"notifications": items})
}

func (c *NotificationController) UnreadCount(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.GetClaims(r.Context())
	count, err := c.DB.UnreadNotificationCount(r.Context(), current.ID)
	if err != nil {
		log.Printf("count notifications: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Notifikasi tidak dapat dimuat."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (c *NotificationController) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.GetClaims(r.Context())
	if err := c.DB.MarkAllNotificationsRead(r.Context(), current.ID); err != nil {
		log.Printf("mark all notifications read: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Notifikasi tidak dapat diperbarui."})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *NotificationController) Update(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.GetClaims(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/notifications/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Endpoint tidak ditemukan."})
		return
	}
	var err error
	switch parts[1] {
	case "read":
		err = c.DB.MarkNotificationRead(r.Context(), parts[0], current.ID)
	case "dismiss":
		err = c.DB.DismissNotification(r.Context(), parts[0], current.ID)
	default:
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Endpoint tidak ditemukan."})
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Notifikasi tidak ditemukan."})
		return
	}
	if err != nil {
		log.Printf("update notification: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Notifikasi tidak dapat diperbarui."})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
