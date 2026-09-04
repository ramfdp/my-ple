package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

type controllerSessionStore struct{ role string }

func (s controllerSessionStore) FindSessionUser(context.Context, string) (model.User, time.Time, error) {
	return model.User{ID: "user", RoleName: s.role, IsActive: true}, time.Now().Add(time.Hour), nil
}

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
	ctrl := NewZoomResourceController(nil)
	handler := middleware.RequireAuth(controllerSessionStore{role: "HR"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ctrl.Update(w, r, 1) }))
	request := httptest.NewRequest(http.MethodPut, "/zoom-resources/1", nil)
	request.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: "test"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}
