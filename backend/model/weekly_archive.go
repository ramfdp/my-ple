package model

import "time"

type ArchiveItem struct {
	WeeklyReport
	EmployeeName   string `json:"employeeName"`
	EmployeeRoleID int    `json:"employeeRoleId"`
	EmployeeRole   string `json:"employeeRole"`
}

type WeeklyArchive struct {
	ID             string        `json:"id"`
	Division       string        `json:"division"`
	WeekStart      time.Time     `json:"weekStart"`
	WeekEnd        time.Time     `json:"weekEnd"` // period key boundary (Monday historically, Friday for current periods)
	Items          []ArchiveItem `json:"items,omitempty"`
	Status         string        `json:"status"`
	FinalizedAt    *time.Time    `json:"finalizedAt"`
	Filename       string        `json:"filename"`
	SHA256         string        `json:"-"`
	StorageKey     string        `json:"-"`
	PDFGeneratedAt *time.Time    `json:"pdfGeneratedAt"`
	ResetAt        *time.Time    `json:"resetAt"`
	CreatedAt      time.Time     `json:"createdAt"`
}

func (a WeeklyArchive) Period() ReportingPeriod {
	start, next := a.WeekStart, a.WeekEnd
	end, label := ReportingPeriodDates(start, next)
	return ReportingPeriod{Start: start, End: end, Label: label, ReadOnly: true}
}

func (a WeeklyArchive) Reports() []WeeklyReport {
	reports := make([]WeeklyReport, len(a.Items))
	for i, item := range a.Items {
		reports[i] = item.WeeklyReport
	}
	return reports
}
