package model

import "time"

type Role struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
}

// ponytail: single unified User model accommodating all profile fields
type User struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	RoleID           int       `json:"roleId"`
	RoleName         string    `json:"role"`
	Username         string    `json:"username"`
	PasswordHash     string    `json:"-"`
	JobTitle         string    `json:"jobTitle"`
	Salary           int64     `json:"salary"`
	Residence        string    `json:"residence"`
	PhoneNumber      string    `json:"phoneNumber"`
	EmploymentStatus string    `json:"employmentStatus"`
	AvatarURL        string    `json:"avatarUrl"`
	IsActive         bool      `json:"isActive"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type Claims struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	RoleID    int    `json:"roleId"`
	RoleName  string `json:"role"`
	ExpiresAt int64  `json:"expiresAt"`
}

type Credentials struct {
	Name             string `json:"name"`
	Email            string `json:"email"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	RoleID           int    `json:"roleId,omitempty"`
	RoleName         string `json:"role,omitempty"`
	JobTitle         string `json:"jobTitle,omitempty"`
	Salary           int64  `json:"salary,omitempty"`
	Residence        string `json:"residence,omitempty"`
	PhoneNumber      string `json:"phoneNumber,omitempty"`
	EmploymentStatus string `json:"employmentStatus,omitempty"`
	AvatarURL        string `json:"avatarUrl,omitempty"`
}

type UpdateProfileInput struct {
	Residence   string `json:"residence"`
	PhoneNumber string `json:"phoneNumber"`
}

type UpdateUserInput struct {
	Name             string `json:"name"`
	Email            string `json:"email"`
	Username         string `json:"username"`
	Password         string `json:"password,omitempty"`
	RoleID           int    `json:"roleId"`
	JobTitle         string `json:"jobTitle"`
	Salary           int64  `json:"salary"`
	Residence        string `json:"residence"`
	PhoneNumber      string `json:"phoneNumber"`
	EmploymentStatus string `json:"employmentStatus"`
	AvatarURL        string `json:"avatarUrl"`
	IsActive         bool   `json:"isActive"`
}

func PublicUser(u User) map[string]any {
	return map[string]any{
		"id":               u.ID,
		"name":             u.Name,
		"email":            u.Email,
		"username":         u.Username,
		"roleId":           u.RoleID,
		"role":             u.RoleName,
		"jobTitle":         u.JobTitle,
		"salary":           u.Salary,
		"residence":        u.Residence,
		"phoneNumber":      u.PhoneNumber,
		"employmentStatus": u.EmploymentStatus,
		"avatarUrl":        u.AvatarURL,
		"isActive":         u.IsActive,
		"createdAt":        u.CreatedAt,
		"updatedAt":        u.UpdatedAt,
	}
}
