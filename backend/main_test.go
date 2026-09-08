package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"my-ple-backend/controller"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

type testSessionStore struct{ user model.User }

func (s testSessionStore) FindSessionUser(context.Context, string) (model.User, time.Time, error) {
	if s.user.ID == "" {
		return model.User{}, time.Time{}, errors.New("invalid session")
	}
	return s.user, time.Now().Add(time.Minute), nil
}

func authenticatedRequest(method, path string, body io.Reader, role string) (*http.Request, testSessionStore) {
	r := httptest.NewRequest(method, path, body)
	r.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: "test-token"})
	return r, testSessionStore{user: model.User{ID: "fb98c49a-8ab7-474b-711d-7f3e2ae3a34f", RoleName: role, IsActive: true}}
}

func TestProfileUpdateAllowsEveryAuthenticatedRoleAndRejectsProtectedFields(t *testing.T) {
	for _, role := range []string{"Employee", "Admin", "Management", "HR", "Other"} {
		t.Run(role, func(t *testing.T) {
			request, store := authenticatedRequest(http.MethodPatch, "/profile", bytes.NewBufferString(`{"phoneNumber":"081234567890","roleId":1}`), role)
			response := httptest.NewRecorder()

			middleware.RequireAuth(store, protectedHandler(nil, controller.NewAuthController(nil, time.Hour, false))).ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("PATCH /profile with protected field status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
			}
		})
	}
}

func TestPDFRouteIsRegisteredBehindAuthentication(t *testing.T) {
	request, store := authenticatedRequest(http.MethodGet, "/reports.pdf", nil, "Employee")
	response := httptest.NewRecorder()

	middleware.RequireAuth(store, protectedHandler(nil, nil)).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("GET /reports.pdf status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func TestWeeklyHistoryRouteIsRegisteredBehindAuthentication(t *testing.T) {
	request, store := authenticatedRequest(http.MethodGet, "/reports/history?page=invalid", nil, "Employee")
	response := httptest.NewRecorder()
	middleware.RequireAuth(store, protectedHandler(nil, nil)).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("history route status=%d", response.Code)
	}
	response = httptest.NewRecorder()
	middleware.RequireAuth(testSessionStore{}, protectedHandler(nil, nil)).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatal("unauthenticated history accessible")
	}
}
