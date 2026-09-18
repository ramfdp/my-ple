package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"my-ple-backend/db"
	"my-ple-backend/model"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestBillingSharedIntegration(t *testing.T) {
	database := checklistTestDB(t)
	ctx := context.Background()
	var job int
	if err := database.Conn.QueryRow(`INSERT INTO jobs(division_id,role_id,name) SELECT division_id,id,$1 FROM report_roles WHERE name='PE' ORDER BY id LIMIT 1 RETURNING id`, "Shared billing "+db.NewID()).Scan(&job); err != nil {
		t.Fatal(err)
	}
	c := model.BillingConfiguration{JobID: job, Name: "Shared", PeriodLabel: "TW-1", Documents: []model.BillingRequirement{{Name: "BAST"}}, Milestones: []model.BillingRequirement{{Name: "Month 1"}, {Name: "Month 2"}}}
	id, err := database.ConfigureBilling(ctx, 0, c)
	if err != nil {
		t.Fatal(err)
	}
	load := func() model.BillingStage {
		t.Helper()
		stages, err := database.BillingStages(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range stages {
			if s.ID == id {
				return s
			}
		}
		t.Fatal("missing stage")
		return model.BillingStage{}
	}
	s := load()
	if len(s.Documents) != 1 || s.Readiness != "BELUM SIAP" {
		t.Fatal(s)
	}
	ok := "OK"
	if err = database.PatchBilling(ctx, id, model.BillingPatch{Version: s.Version, DocumentID: s.Documents[0].ID, Status: &ok}); err != nil {
		t.Fatal(err)
	}
	if err = database.PatchBilling(ctx, id, model.BillingPatch{Version: s.Version, DocumentID: s.Documents[0].ID, Status: &ok}); err != model.ErrBillingConflict {
		t.Fatalf("conflict: %v", err)
	}
	s = load()
	if s.Readiness != "SIAP" || s.Milestones[0].Readiness != "SIAP" {
		t.Fatal(s)
	}
	for i, m := range s.Milestones {
		c.Milestones[i].ID = m.ID
	}
	c.Version = s.Version
	c.Documents = nil // Legacy configuration must preserve shared requirements.
	if _, err = database.ConfigureBilling(ctx, id, c); err != nil {
		t.Fatal(err)
	}
	s = load()
	if len(s.Documents) != 1 || s.Documents[0].Status != "OK" {
		t.Fatal("legacy client lost shared status")
	}
	c.Version = s.Version
	c.Milestones[0].Documents = []model.BillingRequirement{{Name: " bast "}}
	if _, err = database.ConfigureBilling(ctx, id, c); err != model.ErrBillingInvalid {
		t.Fatalf("cross-scope duplicate: %v", err)
	}
	c.Milestones[0].Documents = nil
	other, err := database.ConfigureBilling(ctx, 0, model.BillingConfiguration{JobID: job, Name: "Other", PeriodLabel: "TW-2", Documents: []model.BillingRequirement{{Name: "BAST"}}, Milestones: []model.BillingRequirement{{Name: "Month"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = database.PatchBilling(ctx, other, model.BillingPatch{Version: 1, DocumentID: s.Documents[0].ID, Status: &ok}); err == nil {
		t.Fatal("cross-stage patch accepted")
	}
	bad := model.BillingConfiguration{JobID: job, Name: "Other", PeriodLabel: "TW-2", Version: 1, Documents: []model.BillingRequirement{{ID: s.Documents[0].ID, Name: "BAST"}}, Milestones: []model.BillingRequirement{{Name: "Month"}}}
	if _, err = database.ConfigureBilling(ctx, other, bad); err != model.ErrBillingInvalid {
		t.Fatalf("cross-stage configuration: %v", err)
	}
	removed := s.Documents[0].ID
	c.Documents = []model.BillingRequirement{}
	for i := range c.Milestones {
		c.Milestones[i].Documents = []model.BillingRequirement{{Name: "BAPP"}}
	}
	if _, err = database.ConfigureBilling(ctx, id, c); err != nil {
		t.Fatal(err)
	}
	s = load()
	if len(s.Documents) != 0 || s.Readiness != "BELUM SIAP" {
		t.Fatal(s)
	}
	var archived bool
	if err = database.Conn.QueryRow(`SELECT archived_at IS NOT NULL FROM billing_documents WHERE id=$1`, removed).Scan(&archived); err != nil || !archived {
		t.Fatal("shared document not archived", err)
	}
	if err = database.PatchBilling(ctx, id, model.BillingPatch{Version: s.Version, DocumentID: removed, Status: &ok}); err == nil {
		t.Fatal("archived document patched")
	}
}

func TestBillingSharedMigration(t *testing.T) {
	database := checklistTestDB(t)
	tx, err := database.Conn.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	schema := "billing_migration_" + strings.ReplaceAll(db.NewID(), "-", "")
	if _, err = tx.Exec(`CREATE SCHEMA ` + schema + `; SET LOCAL search_path TO ` + schema + `; CREATE TABLE jobs(id INTEGER PRIMARY KEY); INSERT INTO jobs VALUES(1)`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("db/billing_documents.sql")
	if err != nil {
		t.Fatal(err)
	}
	legacy := strings.Split(string(migration), "ALTER TABLE billing_documents ADD COLUMN")[0]
	if _, err = tx.Exec(legacy + `INSERT INTO billing_stages(id,job_id,name,period_label,notes,version) VALUES(10,1,'Legacy','TW-1','Keep',7); INSERT INTO billing_milestones(id,stage_id,name,sort_order) VALUES(20,10,'Month',0); INSERT INTO billing_documents(id,milestone_id,name,sort_order,status) VALUES(30,20,'BAST',0,'OK');`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = tx.Exec(string(migration)); err != nil {
			t.Fatal(err)
		}
	}
	var preserved bool
	if err = tx.QueryRow(`SELECT d.status='OK' AND d.milestone_id=20 AND d.stage_id IS NULL AND s.version=7 AND s.notes='Keep' FROM billing_documents d CROSS JOIN billing_stages s WHERE d.id=30 AND s.id=10`).Scan(&preserved); err != nil || !preserved {
		t.Fatal("migration changed legacy data", err)
	}
	if _, err = tx.Exec(`INSERT INTO billing_documents(stage_id,name,sort_order) VALUES(10,'RAB Tagihan',0)`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO billing_documents(stage_id,milestone_id,name,sort_order) VALUES(10,20,'Invalid owner',0)`); err == nil {
		t.Fatal("dual owner accepted")
	}
}

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
