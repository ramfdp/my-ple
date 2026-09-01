package controller

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

var (
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,50}$`)
	emailPattern    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

type AuthController struct {
	DB     *db.Database
	Secret string
}

func NewAuthController(database *db.Database, secret string) *AuthController {
	return &AuthController{
		DB:     database,
		Secret: secret,
	}
}

func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var input model.Credentials
	if !DecodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Username = strings.TrimSpace(input.Username)

	if len(input.Name) < 2 || len(input.Name) > 100 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Nama pengguna harus diisi (2-100 karakter)."})
		return
	}
	if !emailPattern.MatchString(input.Email) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Format email tidak valid."})
		return
	}
	if !usernamePattern.MatchString(input.Username) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Username tidak valid (3-50 karakter alfanumerik)."})
		return
	}
	if len(input.Password) < 8 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Password minimal 8 karakter."})
		return
	}

	var role model.Role
	var err error
	if input.RoleID > 0 {
		role, err = c.DB.FindRoleByID(r.Context(), input.RoleID)
	} else if strings.TrimSpace(input.RoleName) != "" {
		role, err = c.DB.FindRoleByName(r.Context(), strings.TrimSpace(input.RoleName))
	} else {
		role, err = c.DB.FindRoleByName(r.Context(), "PMO")
		if err != nil {
			roles, listErr := c.DB.ListRoles(r.Context())
			if listErr == nil && len(roles) > 0 {
				role = roles[0]
				err = nil
			}
		}
	}
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Role tidak valid atau tidak ditemukan."})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Terjadi kesalahan pada server."})
		return
	}
	u, err := c.DB.CreateUser(r.Context(), model.User{
		ID:               db.NewID(),
		Name:             input.Name,
		Email:            input.Email,
		RoleID:           role.ID,
		Username:         input.Username,
		PasswordHash:     string(hash),
		JobTitle:         input.JobTitle,
		Salary:           input.Salary,
		Residence:        input.Residence,
		PhoneNumber:      input.PhoneNumber,
		EmploymentStatus: input.EmploymentStatus,
		AvatarURL:        input.AvatarURL,
		IsActive:         true,
	})
	if db.UniqueViolation(err) {
		WriteJSON(w, http.StatusConflict, map[string]string{"error": "Username atau email sudah terdaftar."})
		return
	}
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Terjadi kesalahan pada server."})
		return
	}
	u.RoleName = role.Name
	WriteJSON(w, http.StatusCreated, map[string]any{"user": model.PublicUser(u)})
}

func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var input model.Credentials
	if !DecodeJSON(w, r, &input) {
		return
	}
	identifier := strings.TrimSpace(input.Username)
	if identifier == "" {
		identifier = strings.TrimSpace(input.Email)
	}
	u, err := c.DB.FindUser(r.Context(), identifier)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(input.Password)) != nil {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Username/email atau password salah."})
		return
	}
	profile := model.Claims{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Username:  u.Username,
		RoleID:    u.RoleID,
		RoleName:  u.RoleName,
		ExpiresAt: time.Now().Add(8 * time.Hour).UnixMilli(),
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"token": middleware.SignToken(profile, c.Secret),
		"user":  model.PublicUser(u),
	})
}

func (c *AuthController) Profile(w http.ResponseWriter, r *http.Request, userID string) {
	u, err := c.DB.FindUserByID(r.Context(), userID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Pengguna tidak ditemukan."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"user": model.PublicUser(u)})
}

func (c *AuthController) UpdateProfile(w http.ResponseWriter, r *http.Request, userID string) {
	var input model.UpdateProfileInput
	if !DecodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	if input.Email != "" && !emailPattern.MatchString(input.Email) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Format email tidak valid."})
		return
	}

	u, err := c.DB.UpdateUserProfile(r.Context(), userID, input)
	if db.UniqueViolation(err) {
		WriteJSON(w, http.StatusConflict, map[string]string{"error": "Email sudah digunakan oleh pengguna lain."})
		return
	}
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Gagal memperbarui profil pengguna."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"message": "Employee profile updated successfully.",
		"user":    model.PublicUser(u),
	})
}

func (c *AuthController) Roles(w http.ResponseWriter, r *http.Request) {
	roles, err := c.DB.ListRoles(r.Context())
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Gagal memuat data role."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"roles": roles})
}

func (c *AuthController) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := c.DB.ListUsers(r.Context())
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Gagal memuat pengguna."})
		return
	}
	result := make([]map[string]any, len(users))
	for i, user := range users {
		result[i] = model.PublicUser(user)
	}
	WriteJSON(w, http.StatusOK, map[string]any{"users": result})
}

func (c *AuthController) UpdateUser(w http.ResponseWriter, r *http.Request, id string) {
	var input model.UpdateUserInput
	if !DecodeJSON(w, r, &input) {
		return
	}
	input.Name, input.Email, input.Username = strings.TrimSpace(input.Name), strings.TrimSpace(strings.ToLower(input.Email)), strings.TrimSpace(input.Username)
	if len(input.Name) < 2 || !emailPattern.MatchString(input.Email) || !usernamePattern.MatchString(input.Username) || input.RoleID < 1 || input.Salary < 0 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Data pengguna tidak valid."})
		return
	}
	if _, err := c.DB.FindRoleByID(r.Context(), input.RoleID); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Role tidak valid."})
		return
	}
	passwordHash := ""
	if input.Password != "" {
		if len(input.Password) < 8 {
			WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Password minimal 8 karakter."})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Terjadi kesalahan pada server."})
			return
		}
		passwordHash = string(hash)
	}
	u, err := c.DB.UpdateUser(r.Context(), id, input, passwordHash)
	if db.UniqueViolation(err) {
		WriteJSON(w, http.StatusConflict, map[string]string{"error": "Username atau email sudah terdaftar."})
		return
	}
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Gagal memperbarui pengguna."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"user": model.PublicUser(u)})
}

func (c *AuthController) DeleteUser(w http.ResponseWriter, r *http.Request, id string) {
	if err := c.DB.DeleteUser(r.Context(), id); err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Pengguna tidak ditemukan."})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
