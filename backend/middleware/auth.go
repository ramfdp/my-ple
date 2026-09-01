package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"my-ple-backend/model"
)

type claimsKey struct{}

func RequireAuth(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current, ok := Authenticated(r, secret)
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
	val := ctx.Value(claimsKey{})
	if val == nil {
		return model.Claims{}, false
	}
	c, ok := val.(model.Claims)
	return c, ok
}

func SignToken(payload model.Claims, secret string) string {
	body, _ := json.Marshal(payload)
	encoded := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func Authenticated(r *http.Request, secret string) (model.Claims, bool) {
	parts := strings.Split(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), ".")
	if len(parts) != 2 {
		return model.Claims{}, false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(expected, actual) {
		return model.Claims{}, false
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	var result model.Claims
	if err != nil || json.Unmarshal(body, &result) != nil || result.ExpiresAt <= time.Now().UnixMilli() {
		return model.Claims{}, false
	}
	return result, true
}
