package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFZP(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.fz")
	if err := os.WriteFile(cfgPath, []byte("#define OUTPUT app\n#define MODE raw\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFZP(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "app" {
		t.Fatalf("Output = %q, want app", cfg.Output)
	}
	if cfg.Mode != "raw" {
		t.Fatalf("Mode = %q, want raw", cfg.Mode)
	}
}
