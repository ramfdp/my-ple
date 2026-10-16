package controller

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"my-ple-backend/db"
	"my-ple-backend/middleware"
)

var aiTaskID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func (h *AIImportHandler) serveAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/internal/ai/v1/analyses/ready" && r.Method == http.MethodGet {
		id, err := strconv.Atoi(r.URL.Query().Get("workspaceId"))
		if err != nil || !h.grants[id] {
			aiError(w, 403)
			return
		}
		batches, err := h.DB.ReadyAIAnalyses(r.Context(), id)
		if err != nil {
			aiError(w, 503)
			return
		}
		WriteJSON(w, 200, map[string]any{"batches": batches})
		return
	}
	if r.URL.Path != "/internal/ai/v1/analyses/import" || r.Method != http.MethodPost {
		aiError(w, 404)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256*1024)
	var b db.AIAnalysisBatch
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if dec.Decode(&b) != nil || dec.Decode(new(any)) != io.EOF || !validAnalysisBatch(b) {
		aiError(w, 400)
		return
	}
	if !h.grants[b.WorkspaceID] {
		aiError(w, 403)
		return
	}
	err := h.DB.ImportAIAnalysis(r.Context(), b)
	switch {
	case errors.Is(err, db.ErrAIPlanStale), errors.Is(err, db.ErrAIPlanConflict):
		aiError(w, 409)
	case errors.Is(err, db.ErrAIPlanInvalid):
		aiError(w, 400)
	case err != nil:
		aiError(w, 503)
	default:
		WriteJSON(w, 200, map[string]string{"id": b.ID, "status": "awaiting_review"})
	}
}

func validAnalysisBatch(b db.AIAnalysisBatch) bool {
	if b.WorkspaceID <= 0 || b.Stale || b.Finalized || b.PlanCreated || !aiValidDate(b.PlanDate) || b.GeneratedAt.IsZero() || !aiDigest.MatchString(b.ID) || !aiDigest.MatchString(b.SnapshotHash) || !aiDigest.MatchString(b.PromptHash) || !aiOneOf(b.ModelMode, "mock", "hermes") || !aiProfile.MatchString(b.ModelProfile) || b.Items == nil || len(b.Items) > 20 || b.ID != db.AnalysisBatchID(b) {
		return false
	}
	seen := map[string]bool{}
	for _, item := range b.Items {
		if seen[item.TaskID] || !aiTaskID.MatchString(item.TaskID) || item.Title != "" || item.RelatedURL != "" || item.ReviewerName != "" || item.ReviewedAt != nil || !aiDigest.MatchString(item.DocumentHash) || !aiOneOf(item.Decision, "PENDING", "UNAVAILABLE") || item.Version != 0 || len([]rune(item.Summary)) > 1000 || len(item.NextSteps) > 8 || item.Decision == "PENDING" && (strings.TrimSpace(item.Summary) == "" || len(item.NextSteps) == 0) {
			return false
		}
		for _, step := range item.NextSteps {
			if strings.TrimSpace(step) == "" || len([]rune(step)) > 250 {
				return false
			}
		}
		seen[item.TaskID] = true
	}
	return true
}

type AIAnalysesController struct{ DB *db.Database }

func (c *AIAnalysesController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		aiError(w, 401)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "ai-analyses" || parts[1] != "workspaces" {
		aiError(w, 404)
		return
	}
	workspace, err := strconv.Atoi(parts[2])
	if err != nil || workspace <= 0 || strconv.Itoa(workspace) != parts[2] {
		aiError(w, 400)
		return
	}
	if _, err = c.DB.WorkspaceAccess(r.Context(), claims.ID, workspace); err != nil {
		if errors.Is(err, db.ErrWorkspaceForbidden) {
			aiError(w, 403)
		} else if errors.Is(err, sql.ErrNoRows) {
			aiError(w, 404)
		} else {
			aiError(w, 503)
		}
		return
	}
	if len(parts) == 3 && r.Method == http.MethodGet {
		b, e := c.DB.AIAnalysis(r.Context(), workspace, "")
		if errors.Is(e, sql.ErrNoRows) {
			WriteJSON(w, 200, map[string]any{"batch": nil})
			return
		}
		if e != nil {
			aiError(w, 503)
			return
		}
		snap, e := c.DB.AISnapshot(r.Context(), workspace)
		if e != nil && !errors.Is(e, db.ErrAISnapshotTooLarge) {
			aiError(w, 503)
			return
		}
		if e == nil {
			var hash string
			hash, e = db.AISnapshotHash(snap)
			if e != nil {
				aiError(w, 503)
				return
			}
			b.Stale = hash != b.SnapshotHash
		} else {
			b.Stale = true
		}
		WriteJSON(w, 200, map[string]any{"batch": b})
		return
	}
	if len(parts) == 7 && parts[4] == "tasks" && parts[6] == "review" && r.Method == http.MethodPatch {
		if !aiDigest.MatchString(parts[3]) || !aiTaskID.MatchString(parts[5]) {
			aiError(w, 400)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var in struct {
			Decision string `json:"decision"`
			Version  *int   `json:"version"`
		}
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if dec.Decode(&in) != nil || dec.Decode(new(any)) != io.EOF || in.Version == nil || *in.Version < 0 || !aiOneOf(in.Decision, "APPROVED", "REJECTED") {
			aiError(w, 400)
			return
		}
		e := c.DB.ReviewAIAnalysis(r.Context(), workspace, parts[3], parts[5], claims.ID, in.Decision, *in.Version)
		switch {
		case errors.Is(e, db.ErrAIPlanConflict), errors.Is(e, db.ErrAIPlanStale):
			aiError(w, 409)
		case errors.Is(e, db.ErrWorkspaceForbidden):
			aiError(w, 403)
		case errors.Is(e, sql.ErrNoRows):
			aiError(w, 404)
		case e != nil:
			aiError(w, 503)
		default:
			WriteJSON(w, 200, map[string]string{"status": "reviewed"})
		}
		return
	}
	aiError(w, 404)
}
