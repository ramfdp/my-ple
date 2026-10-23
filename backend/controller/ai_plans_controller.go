package controller

import (
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"my-ple-backend/db"
	"my-ple-backend/middleware"
)

var aiDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var aiProfile = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]{0,79}$`)
var aiDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func aiValidDate(s string) bool {
	if !aiDate.MatchString(s) {
		return false
	}
	t, e := time.Parse("2006-01-02", s)
	return e == nil && t.Format("2006-01-02") == s
}
func aiOneOf(s string, choices ...string) bool {
	for _, c := range choices {
		if s == c {
			return true
		}
	}
	return false
}
func aiError(w http.ResponseWriter, status int) {
	messages := map[int]string{400: "Data rencana tidak valid.", 401: "Autentikasi diperlukan.", 403: "Akses ditolak.", 404: "Data tidak ditemukan.", 405: "Metode tidak diizinkan.", 409: "Data sudah berubah. Muat ulang.", 413: "Data terlalu besar.", 503: "Layanan AI tidak tersedia."}
	WriteJSON(w, status, map[string]string{"error": messages[status]})
}

func validateAIDraft(d db.AIDraft) bool {
	if d.SchemaVersion != "1.0" || !aiOneOf(d.PromptVersion, "daily-v3", "daily-v4") || d.WorkspaceID <= 0 || !aiValidDate(d.PlanDate) || d.Status != "PENDING_REVIEW" || !aiOneOf(d.EvidenceStatus, "SUFFICIENT", "INSUFFICIENT") || d.GeneratedAt.IsZero() || !aiDigest.MatchString(d.ID) || !aiDigest.MatchString(d.InputSnapshotHash) || !aiDigest.MatchString(d.PromptHash) || !aiOneOf(d.ModelMode, "mock", "hermes") || !aiProfile.MatchString(d.ModelProfile) || d.Selection == nil || d.Recommendations == nil || len(d.Recommendations) > 20 {
		return false
	}
	if d.Selection.Actionable < 0 || d.Selection.Selected != min(d.Selection.Actionable, 20) || d.Selection.Deferred != d.Selection.Actionable-d.Selection.Selected || len(d.Recommendations) > d.Selection.Selected {
		return false
	}
	if (len(d.Recommendations) == 0) != (d.EvidenceStatus == "INSUFFICIENT") {
		return false
	}
	identity := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s|1.0", d.WorkspaceID, d.PlanDate, d.InputSnapshotHash, d.PromptVersion, d.PromptHash, d.ModelMode, d.ModelProfile)
	if d.PromptVersion == "daily-v4" {
		if !aiDigest.MatchString(d.AnalysisBatchID) {
			return false
		}
		identity = fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s|%s|1.0", d.WorkspaceID, d.PlanDate, d.InputSnapshotHash, d.PromptVersion, d.PromptHash, d.ModelMode, d.ModelProfile, d.AnalysisBatchID)
	} else if d.AnalysisBatchID != "" {
		return false
	}
	id := sha256.Sum256([]byte(identity))
	if d.ID != hex.EncodeToString(id[:]) {
		return false
	}
	seen := map[string]bool{}
	for _, r := range d.Recommendations {
		if d.PromptVersion == "daily-v4" {
			if len(r.NextSteps) == 0 || len(r.NextSteps) > 8 {
				return false
			}
			for _, step := range r.NextSteps {
				if strings.TrimSpace(step) == "" || len([]rune(step)) > 250 {
					return false
				}
			}
		} else if len(r.NextSteps) > 0 {
			return false
		}
		if !aiDigest.MatchString(r.ID) || seen[r.ID] || strings.TrimSpace(r.Title) == "" || len([]rune(r.Title)) > 160 || strings.TrimSpace(r.Description) == "" || len([]rune(r.Description)) > 1000 || strings.TrimSpace(r.Reason) == "" || len([]rune(r.Reason)) > 500 || !aiOneOf(r.Priority, "LOW", "MEDIUM", "HIGH", "URGENT") || !aiOneOf(r.RecommendationType, "CONTINUE_EXISTING", "REVIEW_EXISTING", "ADDRESS_OVERDUE", "QA_OR_REVISION") || !aiValidDate(r.SuggestedDueDate) || len(r.SourceTaskIDs) != 1 || len(r.AssigneeIDs) > 10 {
			return false
		}
		rid := sha256.Sum256([]byte(d.ID + "|" + r.SourceTaskIDs[0] + "|" + r.RecommendationType))
		if r.ID != hex.EncodeToString(rid[:]) {
			return false
		}
		seen[r.ID] = true
	}
	return true
}

type AIImportHandler struct {
	DB             *db.Database
	enabled, ready bool
	token          []byte
	grants         map[int]bool
}

func NewAIImportHandler(database *db.Database) *AIImportHandler {
	h := &AIImportHandler{DB: database, enabled: os.Getenv("AI_IMPORT_ENABLED") == "true", grants: map[int]bool{}}
	if !h.enabled {
		return h
	}
	path := os.Getenv("AI_IMPORT_TOKEN_FILE")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 128 || runtime.GOOS == "linux" && info.Mode().Perm()&0007 != 0 {
		return h
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return h
	}
	h.token, err = hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(h.token) != 32 {
		return h
	}
	if snapshotFile := os.Getenv("AI_SERVICE_TOKEN_FILE"); snapshotFile != "" {
		if snapshotToken, readErr := os.ReadFile(snapshotFile); readErr == nil && subtle.ConstantTimeCompare([]byte(strings.TrimSpace(string(snapshotToken))), []byte(strings.TrimSpace(string(raw)))) == 1 {
			return h
		}
	}
	for _, part := range strings.Split(os.Getenv("AI_IMPORT_ALLOWED_WORKSPACE_IDS"), ",") {
		id, e := strconv.Atoi(strings.TrimSpace(part))
		if e != nil || id <= 0 {
			h.grants = map[int]bool{}
			return h
		}
		h.grants[id] = true
	}
	h.ready = len(h.grants) > 0
	return h
}
func (h *AIImportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	analysisRoute := strings.HasPrefix(r.URL.Path, "/internal/ai/v1/analyses/")
	if r.Method != http.MethodPost && !(analysisRoute && r.Method == http.MethodGet) {
		w.Header().Set("Allow", "POST")
		aiError(w, 405)
		return
	}
	if !h.enabled {
		aiError(w, 404)
		return
	}
	if !h.ready || h.DB == nil {
		aiError(w, 503)
		return
	}
	parts := strings.Split(r.Header.Get("Authorization"), " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		aiError(w, 401)
		return
	}
	token, e := hex.DecodeString(parts[1])
	if e != nil || len(token) != 32 || subtle.ConstantTimeCompare(token, h.token) != 1 {
		aiError(w, 401)
		return
	}
	if analysisRoute {
		h.serveAnalysis(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256*1024)
	var d db.AIDraft
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if e = dec.Decode(&d); e != nil {
		aiError(w, 400)
		return
	}
	if dec.Decode(new(any)) != io.EOF || !validateAIDraft(d) {
		aiError(w, 400)
		return
	}
	if !h.grants[d.WorkspaceID] {
		aiError(w, 403)
		return
	}
	e = h.DB.ImportAIPlan(r.Context(), d)
	switch {
	case errors.Is(e, db.ErrAIPlanStale), errors.Is(e, db.ErrAIPlanConflict):
		aiError(w, 409)
	case errors.Is(e, db.ErrAIPlanInvalid):
		aiError(w, 400)
	case errors.Is(e, sql.ErrNoRows):
		aiError(w, 404)
	case e != nil:
		aiError(w, 503)
	default:
		WriteJSON(w, 200, map[string]string{"id": d.ID, "status": "published"})
	}
}

type AIPlansController struct{ DB *db.Database }

func (c *AIPlansController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		aiError(w, 401)
		return
	}
	path := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(path) < 3 || path[0] != "ai-plans" || path[1] != "workspaces" {
		aiError(w, 404)
		return
	}
	workspace, e := strconv.Atoi(path[2])
	if e != nil || workspace <= 0 || strconv.Itoa(workspace) != path[2] {
		aiError(w, 400)
		return
	}
	if !aiOneOf(strings.ToLower(claims.RoleName), "management", "admin") {
		aiError(w, 403)
		return
	}
	manager, e := c.DB.WorkspaceAccess(r.Context(), claims.ID, workspace)
	if errors.Is(e, db.ErrWorkspaceForbidden) || !manager && e == nil {
		aiError(w, 403)
		return
	}
	if e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			aiError(w, 404)
		} else {
			aiError(w, 503)
		}
		return
	}
	if len(path) == 4 && path[3] == "history" && r.Method == "GET" {
		offset, parseErr := strconv.Atoi(r.URL.Query().Get("offset"))
		if r.URL.Query().Get("offset") == "" {
			offset = 0
			parseErr = nil
		}
		if parseErr != nil || offset < 0 || offset > 10000 {
			aiError(w, 400)
			return
		}
		items, e := c.DB.AIPlanHistory(r.Context(), workspace, 20, offset)
		if e != nil {
			aiError(w, 503)
			return
		}
		WriteJSON(w, 200, map[string]any{"drafts": items})
		return
	}
	if len(path) == 3 && r.Method == "GET" {
		if dates, selected := r.URL.Query()["date"]; selected {
			if len(dates) != 1 || !aiValidDate(dates[0]) {
				aiError(w, 400)
				return
			}
			id, lookupErr := c.DB.AIPlanIDByDate(r.Context(), workspace, dates[0])
			if errors.Is(lookupErr, sql.ErrNoRows) {
				WriteJSON(w, 200, map[string]any{"plan": nil})
				return
			}
			if lookupErr != nil {
				aiError(w, 503)
				return
			}
			plan, detailErr := c.DB.AIPlanDetail(r.Context(), workspace, id)
			if detailErr != nil {
				aiError(w, 503)
				return
			}
			WriteJSON(w, 200, map[string]any{"plan": plan})
			return
		}
		items, e := c.DB.AIPlanHistory(r.Context(), workspace, 1, 0)
		if e != nil {
			aiError(w, 503)
			return
		}
		if len(items) == 0 {
			WriteJSON(w, 200, map[string]any{"plan": nil})
			return
		}
		plan, e := c.DB.AIPlanDetail(r.Context(), workspace, items[0].ID)
		if e != nil {
			aiError(w, 503)
			return
		}
		WriteJSON(w, 200, map[string]any{"plan": plan})
		return
	}
	if len(path) >= 4 && !aiDigest.MatchString(path[3]) {
		aiError(w, 400)
		return
	}
	if len(path) == 4 && r.Method == "GET" {
		plan, e := c.DB.AIPlanDetail(r.Context(), workspace, path[3])
		if e != nil {
			if errors.Is(e, sql.ErrNoRows) {
				aiError(w, 404)
			} else {
				aiError(w, 503)
			}
			return
		}
		WriteJSON(w, 200, map[string]any{"plan": plan})
		return
	}
	if len(path) == 7 && path[4] == "recommendations" && path[6] == "review" && r.Method == "PATCH" {
		if !aiDigest.MatchString(path[5]) {
			aiError(w, 400)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var input struct {
			Decision string `json:"decision"`
			Note     string `json:"note"`
			Version  *int   `json:"version"`
		}
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if dec.Decode(&input) != nil || dec.Decode(new(any)) != io.EOF || input.Version == nil || *input.Version < 0 || !aiOneOf(input.Decision, "PENDING", "SUITABLE", "NEEDS_REVISION", "REJECTED") || len([]rune(input.Note)) > 2000 {
			aiError(w, 400)
			return
		}
		review, e := c.DB.ReviewAIPlan(r.Context(), workspace, path[3], path[5], claims.ID, input.Decision, input.Note, *input.Version)
		if e != nil {
			if errors.Is(e, db.ErrAIPlanConflict) || errors.Is(e, db.ErrAIPlanStale) {
				aiError(w, 409)
			} else if errors.Is(e, db.ErrWorkspaceForbidden) {
				aiError(w, 403)
			} else if errors.Is(e, sql.ErrNoRows) {
				aiError(w, 404)
			} else {
				aiError(w, 503)
			}
			return
		}
		WriteJSON(w, 200, map[string]any{"review": review})
		return
	}
	aiError(w, 404)
}
