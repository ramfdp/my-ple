package main

import (
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type authController struct {
	db     *database
	secret string
}

func (c authController) login(w http.ResponseWriter, r *http.Request) {
	var input credentials
	if !decodeJSON(w, r, &input) {
		return
	}
	u, err := c.db.findUser(r.Context(), input.Username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(input.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Username atau password salah."})
		return
	}
	profile := claims{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, ExpiresAt: time.Now().Add(8 * time.Hour).UnixMilli()}
	writeJSON(w, http.StatusOK, map[string]any{"token": signToken(profile, c.secret), "user": publicUser(u)})
}
