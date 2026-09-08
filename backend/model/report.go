package model

import (
	"fmt"
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
	if role == "QA 1" || role == "QA 2" || role == "QA1" || role == "QA2" {
		role = "QA"
	}
	if category == "QA 1" {
		category = "QA1"
	} else if category == "QA 2" {
		category = "QA2"
	} else if category == "QA" {
		category = "QA1"
	}
	return role, category
}

func NormalizeWeeklyReport(report *WeeklyReport) {
	report.Role, report.WorkDivision = NormalizeWeeklyRoleCategory(report.Role, report.WorkDivision)
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
	JobID           *int   `json:"jobId"`
	PrimaryDivision string `json:"primaryDivision"`
	WorkDivision    string `json:"workDivision"`
	Role            string `json:"role"`
	Work            string `json:"work"`
	Description     string `json:"description"`
}
