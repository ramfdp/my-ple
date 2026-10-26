package main

import "testing"

func TestDatabaseURLRequiresEnvironment(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := databaseURL(); err == nil {
		t.Fatal("missing DATABASE_URL accepted")
	}

	t.Setenv("DATABASE_URL", "postgres://configured")
	if got, err := databaseURL(); err != nil || got != "postgres://configured" {
		t.Fatalf("configured DATABASE_URL rejected: got=%q err=%v", got, err)
	}
}
