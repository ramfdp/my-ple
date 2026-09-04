package middleware

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"my-ple-backend/model"
)

type claimsKey struct{}

const SessionCookie = "my_ple_session"

type SessionStore interface {
	FindSessionUser(context.Context, string) (model.User, time.Time, error)
}

func RequireAuth(store SessionStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current, _, ok := Authenticated(r, store)
		if !ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Autentikasi diperlukan."})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, current)))
	})
}

func GetClaims(ctx context.Context) (model.Claims, bool) {
	c, ok := ctx.Value(claimsKey{}).(model.Claims)
	return c, ok
}

func Authenticated(r *http.Request, store SessionStore) (model.Claims, time.Time, bool) {
	cookie, err := r.Cookie(SessionCookie)
	if err != nil || cookie.Value == "" || store == nil {
		return model.Claims{}, time.Time{}, false
	}
	u, expires, err := store.FindSessionUser(r.Context(), HashToken(cookie.Value))
	if err != nil || !u.IsActive || !expires.After(time.Now()) {
		return model.Claims{}, time.Time{}, false
	}
	return model.Claims{ID: u.ID, Name: u.Name, Email: u.Email, Username: u.Username, RoleID: u.RoleID, RoleName: u.RoleName, ExpiresAt: expires.UnixMilli()}, expires, true
}

func NewSessionToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token), nil
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
