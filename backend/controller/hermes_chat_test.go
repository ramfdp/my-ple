package controller

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHermesChatRequiresSession(t *testing.T) {
	c := &HermesChatController{}
	r := httptest.NewRequest("POST", "/hermes/chat", strings.NewReader(`{"question":"test"}`))
	w := httptest.NewRecorder()
	c.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("anonymous status=%d", w.Code)
	}
}
