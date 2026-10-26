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

func TestInitialAdminPasswordRequiresStrongEnvironmentValue(t *testing.T) {
	t.Setenv("INITIAL_ADMIN_PASSWORD", "")
	if _, err := initialAdminPassword(); err == nil {
		t.Fatal("missing password was accepted")
	}

	t.Setenv("INITIAL_ADMIN_PASSWORD", "short")
	if _, err := initialAdminPassword(); err == nil {
		t.Fatal("short password was accepted")
	}

	t.Setenv("INITIAL_ADMIN_PASSWORD", "correct-horse-battery-staple")
	if _, err := initialAdminPassword(); err != nil {
		t.Fatalf("strong password was rejected: %v", err)
	}
}
