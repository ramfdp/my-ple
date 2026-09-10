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

func NewWeeklyController(database *db.Database) *WeeklyController {
	return &WeeklyController{DB: database, Now: time.Now}
}

func (c *WeeklyController) currentPeriod(ctx context.Context, division string) (model.ReportingPeriod, error) {
	return c.DB.WorkingPeriod(ctx, division, c.Now())
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
	if input.Description == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Data laporan tidak valid."})
		return
	}
	report, err := c.DB.CreateReport(r.Context(), current.ID, input)
	if errors.Is(err, db.ErrInvalidReport) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Pekerjaan tidak aktif atau tidak sesuai divisi, role, dan kategori."})
		return
	}
	if errors.Is(err, db.ErrReportClosed) {
		WriteJSON(w, http.StatusConflict, map[string]string{"error": "Periode laporan ditutup. Input baru dibuka Selasa 00:00 WIB."})
		return
	}
	if err != nil {
		log.Printf("create report: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat disimpan."})
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"report": report})
}

func (c *WeeklyController) Update(w http.ResponseWriter, r *http.Request, id string) {
	_, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	var input model.WeeklyReportUpdateInput
	if !DecodeStrictJSON(w, r, &input) {
		return
	}
	if input.Version < 1 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Uraian laporan tidak valid."})
		return
	}
	// The repository validates the category against the report's stored primary division.
	report, err := c.DB.UpdateReport(r.Context(), id, input)
	if errors.Is(err, db.ErrReportConflict) {
		WriteJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "code": "report_conflict", "report": report})
		return
	}
	if errors.Is(err, db.ErrReportClosed) {
		WriteJSON(w, http.StatusConflict, map[string]string{"error": "Laporan periode yang sudah ditutup hanya dapat dibaca."})
		return
	}
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
	_, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	err := c.DB.DeleteReport(r.Context(), id)
	if errors.Is(err, db.ErrReportClosed) {
		WriteJSON(w, http.StatusConflict, map[string]string{"error": "Laporan periode yang sudah ditutup hanya dapat dibaca."})
		return
	}
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
	period, err := c.currentPeriod(r.Context(), division)
	if err != nil {
		log.Printf("load working report period: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat dimuat."})
		return
	}
	reports, err := c.DB.PeriodReports(r.Context(), division, period)
	if err != nil {
		log.Printf("list reports: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat dimuat."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"reports": reports, "period": period})
}
