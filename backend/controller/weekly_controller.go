package controller

import (
	"net/http"
	"strings"

	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

type WeeklyController struct {
	DB *db.Database
}

func NewWeeklyController(database *db.Database) *WeeklyController {
	return &WeeklyController{DB: database}
}

func (c *WeeklyController) Create(w http.ResponseWriter, r *http.Request) {
	current, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}

	var input model.WeeklyReportInput
	if !DecodeJSON(w, r, &input) {
		return
	}
	input.PrimaryDivision = strings.TrimSpace(input.PrimaryDivision)
	input.WorkDivision = strings.TrimSpace(input.WorkDivision)
	input.Description = strings.TrimSpace(input.Description)
	if !model.WeeklyWorkDivisions[input.PrimaryDivision][input.WorkDivision] || input.Description == "" || len(input.Description) > 500 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Data laporan tidak valid."})
		return
	}
	report, err := c.DB.CreateReport(r.Context(), current.ID, input)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat disimpan."})
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"report": report})
}

func (c *WeeklyController) List(w http.ResponseWriter, r *http.Request) {
	division := strings.TrimSpace(r.URL.Query().Get("primaryDivision"))
	if division == "" || len(division) > 100 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Divisi utama wajib diisi."})
		return
	}
	reports, err := c.DB.ListReports(r.Context(), division)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat dimuat."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"reports": reports})
}
