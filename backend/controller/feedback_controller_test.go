package controller

import (
	"context"
	"encoding/json"
	"errors"
	"my-ple-backend/email"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type feedbackSession struct{ address string }

func (s feedbackSession) FindSessionUser(context.Context, string) (model.User, time.Time, error) {
	return model.User{ID: "user", Name: "User", Email: s.address, IsActive: true}, time.Now().Add(time.Hour), nil
}
func TestFeedback(t *testing.T) {
	long, _ := json.Marshal(map[string]string{"message": strings.Repeat("a", 4001)})
	for _, tc := range []struct {
		name, body, address string
		login               bool
		failure             error
		status              int
		calls               int
	}{
		{"success", `{"message":"  Saran 👍\nBaris kedua  "}`, "user@gmail.com", true, nil, 204, 1},
		{"unauthenticated", `{"message":"test"}`, "user@gmail.com", false, nil, 401, 0},
		{"blank", `{"message":" \n "}`, "user@gmail.com", true, nil, 400, 0},
		{"long", string(long), "user@gmail.com", true, nil, 400, 0},
		{"spoof", `{"message":"test","email":"other@gmail.com"}`, "user@gmail.com", true, nil, 400, 0},
		{"recipient", `{"message":"test","to":"other@gmail.com"}`, "user@gmail.com", true, nil, 400, 0},
		{"invalid email", `{"message":"test"}`, "bad", true, nil, 400, 0},
		{"disabled", `{"message":"test"}`, "user@gmail.com", true, email.ErrFeedbackUnavailable, 503, 1},
		{"failure", `{"message":"test"}`, "user@gmail.com", true, errors.New("private SMTP detail"), 502, 1},
		{"uncertain", `{"message":"test"}`, "user@gmail.com", true, email.ErrUncertain, 502, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			handler := middleware.RequireAuth(feedbackSession{tc.address}, FeedbackHandler(func(_ context.Context, u model.Claims, message string) error {
				calls++
				if u.Email != tc.address || strings.TrimSpace(message) != message {
					t.Fatal("untrusted identity or untrimmed message")
				}
				return tc.failure
			}))
			req := httptest.NewRequest("POST", "/feedback", strings.NewReader(tc.body))
			if tc.login {
				req.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: "session"})
			}
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != tc.status || calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", res.Code, calls, res.Body.String())
			}
			if strings.Contains(res.Body.String(), "private SMTP detail") {
				t.Fatal("leaked SMTP detail")
			}
		})
	}
}
