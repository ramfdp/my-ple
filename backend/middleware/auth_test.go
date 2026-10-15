package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"my-ple-backend/model"
)

type sessionStub struct{}

func (sessionStub) FindSessionUser(context.Context, string) (model.User, time.Time, error) {
	return model.User{}, time.Time{}, errors.New("missing")
}

func TestHumanRoutesDoNotAcceptServiceBearer(t *testing.T) {
	h := RequireAuth(sessionStub{}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	r := httptest.NewRequest("GET", "/workspaces", nil)
	r.Header.Set("Authorization", "Bearer service-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("status=%d", w.Code)
	}
}
