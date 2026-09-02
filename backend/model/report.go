package model

import "time"

var WeeklyWorkDivisions = map[string]map[string]bool{
	"Project Engineer":     {"Quality Assurance 1": true, "Quality Assurance 2": true, "Pengawasan": true, "Non-Pengawasan": true, "CIHPK": true},
	"AI Dev":               {"Pengembangan": true, "Pengujian Model": true, "Deployment": true, "Dokumentasi": true},
	"Keuangan":             {"Pembukuan": true, "Pembayaran": true, "Anggaran": true, "Pelaporan": true},
	"Business Development": {"Prospek": true, "Proposal": true, "Kemitraan": true, "Tindak Lanjut": true},
	"HR":                   {"Kehadiran": true, "Rekrutmen": true, "Administrasi": true, "Pengembangan SDM": true},
	"PMO":                  {"Milestone": true, "Rapat": true, "Risiko": true, "Pelaporan": true},
}

type WeeklyReportInput struct {
	PrimaryDivision string `json:"primaryDivision"`
	WorkDivision    string `json:"workDivision"`
	Description     string `json:"description"`
}

type WeeklyReport struct {
	ID              string    `json:"id"`
	UserID          string    `json:"userId"`
	PrimaryDivision string    `json:"primaryDivision"`
	WorkDivision    string    `json:"workDivision"`
	Number          int       `json:"number"`
	Description     string    `json:"description"`
	CompletedAt     time.Time `json:"completedAt"`
}

type WeeklyReportUpdateInput struct {
	PrimaryDivision string `json:"primaryDivision"`
	WorkDivision    string `json:"workDivision"`
	Description     string `json:"description"`
}
