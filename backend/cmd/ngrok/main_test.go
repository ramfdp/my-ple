package main

import (
	"context"
	"testing"
)

func TestRejectInvalidConfigBeforeConnecting(t *testing.T) {
	t.Setenv("NGROK_AUTHTOKEN", "")
	if err := run(context.Background()); err == nil {
		t.Fatal("missing token accepted")
	}
	t.Setenv("NGROK_AUTHTOKEN", "test-token")
	for _, value := range []string{"", "http://example.com", "https://", "https://example.com/", "https://example.com/api", "https://user@example.com", "https://example.com?x=1", "https://example.com#fragment"} {
		t.Setenv("NGROK_URL", value)
		if err := run(context.Background()); err == nil {
			t.Fatalf("invalid URL accepted: %q", value)
		}
	}
}
