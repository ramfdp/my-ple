package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"my-ple-backend/db"
	"my-ple-backend/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBillingDocumentsIntegration(t *testing.T) {
	database := checklistTestDB(t)
	server := httptest.NewServer(newHandler(database, "http://test.local"))
	defer server.Close()
	var cookies []*http.Cookie
	request := func(method, path string, body any, want int) map[string]json.RawMessage {
		t.Helper()
		data, _ := json.Marshal(body)
		req, _ := http.NewRequest(method, server.URL+path, bytes.NewReader(data))
		for _, c := range cookies {
			req.AddCookie(c)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		if res.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, res.StatusCode, want, raw)
		}
		if path == "/auth/login" {
			cookies = res.Cookies()
		}
		out := map[string]json.RawMessage{}
		_ = json.Unmarshal(raw, &out)
		return out
	}
	base := "/billing-documents"
	request("GET", base, nil, 401)
	login := func(role string) {
		t.Helper()
		name := "billing-" + role
		checklistTestUser(t, database, role, name)
		request("POST", "/auth/login", map[string]string{"username": name, "password": "Checklist-test-only-2026"}, 200)
	}
	for _, role := range []string{"HR", "IT", "PMO", "Business Development"} {
		login(role)
		request("GET", base, nil, 403)
		request("POST", base+"/stages", map[string]any{}, 403)
	}
	decode := func(r map[string]json.RawMessage) model.BillingStage {
		t.Helper()
		var s model.BillingStage
		if err := json.Unmarshal(r["stage"], &s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	var job int
	if err := database.Conn.QueryRow(`INSERT INTO jobs(division_id,role_id,name) SELECT division_id,id,$1 FROM report_roles WHERE name='PE' ORDER BY id LIMIT 1 RETURNING id`, "Billing test "+db.NewID()).Scan(&job); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"Admin", "Finance", "Project Engineer", "Management"} {
		login(role)
		request("GET", base, nil, 200)
		config := model.BillingConfiguration{JobID: job, Name: role, PeriodLabel: "TW-1", Milestones: []model.BillingRequirement{{Name: "First", Documents: []model.BillingRequirement{{Name: "BAPP"}, {Name: "BAST"}}}, {Name: "Second", Documents: []model.BillingRequirement{{Name: "RAB Tagihan"}}}}}
		s := decode(request("POST", base+"/stages", config, 200))
		path := fmt.Sprintf("%s/stages/%d", base, s.ID)
		if s.Readiness != "BELUM SIAP" || s.Milestones[0].Documents[0].Status != "BELUM ADA" {
			t.Fatal("default", s)
		}
		for _, m := range s.Milestones {
			for _, d := range m.Documents {
				s = decode(request("PATCH", path, map[string]any{"version": s.Version, "documentId": d.ID, "status": "OK"}, 200))
			}
		}
		if s.Readiness != "SIAP" {
			t.Fatal("ready stage", s)
		}
		oldVersion := s.Version
		d := s.Milestones[0].Documents[0]
		s = decode(request("PATCH", path, map[string]any{"version": s.Version, "documentId": d.ID, "status": "KURANG LENGKAP"}, 200))
		if s.Readiness != "BELUM SIAP" || s.Milestones[0].Readiness != "BELUM SIAP" || s.Milestones[1].Readiness != "SIAP" {
			t.Fatal("readiness propagation", s)
		}
		request("PATCH", path, map[string]any{"version": oldVersion, "documentId": d.ID, "status": "OK"}, 409)
		request("PATCH", path, map[string]any{"version": s.Version, "documentId": d.ID, "status": "INVALID"}, 400)
		request("PATCH", path, map[string]any{"version": s.Version, "readiness": "SIAP"}, 400)
		s = decode(request("PATCH", path, map[string]any{"version": s.Version, "notes": "Stage note"}, 200))
		s = decode(request("PATCH", path, map[string]any{"version": s.Version, "milestoneId": s.Milestones[0].ID, "notes": "Milestone note"}, 200))
		if s.Notes != "Stage note" || s.Milestones[0].Notes != "Milestone note" {
			t.Fatal("notes lost")
		}
		config.Version = s.Version
		for i, m := range s.Milestones {
			config.Milestones[i].ID = m.ID
			for j, d := range m.Documents {
				config.Milestones[i].Documents[j].ID = d.ID
			}
		}
		removed := s.Milestones[0].Documents[0].ID
		config.Milestones[0].Documents = config.Milestones[0].Documents[1:]
		s = decode(request("PUT", path+"/configuration", config, 200))
		if s.Readiness != "SIAP" || s.Milestones[0].Documents[0].Status != "OK" {
			t.Fatal("archived requirement counted or status lost", s)
		}
		var archived bool
		if err := database.Conn.QueryRow(`SELECT archived_at IS NOT NULL FROM billing_documents WHERE id=$1`, removed).Scan(&archived); err != nil || !archived {
			t.Fatal("requirement not archived", err)
		}
		request("PATCH", path, map[string]any{"version": s.Version, "documentId": removed, "status": "OK"}, 404)
		config.Version = s.Version
		config.Milestones[0].Documents = append(config.Milestones[0].Documents, model.BillingRequirement{Name: "New document"})
		s = decode(request("PUT", path+"/configuration", config, 200))
		if s.Readiness != "BELUM SIAP" {
			t.Fatal("new requirement must start unavailable")
		}
		// Foreign IDs and invalid configurations must roll back instead of archiving current rows.
		other := decode(request("POST", base+"/stages", model.BillingConfiguration{JobID: job, Name: "Other", PeriodLabel: "TW-2", Milestones: []model.BillingRequirement{{Name: "Other", Documents: []model.BillingRequirement{{Name: "Other doc"}}}}}, 200))
		request("PATCH", path, map[string]any{"version": s.Version, "documentId": other.Milestones[0].Documents[0].ID, "status": "OK"}, 404)
		config.Version = s.Version
		config.Milestones[0].Documents[0].ID = other.Milestones[0].Documents[0].ID
		request("PUT", path+"/configuration", config, 400)
		config.Milestones = nil
		request("PUT", path+"/configuration", config, 400)
		var result []model.BillingStage
		if err := json.Unmarshal(request("GET", base, nil, 200)["stages"], &result); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, loaded := range result {
			if loaded.ID == s.ID {
				found = true
				if loaded.Version != s.Version || loaded.Notes != s.Notes || len(loaded.Milestones[0].Documents) != 2 {
					t.Fatal("reload/rollback lost data")
				}
			}
		}
		if !found {
			t.Fatal("stage missing")
		}
	}
}
