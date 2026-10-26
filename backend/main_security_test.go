package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandlerSetsBrowserSecurityHeaders(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("COOKIE_SECURE", "false")
	recorder := httptest.NewRecorder()
	newHandler(nil, "http://localhost:5173").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	for name, want := range map[string]string{
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'; base-uri 'none'",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Permissions-Policy":      "camera=(), microphone=(), geolocation=()",
	} {
		if got := recorder.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestSecureCookieConfigurationFailsClosedInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "false")
	if _, err := secureCookieConfig(); err == nil {
		t.Fatal("production accepted insecure session cookies")
	}

	t.Setenv("COOKIE_SECURE", "true")
	if secure, err := secureCookieConfig(); err != nil || !secure {
		t.Fatalf("production secure cookie rejected: secure=%v err=%v", secure, err)
	}
}

func TestSecureCookieDefaultsToSecure(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("COOKIE_SECURE", "")
	if secure, err := secureCookieConfig(); err != nil || !secure {
		t.Fatalf("missing configuration did not fail secure: secure=%v err=%v", secure, err)
	}
}

func TestHSTSSetForSecureDeployment(t *testing.T) {
	recorder := httptest.NewRecorder()
	setSecurityHeaders(recorder, true)
	if got := recorder.Header().Get("Strict-Transport-Security"); got != "max-age=31536000; includeSubDomains" {
		t.Fatalf("Strict-Transport-Security = %q", got)
	}
}

func TestHTTPServerHasTimeouts(t *testing.T) {
	server := newHTTPServer(":3000", http.NotFoundHandler())
	for name, value := range map[string]time.Duration{
		"ReadHeaderTimeout": server.ReadHeaderTimeout,
		"ReadTimeout":       server.ReadTimeout,
		"WriteTimeout":      server.WriteTimeout,
		"IdleTimeout":       server.IdleTimeout,
	} {
		if value <= 0 {
			t.Errorf("%s is not configured", name)
		}
	}
}
