package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"my-ple-backend/db"
	"my-ple-backend/model"
)

func checklistTestDB(t *testing.T) *db.Database {
	t.Helper()
	dsn := os.Getenv("CHECKLIST_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("CHECKLIST_TEST_DATABASE_URL not set")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Hostname() != "127.0.0.1" || u.Path != "/my_ple_checklist_test" {
		t.Fatal("requires isolated local my_ple_checklist_test database")
	}
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	database := db.NewDatabase(conn)
	for i := 0; i < 2; i++ {
		if err = database.Migrate(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	return database
}
func checklistTestUser(t *testing.T, database *db.Database, role, username string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("Checklist-test-only-2026"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Conn.Exec(`INSERT INTO users(id,name,email,role_id,username,password_hash) SELECT $1,$2::text,$3,id,$4,$5 FROM roles WHERE name=$2::text ON CONFLICT(username) DO UPDATE SET password_hash=EXCLUDED.password_hash`, db.NewID(), role, username+"@example.test", username, string(hash))
	if err != nil {
		t.Fatal(err)
	}
}

func TestDocumentChecklistIntegration(t *testing.T) {
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
		if path == "/auth/login" && want == 200 {
			cookies = res.Cookies()
		}
		result := map[string]json.RawMessage{}
		_ = json.Unmarshal(raw, &result)
		return result
	}
	request("GET", "/document-checklists", nil, 401)
	request("GET", "/document-checklists/1/pdf", nil, 401)
	for i, role := range []string{"Project Engineer", "Finance", "Admin", "Management", "HR", "IT", "PMO", "Business Development"} {
		username := fmt.Sprintf("checklist-api-%d", i)
		checklistTestUser(t, database, role, username)
		request("POST", "/auth/login", map[string]string{"username": username, "password": "Checklist-test-only-2026"}, 200)
		want := 403
		if model.CanUseDocumentChecklist(role) {
			want = 200
		}
		request("GET", "/document-checklists", nil, want)
		if want == 403 {
			request("GET", "/document-checklists/1/pdf", nil, 403)
			for _, method := range []string{"GET", "POST", "PUT"} {
				request(method, "/document-checklists/1", nil, 403)
			}
			for _, method := range []string{"POST", "PATCH", "DELETE"} {
				request(method, "/document-checklists/1/tables/1/items/1", nil, 403)
			}
		}
	}
	request("POST", "/auth/login", map[string]string{"username": "checklist-api-2", "password": "Checklist-test-only-2026"}, 200)
	createJob := func(name string) int {
		var id int
		err := database.Conn.QueryRow(`INSERT INTO jobs(division_id,role_id,name) SELECT division_id,id,$1 FROM report_roles WHERE name='PE' ORDER BY id LIMIT 1 RETURNING id`, name+db.NewID()).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	jobA, jobB := createJob("Checklist A "), createJob("Checklist B ")
	a, b := fmt.Sprintf("/document-checklists/%d", jobA), fmt.Sprintf("/document-checklists/%d", jobB)
	var jobs []model.Job
	result := request("GET", "/document-checklists", nil, 200)
	if err := json.Unmarshal(result["jobs"], &jobs); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, j := range jobs {
		if j.Role != "Project Engineer" {
			t.Fatalf("ineligible job: %+v", j)
		}
		found = found || j.ID == jobA
	}
	if !found {
		t.Fatal("new database job missing")
	}
	for _, role := range []string{"QA", "CIHPK"} {
		var excluded int
		err := database.Conn.QueryRow(`INSERT INTO jobs(division_id,role_id,name) SELECT division_id,id,$1 FROM report_roles WHERE name=$2 ORDER BY id LIMIT 1 RETURNING id`, "Excluded "+db.NewID(), role).Scan(&excluded)
		if err != nil {
			t.Fatal(err)
		}
		path := fmt.Sprintf("/document-checklists/%d", excluded)
		for _, method := range []string{"GET", "POST", "PUT"} {
			request(method, path, nil, 404)
		}
		var visible []model.Job
		if err := json.Unmarshal(request("GET", "/document-checklists", nil, 200)["jobs"], &visible); err != nil {
			t.Fatal(err)
		}
		for _, j := range visible {
			if j.ID == excluded {
				t.Fatal("excluded role visible")
			}
		}
	}
	request("GET", "/document-checklists/2147483647", nil, 404)
	if string(request("GET", a, nil, 200)["checklists"]) != "[]" {
		t.Fatal("new job already configured")
	}
	decode := func(result map[string]json.RawMessage) model.DocumentChecklist {
		var c model.DocumentChecklist
		if err := json.Unmarshal(result["checklist"], &c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	signer := func(kind, name, position string) model.ChecklistSigner {
		return model.ChecklistSigner{Type: kind, Name: name, Position: position}
	}
	config := model.ChecklistConfigInput{Title: "First table", RequireDigitalSignature: true, InternalCount: 2, ExternalCount: 1, Signers: []model.ChecklistSigner{signer("internal", "Budi", "Project Manager"), signer("internal", "Andi", "Project Engineer"), signer("external", "Hendrik", "Client")}}
	invalid := config
	invalid.InternalCount = -1
	request("POST", a, invalid, 400)
	request("POST", a, map[string]any{"internalCount": 1.5}, 400)
	c := decode(request("POST", a, config, 200))
	request("GET", b+"/pdf", nil, 409)
	request("POST", a+"/pdf", nil, 405)
	second := decode(request("POST", a, config, 200))
	var tables []model.DocumentChecklist
	if err := json.Unmarshal(request("GET", a, nil, 200)["checklists"], &tables); err != nil {
		t.Fatal(err)
	}
	if len(tables) != 2 || tables[0].ID != c.ID || tables[1].ID != second.ID {
		t.Fatal("multiple table ordering")
	}
	req, _ := http.NewRequest("GET", server.URL+a+"/pdf", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil || res.StatusCode != 200 || res.Header.Get("Content-Type") != "application/pdf" || !bytes.HasPrefix(data, []byte("%PDF-")) || bytes.Count(data, []byte("/Type /Page\n")) != 2 {
		t.Fatal("per-job PDF must include both tables", err)
	}
	secondPath := fmt.Sprintf("%s/tables/%d", a, second.ID)
	a = fmt.Sprintf("%s/tables/%d", a, c.ID)
	foreign := config
	foreign.Version = c.Version
	request("PUT", fmt.Sprintf("%s/tables/%d", b, c.ID), foreign, 404)
	request("GET", fmt.Sprintf("%s/tables/%d", b, c.ID), nil, 404)
	configB := model.ChecklistConfigInput{Title: "Other job", InternalCount: 1, ExternalCount: 2, Signers: []model.ChecklistSigner{signer("internal", "Internal B", "PM"), signer("external", "Client B1", "Client"), signer("external", "Client B2", "Client")}}
	cb := decode(request("POST", b, configB, 200))
	b = fmt.Sprintf("%s/tables/%d", b, cb.ID)
	for _, name := range []string{"Berita Acara", "Laporan Mingguan", "Laporan Final"} {
		c = decode(request("POST", a+"/items", model.ChecklistItemPatch{Field: "documentName", Value: name}, 200))
	}
	itemID := c.Items[0].ID
	request("PATCH", fmt.Sprintf("%s/items/%d", secondPath, itemID), model.ChecklistItemPatch{Field: "notes", Value: "cross-table"}, 404)
	request("PATCH", fmt.Sprintf("%s/items/%d", a, itemID), model.ChecklistItemPatch{Field: "signature", SignerID: second.Signers[0].ID, Checked: true}, 404)
	itemPath := fmt.Sprintf("%s/items/%d", a, itemID)
	for _, p := range []model.ChecklistItemPatch{{Field: "digitalSigned", Checked: true}, {Field: "signature", SignerID: c.Signers[0].ID, Checked: true}, {Field: "signature", SignerID: c.Signers[1].ID, Checked: true}, {Field: "documentPosition", Value: "Di Client"}, {Field: "sentDate", Value: "2026-09-14"}, {Field: "notes", Value: "Menunggu tanda tangan client"}} {
		request("PATCH", itemPath, p, 200)
	}
	request("PATCH", itemPath, model.ChecklistItemPatch{Field: "signature", SignerID: cb.Signers[0].ID, Checked: true}, 404)
	request("PATCH", fmt.Sprintf("%s/items/%d", b, itemID), model.ChecklistItemPatch{Field: "notes", Value: "cross-job"}, 404)
	request("PATCH", itemPath, model.ChecklistItemPatch{Field: "sentDate", Value: "2026-02-30"}, 400)
	c = decode(request("GET", a, nil, 200))
	item := c.Items[0]
	if !item.DigitalSigned || !item.Signatures[c.Signers[0].ID] || !item.Signatures[c.Signers[1].ID] || item.Signatures[c.Signers[2].ID] || item.DocumentPosition != "Di Client" || item.SentDate == nil || *item.SentDate != "2026-09-14" || item.Notes != "Menunggu tanda tangan client" {
		t.Fatalf("lost cells after reload: %+v", item)
	}
	middle, last := c.Items[1].ID, c.Items[2].ID
	c = decode(request("DELETE", fmt.Sprintf("%s/items/%d", a, middle), nil, 200))
	if len(c.Items) != 2 || c.Items[0].ID != itemID || c.Items[1].ID != last {
		t.Fatal("delete/order changed IDs")
	}
	request("PATCH", fmt.Sprintf("%s/items/%d", a, middle), model.ChecklistItemPatch{Field: "notes", Value: "deleted"}, 404)
	config.Title = "Renamed table"
	config.Version = c.Version
	config.Signers = c.Signers
	oldID := c.Signers[0].ID
	config.Signers[0].Name = "Pengganti Budi"
	c = decode(request("PUT", a, config, 200))
	request("PUT", a, config, 409)
	if c.Title != "Renamed table" || decode(request("GET", secondPath, nil, 200)).Title != "First table" {
		t.Fatal("table title isolation")
	}
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if decode(request("GET", a, nil, 200)).Title != "Renamed table" {
		t.Fatal("migration reset title")
	}
	archived := false
	for _, s := range c.Signers {
		if s.ID == oldID {
			archived = s.Archived
		}
		if s.Name == "Pengganti Budi" && c.Items[0].Signatures[s.ID] {
			t.Fatal("replacement inherited signature")
		}
	}
	if !archived || !c.Items[0].Signatures[oldID] {
		t.Fatal("history lost")
	}
	// Invalid foreign signer must roll back the entire configuration update.
	invalid = configB
	invalid.Version = cb.Version
	invalid.Signers = append([]model.ChecklistSigner{}, cb.Signers...)
	invalid.Signers[0].ID = oldID
	request("PUT", b, invalid, 400)
	cb = decode(request("GET", b, nil, 200))
	if cb.Version != 1 || cb.RequireDigitalSignature || len(cb.Items) != 0 {
		t.Fatal("job B isolation/transaction rollback")
	}
	cb = decode(request("POST", b+"/items", model.ChecklistItemPatch{Field: "documentName", Value: "B doc"}, 200))
	request("PATCH", fmt.Sprintf("%s/items/%d", b, cb.Items[0].ID), model.ChecklistItemPatch{Field: "digitalSigned", Checked: true}, 400)
}

// Opt-in real browser fixture server; it only runs against the guarded test database.
func TestDocumentChecklistBrowserServer(t *testing.T) {
	if os.Getenv("CHECKLIST_BROWSER_SERVER") != "1" {
		t.Skip("browser fixture server disabled")
	}
	database := checklistTestDB(t)
	for _, role := range []string{"Admin", "Finance", "Project Engineer", "Management", "HR"} {
		checklistTestUser(t, database, role, "browser-"+role)
	}
	server := &http.Server{Addr: "127.0.0.1:3001", Handler: newHandler(database, "http://127.0.0.1:5189"), ReadHeaderTimeout: 5 * time.Second}
	t.Log("Checklist browser fixture ready on 3001")
	if err := server.ListenAndServe(); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentChecklistMigration(t *testing.T) {
	database := checklistTestDB(t)
	tx, err := database.Conn.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	// A rolled-back schema keeps this legacy-data fixture isolated from other tests.
	schema := "checklist_migration_" + strings.ReplaceAll(db.NewID(), "-", "")
	if _, err = tx.Exec(`CREATE SCHEMA ` + schema + `; SET LOCAL search_path TO ` + schema + `; CREATE TABLE jobs(id INTEGER PRIMARY KEY); INSERT INTO jobs VALUES(1)`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("db/document_checklists.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`
 INSERT INTO document_checklists(id,job_id,title,version) VALUES(10,1,'Legacy',3);
 INSERT INTO document_checklist_signers(id,checklist_id,signer_type,name,position,display_order,archived_at) VALUES(20,10,'internal','Signer','PM',0,NOW());
 INSERT INTO document_checklist_items(id,checklist_id,document_name,notes) VALUES(30,10,'Document','Preserved');
 INSERT INTO document_checklist_signatures(checklist_id,item_id,signer_id,signed) VALUES(10,30,20,true);
 ALTER TABLE document_checklists DROP COLUMN title;
 ALTER TABLE document_checklists ADD CONSTRAINT document_checklists_job_id_key UNIQUE(job_id);
 `); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = tx.Exec(string(migration)); err != nil {
			t.Fatal(err)
		}
	}
	var title, notes string
	var version int
	var signed, archived bool
	if err = tx.QueryRow(`SELECT c.title,c.version,i.notes,s.signed,p.archived_at IS NOT NULL FROM document_checklists c JOIN document_checklist_items i ON i.checklist_id=c.id JOIN document_checklist_signatures s ON s.item_id=i.id JOIN document_checklist_signers p ON p.id=s.signer_id WHERE c.id=10 AND i.id=30 AND p.id=20`).Scan(&title, &version, &notes, &signed, &archived); err != nil {
		t.Fatal(err)
	}
	if title != "Checklist Dokumen" || version != 3 || notes != "Preserved" || !signed || !archived {
		t.Fatal("legacy data changed")
	}
	if _, err = tx.Exec(`INSERT INTO document_checklists(job_id,title) VALUES(1,'Second table')`); err != nil {
		t.Fatal(err)
	}
}
