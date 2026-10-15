package controller

import (
	"context"
	"database/sql"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"my-ple-backend/db"
	"my-ple-backend/model"
)

type aiStore struct {
	result    model.AISnapshot
	err       error
	calls     int
	workspace int
}

func (s *aiStore) AISnapshot(_ context.Context, id int) (model.AISnapshot, error) {
	s.calls++
	s.workspace = id
	return s.result, s.err
}

func TestAISnapshotAuthenticationAndProjection(t *testing.T) {
	secret := strings.Repeat("a", 64)
	file := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(file, []byte(secret+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AI_SNAPSHOT_ENABLED", "true")
	t.Setenv("AI_SERVICE_TOKEN_FILE", file)
	t.Setenv("AI_ALLOWED_WORKSPACE_IDS", "1")
	store := &aiStore{result: model.AISnapshot{SchemaVersion: "1.0", WorkspaceID: 1, SnapshotAt: time.Now().UTC(), Tasks: []model.AITask{{ID: "550e8400-e29b-41d4-a716-446655440001", WorkspaceID: 1, Title: "Kerja", Status: "TODO", Priority: "HIGH", AssigneeIDs: []string{}, Version: 1}}, AllowedAssigneeIDs: []string{}}}
	h := NewAISnapshotHandler(store)
	path := "/internal/ai/v1/workspaces/1/snapshot"
	for _, tc := range []struct {
		name, method, path, auth, cookie string
		status                           int
	}{
		{"missing", "GET", path, "", "", 401}, {"malformed", "GET", path, "Bearer short", "", 401}, {"wrong", "GET", path, "Bearer " + strings.Repeat("b", 64), "", 401},
		{"cookie", "GET", path, "", "my_ple_session=human", 401}, {"method", "POST", path, "Bearer " + secret, "", 405},
		{"invalid", "GET", "/internal/ai/v1/workspaces/1%20OR%201=1/snapshot", "Bearer " + secret, "", 400},
		{"forbidden", "GET", "/internal/ai/v1/workspaces/2/snapshot", "Bearer " + secret, "", 403},
		{"valid", "GET", path, "Bearer " + secret, "", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.auth != "" {
				r.Header.Set("Authorization", tc.auth)
			}
			if tc.cookie != "" {
				r.Header.Set("Cookie", tc.cookie)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if tc.name == "valid" {
				if store.calls != 1 || store.workspace != 1 || strings.Contains(w.Body.String(), "email") || strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "avatar") {
					t.Fatal("bad projection or grant")
				}
			}
		})
	}
	secondReplica := NewAISnapshotHandler(store)
	replicaRequest := httptest.NewRequest("GET", path, nil)
	replicaRequest.Header.Set("Authorization", "Bearer "+secret)
	replicaResponse := httptest.NewRecorder()
	secondReplica.ServeHTTP(replicaResponse, replicaRequest)
	if replicaResponse.Code != 200 || store.calls != 2 {
		t.Fatal("replica configuration differs")
	}
	store.err = db.ErrAISnapshotTooLarge
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", path, nil)
	r.Header.Set("Authorization", "Bearer "+secret)
	h.ServeHTTP(w, r)
	if w.Code != 413 {
		t.Fatalf("oversize=%d", w.Code)
	}
	store.err = sql.ErrNoRows
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatalf("missing=%d", w.Code)
	}
}

func TestAISnapshotDisabledAndMisconfigured(t *testing.T) {
	path := "/internal/ai/v1/workspaces/1/snapshot"
	store := &aiStore{}
	for _, tc := range []struct {
		name, enabled, file, grants string
		status                      int
	}{
		{"disabled", "false", "", "", 404}, {"missing-token", "true", "", "1", 503}, {"missing-grants", "true", "missing", "", 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AI_SNAPSHOT_ENABLED", tc.enabled)
			t.Setenv("AI_SERVICE_TOKEN_FILE", tc.file)
			t.Setenv("AI_ALLOWED_WORKSPACE_IDS", tc.grants)
			w := httptest.NewRecorder()
			NewAISnapshotHandler(store).ServeHTTP(w, httptest.NewRequest("GET", path, nil))
			if w.Code != tc.status {
				t.Fatalf("status=%d", w.Code)
			}
		})
	}
	if store.calls != 0 {
		t.Fatal(errors.New("store accessed without credentials"))
	}
}
