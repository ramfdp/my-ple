package model

type User struct {
	ID           string
	Username     string
	DisplayName  string
	PasswordHash string
}

type Claims struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type Credentials struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

func PublicUser(u User) map[string]string {
	return map[string]string{
		"id":          u.ID,
		"username":    u.Username,
		"displayName": u.DisplayName,
	}
}
