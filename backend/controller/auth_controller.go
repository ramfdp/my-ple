package controller

import (
	"database/sql"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

const maxAvatarSize = 1 << 20

var (
	usernamePattern  = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,50}$`)
	emailPattern     = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phonePattern     = regexp.MustCompile(`^(?:\+62|62|0)8[1-9][0-9]{7,10}$`)
	userIDPattern    = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	employmentStatus = map[string]bool{"Permanent Employee": true, "Contract Employee": true, "Intern": true}
)

func ownsAvatar(id, path string) bool {
	return strings.HasPrefix(path, "/uploads/"+id+".") || strings.HasPrefix(path, "/uploads/"+id+"-")
}

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
	if input.EmploymentStatus == "" {
		input.EmploymentStatus = "Permanent Employee"
	}
	if !employmentStatus[input.EmploymentStatus] {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Status kepegawaian tidak valid."})
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
	if !DecodeStrictJSON(w, r, &input) {
		return
	}
	input.Residence = strings.TrimSpace(input.Residence)
	input.PhoneNumber = strings.TrimSpace(input.PhoneNumber)
	cleanPhone := strings.NewReplacer(" ", "", "-", "").Replace(input.PhoneNumber)
	if len(input.Residence) > 255 || len(input.PhoneNumber) > 50 || !phonePattern.MatchString(cleanPhone) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Nomor telepon atau alamat tidak valid."})
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
	if input.AvatarURL != "" && !ownsAvatar(id, input.AvatarURL) {
		input.AvatarURL = ""
	}
	if len(input.Name) < 2 || !emailPattern.MatchString(input.Email) || !usernamePattern.MatchString(input.Username) || input.RoleID < 1 || input.Salary < 0 || !employmentStatus[input.EmploymentStatus] {
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

func (c *AuthController) UploadAvatar(w http.ResponseWriter, r *http.Request, id string) {
	if !userIDPattern.MatchString(id) {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Pengguna tidak ditemukan."})
		return
	}
	previous, err := c.DB.FindUserByID(r.Context(), id)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Pengguna tidak ditemukan."})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize+10240)
	if err := r.ParseMultipartForm(maxAvatarSize + 10240); err != nil {
		WriteJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "Ukuran foto maksimal 1 MB."})
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil || header.Size > maxAvatarSize {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Pilih foto berukuran maksimal 1 MB."})
		return
	}
	defer file.Close()
	_, format, err := image.Decode(io.LimitReader(file, maxAvatarSize+1))
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "File harus berupa gambar JPEG, PNG, atau GIF yang valid."})
		return
	}
	ext := map[string]string{"jpeg": "jpg", "png": "png", "gif": "gif"}[format]
	if ext == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "File harus berupa gambar JPEG, PNG, atau GIF yang valid."})
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Gagal memproses foto."})
		return
	}
	dir := os.Getenv("UPLOAD_DIR")
	if dir == "" {
		dir = "upload"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Gagal menyimpan foto."})
		return
	}
	name := fmt.Sprintf("%s-%s.%s", id, db.NewID(), ext)
	tmp, err := os.CreateTemp(dir, ".avatar-*")
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Gagal menyimpan foto."})
		return
	}
	_, copyErr := io.Copy(tmp, io.LimitReader(file, maxAvatarSize+1))
	closeErr := tmp.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(tmp.Name())
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Gagal menyimpan foto."})
		return
	}
	if err := os.Rename(tmp.Name(), filepath.Join(dir, name)); err != nil {
		os.Remove(tmp.Name())
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Gagal menyimpan foto."})
		return
	}
	u, err := c.DB.UpdateAvatar(r.Context(), id, "/uploads/"+name)
	if err != nil {
		os.Remove(filepath.Join(dir, name))
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Pengguna tidak ditemukan."})
		return
	}
	if ownsAvatar(id, previous.AvatarURL) && previous.AvatarURL != u.AvatarURL {
		_ = os.Remove(filepath.Join(dir, filepath.Base(previous.AvatarURL)))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"user": model.PublicUser(u)})
}

func (c *AuthController) DeleteUser(w http.ResponseWriter, r *http.Request, id string) {
	if err := c.DB.DeleteUser(r.Context(), id); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("delete user %s: %v", id, err)
			WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Gagal menghapus pengguna."})
			return
		}
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Pengguna tidak ditemukan."})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
