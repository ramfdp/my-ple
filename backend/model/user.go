package model

type Role struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type User struct {
	ID           string `json:"id"`
	RoleID       int    `json:"roleId"`
	RoleName     string `json:"roleName"`
	Username     string `json:"username"`
	DisplayName  string `json:"displayName"`
	PasswordHash string `json:"-"`
}

type Claims struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	RoleID      int    `json:"roleId"`
	RoleName    string `json:"roleName"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type Credentials struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
	RoleID      int    `json:"roleId,omitempty"`
	RoleName    string `json:"roleName,omitempty"`
}

func PublicUser(u User) map[string]any {
	return map[string]any{
		"id":          u.ID,
		"username":    u.Username,
		"displayName": u.DisplayName,
		"roleId":      u.RoleID,
		"role":        u.RoleName,
	}
}
