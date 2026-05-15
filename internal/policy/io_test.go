package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndReadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	cfg := DefaultConfig()
	cfg.Mode = "auto"
	cfg.MaxAmountPerRequest = "10.50"

	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600, got %v", info.Mode().Perm())
	}

	got, err := ReadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != "auto" || got.MaxAmountPerRequest != "10.50" {
		t.Fatalf("unexpected config: %+v", got)
	}
}

func TestValidateConfigRejectsBadMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = "yolo"
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected error for invalid mode")
	}
}
