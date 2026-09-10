package model

import "time"

const NotificationZoomBookingCreated = "zoom_booking_created"
const NotificationWorkItemAssigned = "work_item_assigned"

type NotificationActor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type NotificationBooking struct {
	Description string    `json:"description"`
	ID          string    `json:"id"`
	ZoomName    string    `json:"zoomName"`
	StartAt     time.Time `json:"startAt"`
	EndAt       time.Time `json:"endAt"`
}

type Notification struct {
	Leave      *LeaveNotification     `json:"leave,omitempty"`
	Workspace  *WorkspaceNotification `json:"workspace,omitempty"`
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Actor      NotificationActor      `json:"actor"`
	EntityType string                 `json:"entityType"`
	EntityID   string                 `json:"entityId"`
	Booking    *NotificationBooking   `json:"booking,omitempty"`
	CreatedAt  time.Time              `json:"createdAt"`
	ReadAt     *time.Time             `json:"readAt"`
}

type WorkspaceNotification struct {
	Title       string `json:"title"`
	WorkspaceID int    `json:"workspaceId"`
	Message     string `json:"message"`
	ProjectName string `json:"projectName,omitempty"`
	StartDate   string `json:"startDate,omitempty"`
	DueDate     string `json:"dueDate,omitempty"`
	Priority    string `json:"priority,omitempty"`
}

type WorkAssignmentEmail struct {
	WorkspaceNotification
	WorkspaceName  string `json:"workspaceName,omitempty"`
	TaskID         string `json:"taskId,omitempty"`
	AssignedByName string `json:"assignedByName"`
	Description    string `json:"description"`
}

type NotificationEvent struct {
	Event        string       `json:"event"`
	Notification Notification `json:"notification"`
}
