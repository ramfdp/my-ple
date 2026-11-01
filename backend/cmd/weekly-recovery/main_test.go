package main

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestRecoveryOptionsRequireReviewedApply(t *testing.T) {
	for _, args := range [][]string{{"-apply"}, {"-apply", "-expected-count", "1"}, {"-fingerprint", strings.Repeat("a", 64)}, {"extra"}, {"-apply", "-expected-count", "-1", "-fingerprint", strings.Repeat("a", 64)}, {"-apply", "-expected-count", "1", "-fingerprint", strings.Repeat("z", 64)}} {
		if _, err := parseOptions(args); err == nil {
			t.Fatalf("accepted unsafe arguments %v", args)
		}
	}
	preview, err := parseOptions(nil)
	if err != nil || preview.apply {
		t.Fatalf("default is not preview: %+v %v", preview, err)
	}
	apply, err := parseOptions([]string{"-apply", "-expected-count", "1", "-fingerprint", strings.Repeat("a", 64)})
	if err != nil || !apply.apply || apply.count != 1 {
		t.Fatalf("valid apply rejected: %+v %v", apply, err)
	}
}

func TestInvalidRecoveryFlagsFailBeforeDatabase(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if err := run(context.Background(), []string{"-apply"}, io.Discard); err == nil || !strings.Contains(err.Error(), "requires -expected-count") {
		t.Fatalf("validation: %v", err)
	}
	if err := run(context.Background(), nil, io.Discard); err == nil || err.Error() != "DATABASE_URL is required" {
		t.Fatalf("database guard: %v", err)
	}
}
