package model

import "time"

type Role struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
}

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	RoleID       int       `json:"roleId"`
	RoleName     string    `json:"role"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
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
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	RoleID   int    `json:"roleId,omitempty"`
	RoleName string `json:"role,omitempty"`
}

func PublicUser(u User) map[string]any {
	return map[string]any{
		"id":        u.ID,
		"name":      u.Name,
		"email":     u.Email,
		"username":  u.Username,
		"roleId":    u.RoleID,
		"role":      u.RoleName,
		"createdAt": u.CreatedAt,
		"updatedAt": u.UpdatedAt,
	}
}
