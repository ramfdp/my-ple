package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

func TestPDFRouteIsRegisteredBehindAuthentication(t *testing.T) {
	const secret = "test-secret"
	request := httptest.NewRequest(http.MethodGet, "/reports.pdf?primaryDivision=invalid", nil)
	request.Header.Set("Authorization", "Bearer "+middleware.SignToken(model.Claims{ID: "user", ExpiresAt: time.Now().Add(time.Minute).UnixMilli()}, secret))
	response := httptest.NewRecorder()

	middleware.RequireAuth(secret, protectedHandler(nil, nil)).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("GET /reports.pdf status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}
