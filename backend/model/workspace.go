package model

import (
	"encoding/json"
	"time"
)

const (
	WorkTodo         = "TODO"
	WorkInProgress   = "IN_PROGRESS"
	WorkNeedRevision = "NEED_REVISION"
	WorkDone         = "DONE"
	PriorityLow      = "LOW"
	PriorityMedium   = "MEDIUM"
	PriorityHigh     = "HIGH"
	PriorityUrgent   = "URGENT"
	PagePersonal     = "PERSONAL"
	PageDivision     = "DIVISION"
)

type Workspace struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	MemberCount     int    `json:"memberCount"`
	ActiveTaskCount int    `json:"activeTaskCount"`
	CanManage       bool   `json:"canManage"`
}
type WorkspaceMember struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl"`
}
type WorkItem struct {
	ID           string            `json:"id"`
	WorkspaceID  int               `json:"workspaceId"`
	JobID        *int              `json:"jobId"`
	JobName      string            `json:"jobName"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	RelatedURL   string            `json:"relatedUrl"`
	StartDate    string            `json:"startDate"`
	DueDate      string            `json:"dueDate"`
	Status       string            `json:"status"`
	Priority     string            `json:"priority"`
	CreatedBy    string            `json:"createdBy"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	Version      int               `json:"version"`
	Assignees    []WorkspaceMember `json:"assignees"`
	CommentCount int               `json:"commentCount"`
	CanEdit      bool              `json:"canEdit"`
	CanDelete    bool              `json:"canDelete"`
}
type WorkItemInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	RelatedURL  string   `json:"relatedUrl"`
	JobID       *int     `json:"jobId"`
	StartDate   string   `json:"startDate"`
	DueDate     string   `json:"dueDate"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	AssigneeIDs []string `json:"assigneeIds"`
	Version     int      `json:"version"`
}
type WorkComment struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}
type WorkActivity struct {
	ID        string          `json:"id"`
	TaskID    string          `json:"taskId"`
	ActorName string          `json:"actorName"`
	Action    string          `json:"action"`
	Field     string          `json:"field"`
	OldValue  json.RawMessage `json:"oldValue"`
	NewValue  json.RawMessage `json:"newValue"`
	CreatedAt time.Time       `json:"createdAt"`
}

// A bounded document: text, heading, list, checklist and links; never executable HTML.
type PageLine struct {
	Kind    string `json:"kind"`
	Text    string `json:"text"`
	Checked bool   `json:"checked"`
	URL     string `json:"url,omitempty"`
}
type WorkspacePage struct {
	ID          string     `json:"id"`
	WorkspaceID int        `json:"workspaceId"`
	ParentID    *string    `json:"parentId"`
	Title       string     `json:"title"`
	Content     []PageLine `json:"content"`
	Visibility  string     `json:"visibility"`
	CreatedBy   string     `json:"createdBy"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	Version     int        `json:"version"`
	CanEdit     bool       `json:"canEdit"`
}
type WorkspacePageInput struct {
	ParentID   *string    `json:"parentId"`
	Title      string     `json:"title"`
	Content    []PageLine `json:"content"`
	Visibility string     `json:"visibility"`
	Version    int        `json:"version"`
}
