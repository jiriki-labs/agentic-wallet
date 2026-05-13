package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckAuthPermissions_BroadDirPermissions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("JIRIKI_HOME", tmp)

	// Set dir to 0755 (too broad)
	if err := os.Chmod(tmp, 0755); err != nil {
		t.Fatal(err)
	}

	if err := CheckAuthPermissions(); err == nil {
		t.Fatal("expected error for 0755 dir, got nil")
	}
}

func TestCheckAuthPermissions_BroadAuthFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("JIRIKI_HOME", tmp)

	// Set dir correctly
	if err := os.Chmod(tmp, 0700); err != nil {
		t.Fatal(err)
	}
	// Write auth file with 0644
	authPath := filepath.Join(tmp, "auth")
	if err := os.WriteFile(authPath, []byte("token"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CheckAuthPermissions(); err == nil {
		t.Fatal("expected error for 0644 auth file, got nil")
	}
}

func TestCheckAuthPermissions_Correct(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("JIRIKI_HOME", tmp)

	if err := os.Chmod(tmp, 0700); err != nil {
		t.Fatal(err)
	}
	authPath := filepath.Join(tmp, "auth")
	if err := os.WriteFile(authPath, []byte("token"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := CheckAuthPermissions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestStartupRefusedOnBroadPermissions asserts that a daemon-style boot check
// (CheckAuthPermissions) refuses to proceed when the auth file is world-readable.
// This is the unit-level equivalent of the acceptance test for Phase 1.5.
func TestStartupRefusedOnBroadPermissions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("JIRIKI_HOME", tmp)

	// Dir at 0700 (safe), auth file at 0644 (unsafe — broader than 0600)
	if err := os.Chmod(tmp, 0700); err != nil {
		t.Fatal(err)
	}
	authPath := filepath.Join(tmp, "auth")
	if err := os.WriteFile(authPath, []byte("mytoken"), 0644); err != nil {
		t.Fatal(err)
	}

	err := CheckAuthPermissions()
	if err == nil {
		t.Fatal("expected daemon-startup refusal for 0644 auth file, got nil error")
	}
	t.Logf("startup correctly refused: %v", err)
}
