package model

import "time"

// AISnapshot contains only actionable work evidence for one granted workspace.
type AISnapshot struct {
	SchemaVersion      string    `json:"schemaVersion"`
	WorkspaceID        int       `json:"workspaceId"`
	SnapshotAt         time.Time `json:"snapshotAt"`
	Tasks              []AITask  `json:"tasks"`
	AllowedAssigneeIDs []string  `json:"allowedAssigneeIds"`
}

type AITask struct {
	ID            string   `json:"id"`
	WorkspaceID   int      `json:"workspaceId"`
	JobID         *int     `json:"jobId"`
	JobName       string   `json:"jobName"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	StartDate     string   `json:"startDate"`
	DueDate       string   `json:"dueDate"`
	Status        string   `json:"status"`
	Priority      string   `json:"priority"`
	AssigneeIDs   []string `json:"assigneeIds"`
	RevisionCount int      `json:"revisionCount"`
	RequiresQA    bool     `json:"requiresQa"`
	QAStage       string   `json:"qaStage"`
	Archived      bool     `json:"archived"`
	Version       int      `json:"version"`
}
