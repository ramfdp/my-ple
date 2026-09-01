package main

import (
	"context"
	"net/http"
	"strings"
	"time"
)

var weeklyWorkDivisions = map[string]map[string]bool{
	"Project Engineer":     {"Quality Assurance 1": true, "Quality Assurance 2": true, "Pengawasan": true, "Non-Pengawasan": true, "CIHPK": true},
	"AI Dev":               {"Pengembangan": true, "Pengujian Model": true, "Deployment": true, "Dokumentasi": true},
	"Keuangan":             {"Pembukuan": true, "Pembayaran": true, "Anggaran": true, "Pelaporan": true},
	"Business Development": {"Prospek": true, "Proposal": true, "Kemitraan": true, "Tindak Lanjut": true},
	"HR":                   {"Kehadiran": true, "Rekrutmen": true, "Administrasi": true, "Pengembangan SDM": true},
	"PMO":                  {"Milestone": true, "Rapat": true, "Risiko": true, "Pelaporan": true},
}

type weeklyController struct{ db *database }
type weeklyReportInput struct {
	PrimaryDivision string `json:"primaryDivision"`
	WorkDivision    string `json:"workDivision"`
	Description     string `json:"description"`
}
type weeklyReport struct {
	ID              string    `json:"id"`
	PrimaryDivision string    `json:"primaryDivision"`
	WorkDivision    string    `json:"workDivision"`
	Number          int       `json:"number"`
	Description     string    `json:"description"`
	CompletedAt     time.Time `json:"completedAt"`
}

func (c weeklyController) create(w http.ResponseWriter, r *http.Request, current claims) {
	var input weeklyReportInput
	if !decodeJSON(w, r, &input) {
		return
	}
	input.PrimaryDivision = strings.TrimSpace(input.PrimaryDivision)
	input.WorkDivision = strings.TrimSpace(input.WorkDivision)
	input.Description = strings.TrimSpace(input.Description)
	if !weeklyWorkDivisions[input.PrimaryDivision][input.WorkDivision] || input.Description == "" || len(input.Description) > 500 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Data laporan tidak valid."})
		return
	}
	report, err := c.createReport(r.Context(), current.ID, input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat disimpan."})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"report": report})
}

func (c weeklyController) list(w http.ResponseWriter, r *http.Request) {
	division := strings.TrimSpace(r.URL.Query().Get("primaryDivision"))
	if division == "" || len(division) > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Divisi utama wajib diisi."})
		return
	}
	reports, err := c.listReports(r.Context(), division)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat dimuat."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": reports})
}

func (c weeklyController) createReport(ctx context.Context, userID string, input weeklyReportInput) (weeklyReport, error) {
	var result weeklyReport
	err := c.db.conn.QueryRowContext(ctx, `
		WITH next_number AS (
			INSERT INTO report_sequences (primary_division, work_division, last_number)
			VALUES ($1, $2, 1)
			ON CONFLICT (primary_division, work_division)
			DO UPDATE SET last_number = report_sequences.last_number + 1
			RETURNING last_number
		)
		INSERT INTO reports (id, user_id, primary_division, work_division, number, description, completed_at)
		SELECT $3, $4, $1, $2, last_number, $5, NOW() FROM next_number
		RETURNING id, primary_division, work_division, number, description, completed_at`,
		input.PrimaryDivision, input.WorkDivision, newID(), userID, input.Description,
	).Scan(&result.ID, &result.PrimaryDivision, &result.WorkDivision, &result.Number, &result.Description, &result.CompletedAt)
	return result, err
}

func (c weeklyController) listReports(ctx context.Context, primaryDivision string) ([]weeklyReport, error) {
	rows, err := c.db.conn.QueryContext(ctx, `SELECT id, primary_division, work_division, number, description, completed_at FROM reports WHERE primary_division = $1 ORDER BY work_division, number`, primaryDivision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reports []weeklyReport
	for rows.Next() {
		var item weeklyReport
		if err := rows.Scan(&item.ID, &item.PrimaryDivision, &item.WorkDivision, &item.Number, &item.Description, &item.CompletedAt); err != nil {
			return nil, err
		}
		reports = append(reports, item)
	}
	return reports, rows.Err()
}
