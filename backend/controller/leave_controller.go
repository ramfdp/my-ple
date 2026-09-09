package controller

import (
	"database/sql"
	"errors"
	"log"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"my-ple-backend/notification"
	"net/http"
	"strconv"
	"strings"
)

type LeaveController struct {
	DB            *db.Database
	Notifications *notification.Service
}

func leaveError(w http.ResponseWriter, err error) {
	status, message := 500, "Gagal memproses pengajuan."
	switch {
	case errors.Is(err, db.ErrLeaveInvalid):
		status, message = 400, err.Error()
	case errors.Is(err, db.ErrLeaveForbidden):
		status, message = 403, err.Error()
	case errors.Is(err, db.ErrLeaveConflict):
		status, message = 409, err.Error()
	case errors.Is(err, sql.ErrNoRows):
		status, message = 404, "Pengajuan tidak ditemukan."
	}
	WriteJSON(w, status, map[string]string{"error": message})
}
func (c *LeaveController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, 401, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	path := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	review := len(path) > 1 && path[1] == "approvals"
	if review {
		path = append(path[:1], path[2:]...)
	}
	if len(path) == 1 && r.Method == "GET" {
		offset := 0
		if raw := r.URL.Query().Get("offset"); raw != "" {
			var err error
			offset, err = strconv.Atoi(raw)
			if err != nil {
				leaveError(w, db.ErrLeaveInvalid)
				return
			}
		}
		items, err := c.DB.LeaveRequests(r.Context(), user.ID, review, r.URL.Query().Get("status"), r.URL.Query().Get("type"), offset)
		if err != nil {
			leaveError(w, err)
			return
		}
		WriteJSON(w, 200, map[string]any{"requests": items})
		return
	}
	if len(path) == 2 && r.Method == "GET" {
		item, err := c.DB.LeaveRequest(r.Context(), user.ID, path[1], review)
		if err != nil {
			leaveError(w, err)
			return
		}
		WriteJSON(w, 200, map[string]any{"request": item})
		return
	}
	var item model.LeaveRequest
	var err error
	if len(path) == 1 && !review && r.Method == "POST" {
		var in model.LeaveInput
		if !DecodeStrictJSON(w, r, &in) {
			return
		}
		item, err = c.DB.CreateLeave(r.Context(), user.ID, in)
	} else if len(path) == 3 && review && path[2] == "decision" && r.Method == "POST" {
		var in model.LeaveDecision
		if !DecodeStrictJSON(w, r, &in) {
			return
		}
		item, err = c.DB.DecideLeave(r.Context(), user.ID, path[1], in)
	} else {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		leaveError(w, err)
		return
	}
	// Business data is already committed. Notification failure is retried by the existing worker.
	_, notifyErr := c.DB.NotifyLeave(r.Context(), item.ID)
	if notifyErr != nil {
		log.Printf("leave notification request=%s category=enqueue_failed", item.ID)
	} else if c.Notifications != nil && c.Notifications.Hub != nil {
		recipients, recipientErr := c.DB.LeaveRecipients(r.Context(), item.ID, item.Status)
		if recipientErr == nil {
			c.Notifications.Hub.SendEvent(recipients, map[string]string{"event": "leave.changed"}, "")
		}
	}
	WriteJSON(w, 200, map[string]any{"request": item})
}
