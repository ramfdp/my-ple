package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

func TestZoomResourceEditRoles(t *testing.T) {
	for _, role := range []string{"Admin", "Management"} {
		if !canEditZoomResources(role) {
			t.Fatalf("%s must be allowed", role)
		}
	}
	for _, role := range []string{"HR", "Project Engineer", "Finance", "IT"} {
		if canEditZoomResources(role) {
			t.Fatalf("%s must be view-only", role)
		}
	}
}

func TestZoomResourceUpdateRejectsHRRequest(t *testing.T) {
	secret := "test-secret"
	ctrl := NewZoomResourceController(nil)
	handler := middleware.RequireAuth(secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ctrl.Update(w, r, 1) }))
	token := middleware.SignToken(model.Claims{ID: "user", RoleName: "HR", ExpiresAt: time.Now().Add(time.Hour).UnixMilli()}, secret)
	request := httptest.NewRequest(http.MethodPut, "/zoom-resources/1", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}
