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

type regularUserSession struct{}

func (regularUserSession) FindSessionUser(context.Context, string) (model.User, time.Time, error) {
	return model.User{ID: "user", RoleName: "Staff", IsActive: true}, time.Now().Add(time.Hour), nil
}

func TestRegularUserCanReachJobMutations(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/jobs", nil)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: "session"})
	res := httptest.NewRecorder()
	middleware.RequireAuth(regularUserSession{}, &JobController{}).ServeHTTP(res, req)
	if res.Code == http.StatusForbidden {
		t.Fatal("regular authenticated user was forbidden from job mutations")
	}
}
