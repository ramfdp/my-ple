package controller

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"my-ple-backend/db"
	"my-ple-backend/model"
)

type aiSnapshotStore interface {
	AISnapshot(context.Context, int) (model.AISnapshot, error)
}

type AISnapshotHandler struct {
	store   aiSnapshotStore
	token   []byte
	grants  map[int]bool
	enabled bool
	ready   bool
}

// NewAISnapshotHandler loads a dedicated machine token once per backend process.
// Invalid enabled configuration remains unavailable until the process restarts.
func NewAISnapshotHandler(store aiSnapshotStore) *AISnapshotHandler {
	h := &AISnapshotHandler{store: store, enabled: os.Getenv("AI_SNAPSHOT_ENABLED") == "true"}
	if !h.enabled {
		return h
	}
	path := os.Getenv("AI_SERVICE_TOKEN_FILE")
	if path == "" {
		return h
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() > 128 || runtime.GOOS == "linux" && info.Mode().Perm()&0007 != 0 {
		return h
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return h
	}
	token, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(token) != 32 {
		return h
	}
	grants := map[int]bool{}
	for _, part := range strings.Split(os.Getenv("AI_ALLOWED_WORKSPACE_IDS"), ",") {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id <= 0 {
			return h
		}
		grants[id] = true
	}
	if len(grants) == 0 {
		return h
	}
	h.token, h.grants, h.ready = token, grants, true
	return h
}

func (h *AISnapshotHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	write := func(code int, message string) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		write(405, "Method not allowed")
		return
	}
	if !h.enabled {
		write(404, "Not found")
		return
	}
	if !h.ready || h.store == nil {
		write(503, "AI snapshot unavailable")
		return
	}
	const prefix = "/internal/ai/v1/workspaces/"
	path := strings.TrimPrefix(r.URL.Path, prefix)
	if !strings.HasSuffix(path, "/snapshot") {
		write(404, "Not found")
		return
	}
	idText := strings.TrimSuffix(path, "/snapshot")
	id, err := strconv.Atoi(idText)
	if err != nil || id <= 0 || strconv.Itoa(id) != idText {
		write(400, "Invalid workspace ID")
		return
	}
	value := r.Header.Get("Authorization")
	parts := strings.Split(value, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		write(401, "Unauthorized")
		return
	}
	provided, err := hex.DecodeString(parts[1])
	if err != nil || len(provided) != 32 || subtle.ConstantTimeCompare(provided, h.token) != 1 {
		write(401, "Unauthorized")
		return
	}
	if !h.grants[id] {
		write(403, "Workspace forbidden")
		return
	}
	s, err := h.store.AISnapshot(r.Context(), id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		write(404, "Workspace not found")
		return
	case errors.Is(err, db.ErrAISnapshotTooLarge):
		write(413, "Workspace exceeds AI snapshot limit")
		return
	case err != nil:
		write(503, "AI snapshot unavailable")
		return
	}
	if s.WorkspaceID != id || len(s.Tasks) > 200 {
		write(503, "AI snapshot unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(s)
}
