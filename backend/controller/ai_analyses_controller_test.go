package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"my-ple-backend/db"
	"my-ple-backend/middleware"
)

func TestAnalysisBatchValidation(t *testing.T) {
	b := db.AIAnalysisBatch{WorkspaceID: 1, PlanDate: "2026-10-19", SnapshotHash: strings.Repeat("a", 64), PromptHash: strings.Repeat("b", 64), ModelMode: "mock", ModelProfile: "mock-v1", GeneratedAt: time.Now().UTC(), Items: []db.AIAnalysisItem{{TaskID: "550e8400-e29b-41d4-a716-446655440001", Summary: "Periksa laporan", NextSteps: []string{"Bandingkan angka"}, DocumentHash: strings.Repeat("c", 64), Decision: "PENDING"}}}
	b.ID = db.AnalysisBatchID(b)
	if !validAnalysisBatch(b) {
		t.Fatal("valid batch rejected")
	}
	b.Stale = true
	if validAnalysisBatch(b) {
		t.Fatal("read-only state accepted on import")
	}
	b.Stale = false
	b.Items[0].Decision = "APPROVED"
	if validAnalysisBatch(b) {
		t.Fatal("preapproved batch accepted")
	}
	b.Items[0].Decision = "PENDING"
	b.Items[0].DocumentHash = "not a digest"
	if validAnalysisBatch(b) {
		t.Fatal("invalid evidence hash accepted")
	}
}

func TestAnalysisReviewRequiresSession(t *testing.T) {
	route := middleware.RequireAuth(aiSessionStub{role: "Project Engineer"}, &AIAnalysesController{})
	w := httptest.NewRecorder()
	route.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ai-analyses/workspaces/1", nil))
	if w.Code != 401 {
		t.Fatalf("missing session: %d", w.Code)
	}
}

func TestApprovedDailyDraftRequiresAnalysisAndSteps(t *testing.T) {
	d := validTestDraft()
	d.PromptVersion = "daily-v4"
	d.AnalysisBatchID = strings.Repeat("c", 64)
	d.EvidenceStatus = "SUFFICIENT"
	d.Selection = &db.AISelection{Actionable: 1, Selected: 1}
	id := sha256.Sum256([]byte("1|2026-10-16|" + d.InputSnapshotHash + "|daily-v4|" + d.PromptHash + "|mock|mock-v1|" + d.AnalysisBatchID + "|1.0"))
	d.ID = hex.EncodeToString(id[:])
	task := "550e8400-e29b-41d4-a716-446655440001"
	recID := sha256.Sum256([]byte(d.ID + "|" + task + "|CONTINUE_EXISTING"))
	d.Recommendations = []db.AIRecommendation{{ID: hex.EncodeToString(recID[:]), Title: "Lanjutkan laporan", Description: "Periksa laporan", NextSteps: []string{"Bandingkan angka"}, Priority: "HIGH", SourceTaskIDs: []string{task}, RecommendationType: "CONTINUE_EXISTING", SuggestedDueDate: d.PlanDate, Reason: "Pekerjaan masih terbuka"}}
	if !validateAIDraft(d) {
		t.Fatal("approved daily draft rejected")
	}
	d.Recommendations[0].NextSteps = nil
	if validateAIDraft(d) {
		t.Fatal("draft without next steps accepted")
	}
	d.Recommendations[0].NextSteps = []string{"Bandingkan angka"}
	d.AnalysisBatchID = ""
	if validateAIDraft(d) {
		t.Fatal("draft without analysis batch accepted")
	}
}
