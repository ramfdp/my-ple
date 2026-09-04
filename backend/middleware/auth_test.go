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

type authStore struct{ expires time.Time }

func (s authStore) FindSessionUser(context.Context, string) (model.User, time.Time, error) {
	if s.expires.IsZero() {
		return model.User{}, time.Time{}, errors.New("missing")
	}
	return model.User{ID: "user", IsActive: true}, s.expires, nil
}

func TestSessionTokensAreRandomAndOnlyTheirHashIsStable(t *testing.T) {
	a, ah, err := NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	b, bh, err := NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if a == b || ah == bh || ah != HashToken(a) || len(a) < 40 || len(ah) != 64 {
		t.Fatal("session token generation is not secure")
	}
}

func TestRequireAuthRejectsMissingAndExpiredSessions(t *testing.T) {
	for name, store := range map[string]authStore{"missing": {}, "expired": {expires: time.Now().Add(-time.Minute)}} {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
			if name == "expired" {
				r.AddCookie(&http.Cookie{Name: SessionCookie, Value: "expired"})
			}
			w := httptest.NewRecorder()
			RequireAuth(store, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("protected handler called") })).ServeHTTP(w, r)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d, want 401", w.Code)
			}
		})
	}
}
