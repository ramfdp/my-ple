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

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,50}$`)

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
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if !usernamePattern.MatchString(input.Username) || len(input.DisplayName) < 2 || len(input.DisplayName) > 100 || len(input.Password) < 8 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Username, nama, atau password tidak valid."})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Terjadi kesalahan pada server."})
		return
	}
	u, err := c.DB.CreateUser(r.Context(), model.User{
		ID:           db.NewID(),
		Username:     input.Username,
		DisplayName:  input.DisplayName,
		PasswordHash: string(hash),
	})
	if db.UniqueViolation(err) {
		WriteJSON(w, http.StatusConflict, map[string]string{"error": "Username sudah digunakan."})
		return
	}
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Terjadi kesalahan pada server."})
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"user": model.PublicUser(u)})
}

func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var input model.Credentials
	if !DecodeJSON(w, r, &input) {
		return
	}
	u, err := c.DB.FindUser(r.Context(), input.Username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(input.Password)) != nil {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Username atau password salah."})
		return
	}
	profile := model.Claims{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		ExpiresAt:   time.Now().Add(8 * time.Hour).UnixMilli(),
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"token": middleware.SignToken(profile, c.Secret),
		"user":  model.PublicUser(u),
	})
}
