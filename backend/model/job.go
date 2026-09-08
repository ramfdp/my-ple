package model

import "encoding/json"

type JobPIC struct {
	UserID   string `json:"userId"`
	Name     string `json:"name"`
	IsActive bool   `json:"isActive"`
	Username string `json:"username,omitempty"`
}

type JobPICs []JobPIC

func (p *JobPICs) Scan(value any) error {
	if value == nil {
		*p = JobPICs{}
		return nil
	}
	return json.Unmarshal(value.([]byte), p)
}

type Job struct {
	PICs          JobPICs `json:"pics"`
	ID            int     `json:"id"`
	DivisionID    int     `json:"divisionId"`
	RoleID        *int    `json:"roleId"`
	JobCategoryID *int    `json:"jobCategoryId"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	IsActive      bool    `json:"isActive"`
	SortOrder     int     `json:"sortOrder"`
	Division      string  `json:"division"`
	Role          string  `json:"role"`
	Category      string  `json:"category"`
	Referenced    bool    `json:"referenced"`
}

type JobMaster struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	DivisionID *int   `json:"divisionId,omitempty"`
	RoleID     *int   `json:"roleId"`
}
