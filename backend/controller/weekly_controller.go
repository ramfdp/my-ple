package controller

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

type WeeklyController struct {
	DB  *db.Database
	Now func() time.Time
}

func validReportInput(input model.WeeklyReportInput) bool {
	return model.ValidWeeklyReport(input.PrimaryDivision, input.WorkDivision, input.Role, input.Work, input.Description)
}

func NewWeeklyController(database *db.Database) *WeeklyController {
	return &WeeklyController{DB: database, Now: time.Now}
}

func (c *WeeklyController) currentPeriod(ctx context.Context) (model.ReportingPeriod, error) {
	start, next := model.ReportingWeek(c.Now())
	if err := c.DB.CleanupExpiredReports(ctx, start); err != nil {
		return model.ReportingPeriod{}, err
	}
	return model.ReportingPeriod{Start: start, End: next.AddDate(0, 0, -1), Label: model.IndonesianWeekLabel(start, next)}, nil
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
	input.Role = strings.TrimSpace(input.Role)
	input.Work = strings.TrimSpace(input.Work)
	input.Role, input.WorkDivision = model.NormalizeWeeklyRoleCategory(input.Role, input.WorkDivision)
	if !validReportInput(input) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Data laporan tidak valid."})
		return
	}
	if _, err := c.currentPeriod(r.Context()); err != nil {
		log.Printf("cleanup reports: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat disimpan."})
		return
	}
	report, err := c.DB.CreateReport(r.Context(), current.ID, input)
	if err != nil {
		log.Printf("create report: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat disimpan."})
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"report": report})
}

func (c *WeeklyController) Update(w http.ResponseWriter, r *http.Request, id string) {
	current, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	if _, err := c.currentPeriod(r.Context()); err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat diperbarui."})
		return
	}
	var input model.WeeklyReportUpdateInput
	if !DecodeJSON(w, r, &input) {
		return
	}
	input.Description = strings.TrimSpace(input.Description)
	input.PrimaryDivision = strings.TrimSpace(input.PrimaryDivision)
	input.WorkDivision = strings.TrimSpace(input.WorkDivision)
	input.Role = strings.TrimSpace(input.Role)
	input.Work = strings.TrimSpace(input.Work)
	input.Role, input.WorkDivision = model.NormalizeWeeklyRoleCategory(input.Role, input.WorkDivision)
	if !model.ValidWeeklyReport(input.PrimaryDivision, input.WorkDivision, input.Role, input.Work, input.Description) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Uraian laporan tidak valid."})
		return
	}
	// The repository validates the category against the report's stored primary division.
	report, err := c.DB.UpdateReport(r.Context(), id, current.ID, input)
	if errors.Is(err, db.ErrInvalidReport) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Data laporan tidak valid."})
		return
	}
	if err == sql.ErrNoRows {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Laporan tidak ditemukan."})
		return
	}
	if err != nil {
		log.Printf("update report: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat diperbarui."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"report": report})
}

func (c *WeeklyController) Delete(w http.ResponseWriter, r *http.Request, id string) {
	current, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	if _, err := c.currentPeriod(r.Context()); err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat dihapus."})
		return
	}
	err := c.DB.DeleteReport(r.Context(), id, current.ID)
	if err == sql.ErrNoRows {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Laporan tidak ditemukan."})
		return
	}
	if err != nil {
		log.Printf("delete report: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat dihapus."})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *WeeklyController) List(w http.ResponseWriter, r *http.Request) {
	division := strings.TrimSpace(r.URL.Query().Get("primaryDivision"))
	if division == "" || len(division) > 100 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Divisi utama wajib diisi."})
		return
	}
	period, err := c.currentPeriod(r.Context())
	if err != nil {
		log.Printf("cleanup reports: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat dimuat."})
		return
	}
	reports, err := c.DB.ListReports(r.Context(), division, period.Start, period.End.AddDate(0, 0, 1))
	if err != nil {
		log.Printf("list reports: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat dimuat."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"reports": reports, "period": period})
}
