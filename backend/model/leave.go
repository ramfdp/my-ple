package model

import "time"

type LeaveRequest struct {
	ID           string     `json:"id"`
	UserID       string     `json:"userId"`
	EmployeeName string     `json:"employeeName"`
	Division     string     `json:"division"`
	RequestType  string     `json:"requestType"`
	StartDate    string     `json:"startDate"`
	EndDate      string     `json:"endDate"`
	Reason       string     `json:"reason"`
	Notes        string     `json:"notes"`
	Status       string     `json:"status"`
	ReviewedBy   *string    `json:"reviewedBy"`
	ReviewerName string     `json:"reviewerName"`
	ReviewedAt   *time.Time `json:"reviewedAt"`
	ReviewerNote string     `json:"reviewerNote"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}
type LeaveInput struct {
	RequestID   string `json:"requestId"`
	RequestType string `json:"requestType"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Reason      string `json:"reason"`
	Notes       string `json:"notes"`
}
type LeaveDecision struct {
	Status       string `json:"status"`
	ReviewerNote string `json:"reviewerNote"`
}
type LeaveNotification struct {
	LeaveRequest
	Title   string `json:"title"`
	Message string `json:"message"`
	Path    string `json:"path"`
}
