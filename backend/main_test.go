package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"my-ple-backend/controller"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

func TestProfileUpdateRejectsProtectedFields(t *testing.T) {
	const secret = "test-secret"
	request := httptest.NewRequest(http.MethodPatch, "/profile", bytes.NewBufferString(`{"phoneNumber":"081234567890","roleId":1}`))
	request.Header.Set("Authorization", "Bearer "+middleware.SignToken(model.Claims{ID: "fb98c49a-8ab7-474b-711d-7f3e2ae3a34f", ExpiresAt: time.Now().Add(time.Minute).UnixMilli()}, secret))
	response := httptest.NewRecorder()

	middleware.RequireAuth(secret, protectedHandler(nil, controller.NewAuthController(nil, secret))).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("PATCH /profile with protected field status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

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
