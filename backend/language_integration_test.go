package main

import (
	"bytes"
	"encoding/json"
	"io"
	"my-ple-backend/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLanguagePreference(t *testing.T) {
	database := checklistTestDB(t)
	server := httptest.NewServer(newHandler(database, "http://test.local"))
	defer server.Close()
	var cookies []*http.Cookie
	request := func(method, path string, input any, want int) map[string]json.RawMessage {
		t.Helper()
		data, _ := json.Marshal(input)
		req, _ := http.NewRequest(method, server.URL+path, bytes.NewReader(data))
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		if res.StatusCode != want {
			t.Fatalf("%s %s: %d %s", method, path, res.StatusCode, body)
		}
		if path == "/auth/login" {
			cookies = res.Cookies()
		}
		result := map[string]json.RawMessage{}
		_ = json.Unmarshal(body, &result)
		return result
	}
	request("PATCH", "/profile/language", map[string]string{"language": "en"}, 401)
	login := func(username string) {
		request("POST", "/auth/login", map[string]string{"username": username, "password": "Checklist-test-only-2026"}, 200)
	}
	user := func(body map[string]json.RawMessage) model.User {
		t.Helper()
		var value model.User
		if err := json.Unmarshal(body["user"], &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	checklistTestUser(t, database, "Admin", "language-admin")
	checklistTestUser(t, database, "Project Engineer", "language-pe")
	login("language-admin")
	if _, err := database.Conn.Exec(`UPDATE users SET residence='Existing residence',phone_number='+6281234567890' WHERE username='language-admin'`); err != nil {
		t.Fatal(err)
	}
	request("PATCH", "/profile/language", map[string]string{"language": "id"}, 200)
	before := user(request("GET", "/auth/me", nil, 200))
	for _, bad := range []string{"", "fr", "EN"} {
		request("PATCH", "/profile/language", map[string]string{"language": bad}, 400)
	}
	request("PATCH", "/profile/language", map[string]string{"language": "en", "userId": "another-user"}, 400)
	after := user(request("PATCH", "/profile/language", map[string]string{"language": "en"}, 200))
	if after.Language != "en" || after.PhoneNumber != before.PhoneNumber || after.Residence != before.Residence || after.RoleID != before.RoleID {
		t.Fatal("language update changed profile fields")
	}
	profile := user(request("PATCH", "/profile", map[string]string{"residence": "Updated residence", "phoneNumber": "+6281234567890"}, 200))
	if profile.Language != "en" {
		t.Fatal("profile save reset language")
	}
	login("language-admin")
	if user(request("GET", "/auth/me", nil, 200)).Language != "en" {
		t.Fatal("language not restored across login")
	}
	login("language-pe")
	if user(request("GET", "/auth/me", nil, 200)).Language != "id" {
		t.Fatal("language leaked to another account")
	}
	if user(request("PATCH", "/profile/language", map[string]string{"language": "en"}, 200)).Language != "en" {
		t.Fatal("regular user could not change language")
	}
	request("PATCH", "/profile/language", map[string]string{"language": "id"}, 200)
}
