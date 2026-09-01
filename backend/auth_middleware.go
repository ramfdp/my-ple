package main

import (
	"context"
	"net/http"
)

type authClaimsKey struct{}

func requireAuth(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current, ok := authenticated(r, secret)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Autentikasi diperlukan."})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authClaimsKey{}, current)))
	})
}
