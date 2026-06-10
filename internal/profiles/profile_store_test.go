package profiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndReadProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".profile.config")

	if err := SaveProfile(path, "performance"); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}
	p, err := ReadSavedProfile(path)
	if err != nil {
		t.Fatalf("ReadSavedProfile failed: %v", err)
	}
	if p != "performance" {
		t.Fatalf("expected performance, got %q", p)
	}
}

func TestReadSavedProfile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".profile.config")
	if err := os.WriteFile(path, []byte("\n\t"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	p, err := ReadSavedProfile(path)
	if err != nil {
		t.Fatalf("ReadSavedProfile failed: %v", err)
	}
	if p != "" {
		t.Fatalf("expected empty profile, got %q", p)
	}
}
