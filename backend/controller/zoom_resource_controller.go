package controller

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

type ZoomResourceController struct{ DB *db.Database }

func NewZoomResourceController(database *db.Database) *ZoomResourceController {
	return &ZoomResourceController{DB: database}
}

func canEditZoomResources(role string) bool {
	return strings.EqualFold(role, "Admin") || strings.EqualFold(role, "Management")
}

func (c *ZoomResourceController) List(w http.ResponseWriter, r *http.Request) {
	resources, err := c.DB.ListZoomResources(r.Context())
	if err != nil {
		log.Printf("list zoom resources: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Informasi Zoom tidak dapat dimuat."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"resources": resources})
}

func (c *ZoomResourceController) Update(w http.ResponseWriter, r *http.Request, id int) {
	current, _ := middleware.GetClaims(r.Context())
	if !canEditZoomResources(current.RoleName) {
		WriteJSON(w, http.StatusForbidden, map[string]string{"error": "Anda tidak memiliki izin untuk mengubah informasi Zoom."})
		return
	}
	var input model.ZoomResourceUpdateInput
	if !DecodeJSON(w, r, &input) {
		return
	}
	input.Description = strings.TrimSpace(input.Description)
	input.MeetingID = strings.TrimSpace(input.MeetingID)
	input.HostKey = strings.TrimSpace(input.HostKey)
	input.Passcode = strings.TrimSpace(input.Passcode)
	input.MeetingURL = strings.TrimSpace(input.MeetingURL)
	if !model.ValidZoomResourceUpdate(input) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Informasi Zoom tidak valid."})
		return
	}
	resource, err := c.DB.UpdateZoomResource(r.Context(), id, input)
	if errors.Is(err, sql.ErrNoRows) {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Resource Zoom tidak ditemukan."})
		return
	}
	if err != nil {
		log.Printf("update zoom resource: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Informasi Zoom tidak dapat diperbarui."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"resource": resource})
}
