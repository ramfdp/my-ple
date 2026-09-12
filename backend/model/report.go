package model

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"
)

const ReportingTimezone = "Asia/Jakarta"

var indonesianMonths = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func ReportingWeek(now time.Time) (time.Time, time.Time) {
	location, _ := time.LoadLocation(ReportingTimezone)
	local := now.In(location)
	daysSinceMonday := (int(local.Weekday()) + 6) % 7
	start := time.Date(local.Year(), local.Month(), local.Day()-daysSinceMonday, 0, 0, 0, 0, location)
	return start, start.AddDate(0, 0, 7)
}

func IndonesianWeekLabel(start, next time.Time) string {
	end := next.AddDate(0, 0, -1)
	if start.Month() == end.Month() && start.Year() == end.Year() {
		return fmt.Sprintf("%d - %d %s %d", start.Day(), end.Day(), indonesianMonths[end.Month()-1], end.Year())
	}
	return fmt.Sprintf("%d %s - %d %s %d", start.Day(), indonesianMonths[start.Month()-1], end.Day(), indonesianMonths[end.Month()-1], end.Year())
}

func NormalizeWeeklyRoleCategory(role, category string) (string, string) {
	if role == "Project Engineer" {
		role = "PE"
	}
	if role == "Quality Assurance" || role == "QA 1" || role == "QA 2" || role == "QA1" || role == "QA2" {
		role = "QA"
	}
	if role == "PE" {
		if category == "MK/Supervisi" {
			category = "MK supervisi"
		}
		if category == "Non MK/Non Supervisi" {
			category = "Non-MK supervisi"
		}
	}
	if role == "QA" {
		if category == "QA" || category == "QA 1" {
			category = "QA1"
		}
		if category == "QA 2" {
			category = "QA2"
		}
	}
	return role, category
}

func CanonicalWeeklyRoleCategory(role, category string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "pe", "project engineer":
		role = "Project Engineer"
	case "qa", "qa1", "qa 1", "qa2", "qa 2", "quality assurance":
		role = "Quality Assurance"
	case "cihpk":
		role = "CIHPK"
	}
	switch role {
	case "Project Engineer":
		switch strings.ToLower(strings.TrimSpace(category)) {
		case "mk", "supervisi", "mk supervisi", "mk/supervisi":
			category = "MK/Supervisi"
		case "non mk", "non-mk supervisi", "non supervisi", "non-supervisory", "non mk/non supervisi":
			category = "Non MK/Non Supervisi"
		}
	case "Quality Assurance":
		switch strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(category)), " ", "") {
		case "QA", "QA1":
			category = "QA 1"
		case "QA2":
			category = "QA 2"
		}
	case "CIHPK":
		category = "MK/Supervisi"
	}
	return role, category
}

func NormalizeWeeklyReport(report *WeeklyReport) {
	if strings.TrimSpace(report.Role) == "" {
		report.Role = report.PrimaryDivision
	}
	if strings.TrimSpace(report.WorkDivision) == "" {
		report.WorkDivision = "Tanpa Kategori"
	}
	report.Role, report.WorkDivision = CanonicalWeeklyRoleCategory(report.Role, report.WorkDivision)
	if report.Role == "CIHPK" {
		switch report.Work {
		case "Jember Station Area Redevelopment", "Penataan Kawasan Stasiun Jember":
			report.Work = "Supervisi Penataan Kawasan Stasiun Jember (Tanggal SPMK: 30 Desember 2025)"
		case "JAKG Supervising Consultant (SPMK Date: March 12, 2026)", "Pengawasan Renovasi Gudang Besar JAKG":
			report.Work = "Konsultan Pengawas JAKG (Tanggal SPMK: 12 Maret 2026)"
		case "Supervising Consultant for the Manggarai Balai Yasa Hydrant Installation (SPMK Date: June 8, 2026)", "Pengawasan Instalasi Hidran BY Manggarai":
			report.Work = "Konsultan Pengawas Instalasi Hydrant Balai Yasa Manggarai (Tanggal SPMK: 08 Juni 2026)"
		}
	}
}

type WeeklyReportInput struct {
	JobID           *int   `json:"jobId"`
	PrimaryDivision string `json:"primaryDivision"`
	WorkDivision    string `json:"workDivision"`
	Role            string `json:"role"`
	Work            string `json:"work"`
	Description     string `json:"description"`
}
type WeeklyReport struct {
	Version         int       `json:"version"`
	DisplayNumber   int       `json:"displayNumber"`
	PICs            JobPICs   `json:"pics,omitempty"`
	JobID           *int      `json:"jobId,omitempty"`
	ID              string    `json:"id"`
	UserID          string    `json:"userId"`
	PrimaryDivision string    `json:"primaryDivision"`
	WorkDivision    string    `json:"workDivision"`
	Role            string    `json:"role"`
	Work            string    `json:"work"`
	Number          int       `json:"number"`
	Description     string    `json:"description"`
	CompletedAt     time.Time `json:"completedAt"`
}
type ReportingPeriod struct {
	Status   string    `json:"status"`
	ReadOnly bool      `json:"readOnly"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Label    string    `json:"label"`
}
type WeeklyReportUpdateInput struct {
	Version         int     `json:"version"`
	JobID           *int    `json:"jobId"`
	PrimaryDivision *string `json:"primaryDivision"`
	WorkDivision    *string `json:"workDivision"`
	Role            *string `json:"role"`
	Work            *string `json:"work"`
	Description     *string `json:"description"`
}
