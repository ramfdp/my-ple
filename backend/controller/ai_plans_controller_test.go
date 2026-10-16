package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

type aiSessionStub struct{ role string }

func (s aiSessionStub) FindSessionUser(context.Context, string) (model.User, time.Time, error) {
	return model.User{ID: "550e8400-e29b-41d4-a716-446655440111", Name: "Fixture", RoleName: s.role, IsActive: true}, time.Now().Add(time.Hour), nil
}
func TestAIPlanHumanSessionRequired(t *testing.T) {
	route := middleware.RequireAuth(aiSessionStub{role: "Project Engineer"}, &AIPlansController{})
	w := httptest.NewRecorder()
	route.ServeHTTP(w, httptest.NewRequest("GET", "/ai-plans/workspaces/1", nil))
	if w.Code != 401 {
		t.Fatalf("missing session: %d", w.Code)
	}
	req := httptest.NewRequest("GET", "/ai-plans/workspaces/1", nil)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: "synthetic-session"})
	w = httptest.NewRecorder()
	route.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("ordinary employee: %d", w.Code)
	}
}

func validTestDraft() db.AIDraft {
	d := db.AIDraft{SchemaVersion: "1.0", WorkspaceID: 1, PlanDate: "2026-10-16", Status: "PENDING_REVIEW", EvidenceStatus: "INSUFFICIENT", GeneratedAt: time.Now().UTC(), InputSnapshotHash: strings.Repeat("a", 64), PromptVersion: "daily-v3", PromptHash: strings.Repeat("b", 64), ModelMode: "mock", ModelProfile: "mock-v1", Selection: &db.AISelection{}, Recommendations: []db.AIRecommendation{}}
	id := sha256.Sum256([]byte("1|2026-10-16|" + d.InputSnapshotHash + "|daily-v3|" + d.PromptHash + "|mock|mock-v1|1.0"))
	d.ID = hex.EncodeToString(id[:])
	return d
}
func TestAIImportValidationAndAuthentication(t *testing.T) {
	d := validTestDraft()
	if !validateAIDraft(d) {
		t.Fatal("valid draft rejected")
	}
	d.Selection.Actionable = 21
	if validateAIDraft(d) {
		t.Fatal("false selection accepted")
	}
	d = validTestDraft()
	d.PlanDate = "2026-02-30"
	if validateAIDraft(d) {
		t.Fatal("invalid date accepted")
	}
	t.Setenv("AI_IMPORT_ENABLED", "false")
	h := NewAIImportHandler(nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/internal/ai/v1/drafts/import", nil))
	if w.Code != 404 {
		t.Fatalf("disabled: %d", w.Code)
	}
	t.Setenv("AI_IMPORT_ENABLED", "true")
	t.Setenv("AI_IMPORT_TOKEN_FILE", "")
	h = NewAIImportHandler(nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/internal/ai/v1/drafts/import", nil))
	if w.Code != 503 {
		t.Fatalf("missing token: %d", w.Code)
	}
	file := t.TempDir() + "/token"
	if err := os.WriteFile(file, []byte(strings.Repeat("c", 64)), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AI_IMPORT_TOKEN_FILE", file)
	t.Setenv("AI_IMPORT_ALLOWED_WORKSPACE_IDS", "1")
	h = NewAIImportHandler(nil)
	h.DB = &db.Database{}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/internal/ai/v1/drafts/import", nil))
	if w.Code != 401 {
		t.Fatalf("missing bearer: %d", w.Code)
	}
	w = httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/internal/ai/v1/drafts/import", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+strings.Repeat("d", 64))
	h.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("wrong bearer: %d", w.Code)
	}
	d = validTestDraft()
	d.WorkspaceID = 2
	body, _ := json.Marshal(d)
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/internal/ai/v1/drafts/import", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+strings.Repeat("c", 64))
	h.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("malformed identity: %d", w.Code)
	}
	d = validTestDraft()
	d.WorkspaceID = 2
	id := sha256.Sum256([]byte("2|2026-10-16|" + d.InputSnapshotHash + "|daily-v3|" + d.PromptHash + "|mock|mock-v1|1.0"))
	d.ID = hex.EncodeToString(id[:])
	body, _ = json.Marshal(d)
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/internal/ai/v1/drafts/import", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+strings.Repeat("c", 64))
	h.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("workspace token grant: %d", w.Code)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/internal/ai/v1/drafts/import", strings.NewReader(`{"schemaVersion":"1.0","role":"Admin"}`))
	req.Header.Set("Authorization", "Bearer "+strings.Repeat("c", 64))
	h.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("unknown role field: %d", w.Code)
	}
}
func TestPublicGatewayBlocksAIImport(t *testing.T) {
	for _, name := range []string{"../../nginx.conf", "../../nginx.testing.conf"} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		s := string(raw)
		for _, prefix := range []string{"location ^~ /internal/ai/ { return 404; }", "location ^~ /api/internal/ai/ { return 404; }"} {
			if !strings.Contains(s, prefix) {
				t.Fatalf("%s does not block %s", name, prefix)
			}
		}
	}
	_ = http.MethodPost
}
