package model

import "time"

type OvertimeEntry struct {
	WorkDate        string `json:"workDate"`
	StartTime       string `json:"startTime"`
	EndTime         string `json:"endTime"`
	DayType         string `json:"dayType"`
	IsOverride      bool   `json:"isOverride"`
	OverrideReason  string `json:"overrideReason"`
	Project         string `json:"project"`
	Description     string `json:"description"`
	ActualMinutes   int    `json:"actualMinutes"`
	WeightedMinutes int    `json:"weightedMinutes"`
}

type OvertimeSheet struct {
	ID              string          `json:"id"`
	UserID          string          `json:"userId"`
	EmployeeName    string          `json:"employeeName"`
	EmployeeRole    string          `json:"employeeRole"`
	Period          string          `json:"period"`
	Status          string          `json:"status"`
	Version         int             `json:"version"`
	Entries         []OvertimeEntry `json:"entries"`
	ActualMinutes   int             `json:"actualMinutes"`
	WeightedMinutes int             `json:"weightedMinutes"`
	SubmittedAt     *time.Time      `json:"submittedAt"`
	ReviewedBy      *string         `json:"reviewedBy"`
	ReviewerName    string          `json:"reviewerName"`
	ReviewedAt      *time.Time      `json:"reviewedAt"`
	ReviewerNote    string          `json:"reviewerNote"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

type OvertimeEntryInput struct {
	WorkDate       string `json:"workDate"`
	StartTime      string `json:"startTime"`
	EndTime        string `json:"endTime"`
	DayType        string `json:"dayType"`
	OverrideReason string `json:"overrideReason"`
	Project        string `json:"project"`
	Description    string `json:"description"`
}

type OvertimeSaveInput struct {
	Version int                  `json:"version"`
	Entries []OvertimeEntryInput `json:"entries"`
}

type OvertimeDecision struct {
	Status       string `json:"status"`
	ReviewerNote string `json:"reviewerNote"`
}

type OvertimeRosterItem struct {
	SheetID      string     `json:"sheetId"`
	UserID       string     `json:"userId"`
	EmployeeName string     `json:"employeeName"`
	EmployeeRole string     `json:"employeeRole"`
	Status       string     `json:"status"`
	SubmittedAt  *time.Time `json:"submittedAt"`
	ReviewedAt   *time.Time `json:"reviewedAt"`
}

type OvertimeNotification struct {
	SheetID      string `json:"sheetId"`
	EmployeeName string `json:"employeeName"`
	Period       string `json:"period"`
	Status       string `json:"status"`
	Title        string `json:"title"`
	Message      string `json:"message"`
	Path         string `json:"path"`
}
