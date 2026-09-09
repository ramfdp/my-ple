package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"my-ple-backend/email"
)

func TestZoomEmailConfigurationKeepsHTTPAvailable(t *testing.T) {
	for k, v := range map[string]string{"SMTP_HOST": "smtp.gmail.com", "SMTP_FROM": "aiple5766@gmail.com", "SMTP_USERNAME": "aiple5766@gmail.com", "SMTP_PASSWORD": "test-placeholder", "SMTP_PORT": "587"} {
		t.Setenv(k, v)
	}
	for _, enabled := range []string{"false", "true"} {
		t.Setenv("ZOOM_EMAIL_ENABLED", enabled)
		if err := email.ValidateZoomConfig(); err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(newHandler(nil, "test", "http://localhost:5173"))
		response, err := http.Get(server.URL + "/health")
		if err != nil {
			server.Close()
			t.Fatal(err)
		}
		response.Body.Close()
		server.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatal(response.StatusCode)
		}
	}
}
