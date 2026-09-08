package model

type Job struct {
	ID            int    `json:"id"`
	DivisionID    int    `json:"divisionId"`
	RoleID        *int   `json:"roleId"`
	JobCategoryID *int   `json:"jobCategoryId"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	IsActive      bool   `json:"isActive"`
	SortOrder     int    `json:"sortOrder"`
	Division      string `json:"division"`
	Role          string `json:"role"`
	Category      string `json:"category"`
	Referenced    bool   `json:"referenced"`
}

type JobMaster struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	DivisionID *int   `json:"divisionId,omitempty"`
	RoleID     *int   `json:"roleId"`
}
