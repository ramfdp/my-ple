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

type OvertimeController struct {
	DB            *db.Database
	Notifications *notification.Service
}

func overtimeError(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "Gagal memproses data lembur."
	switch {
	case errors.Is(err, db.ErrOvertimeInvalid), errors.Is(err, model.ErrInvalidOvertime):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, db.ErrOvertimeForbidden):
		status, message = http.StatusForbidden, err.Error()
	case errors.Is(err, db.ErrOvertimeConflict):
		status, message = http.StatusConflict, err.Error()
	case errors.Is(err, db.ErrOvertimeWindow):
		status, message = http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, sql.ErrNoRows):
		status, message = http.StatusNotFound, "Data lembur tidak ditemukan."
	}
	WriteJSON(w, status, map[string]string{"error": message})
}

func (c *OvertimeController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	path := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	review := path[0] == "overtime-approvals"
	if review {
		c.review(w, r, claims.ID, path)
		return
	}
	if len(path) < 2 {
		http.NotFound(w, r)
		return
	}
	period := path[1]
	if len(path) == 2 && r.Method == http.MethodGet {
		item, err := c.DB.OvertimeSheet(r.Context(), claims.ID, period)
		if err != nil {
			overtimeError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, map[string]any{"sheet": item, "submissionOpen": model.OvertimeSubmissionOpen(c.DB.Now(), period) || item.Status == "REJECTED"})
		return
	}
	if len(path) == 2 && r.Method == http.MethodPut {
		var input model.OvertimeSaveInput
		if !DecodeStrictJSON(w, r, &input) {
			return
		}
		item, err := c.DB.SaveOvertimeSheet(r.Context(), claims.ID, period, input)
		if err != nil {
			overtimeError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, map[string]any{"sheet": item})
		return
	}
	if len(path) == 3 && path[2] == "submit" && r.Method == http.MethodPost {
		var input struct {
			Version int `json:"version"`
		}
		if !DecodeStrictJSON(w, r, &input) {
			return
		}
		item, err := c.DB.SubmitOvertimeSheet(r.Context(), claims.ID, period, input.Version)
		if err != nil {
			overtimeError(w, err)
			return
		}
		if _, notifyErr := c.DB.NotifyOvertime(r.Context(), item.ID); notifyErr != nil {
			log.Printf("overtime notification sheet=%s category=enqueue_failed", item.ID)
		}
		c.Notifications.Changed(r.Context(), "overtime.changed", 0, item.UserID, 0)
		WriteJSON(w, http.StatusOK, map[string]any{"sheet": item})
		return
	}
	http.NotFound(w, r)
}

func (c *OvertimeController) review(w http.ResponseWriter, r *http.Request, userID string, path []string) {
	if len(path) == 1 && r.Method == http.MethodGet {
		offset, err := strconv.Atoi(defaultString(r.URL.Query().Get("offset"), "0"))
		if err != nil {
			overtimeError(w, db.ErrOvertimeInvalid)
			return
		}
		items, err := c.DB.OvertimeRoster(r.Context(), userID, r.URL.Query().Get("period"), r.URL.Query().Get("status"), offset)
		if err != nil {
			overtimeError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, map[string]any{"employees": items})
		return
	}
	if len(path) == 2 && r.Method == http.MethodGet {
		item, err := c.DB.OvertimeReviewSheet(r.Context(), userID, path[1])
		if err != nil {
			overtimeError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, map[string]any{"sheet": item})
		return
	}
	if len(path) == 3 && path[2] == "decision" && r.Method == http.MethodPost {
		var input model.OvertimeDecision
		if !DecodeStrictJSON(w, r, &input) {
			return
		}
		item, err := c.DB.DecideOvertime(r.Context(), userID, path[1], input)
		if err != nil {
			overtimeError(w, err)
			return
		}
		if _, notifyErr := c.DB.NotifyOvertime(r.Context(), item.ID); notifyErr != nil {
			log.Printf("overtime notification sheet=%s category=enqueue_failed", item.ID)
		}
		c.Notifications.Changed(r.Context(), "overtime.changed", 0, item.UserID, 0)
		WriteJSON(w, http.StatusOK, map[string]any{"sheet": item})
		return
	}
	http.NotFound(w, r)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
