package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndLoadManifest(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.txt")
	fileB := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(fileA, []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileB, []byte("bravo"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := WriteManifest(manifestPath, dir); err != nil {
		t.Fatal(err)
	}
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(manifest.Entries))
	}
}

func TestVerifyRootReportsDifferences(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "foo.txt")
	fileB := filepath.Join(dir, "bar.txt")
	if err := os.WriteFile(fileA, []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileB, []byte("bravo"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := WriteManifest(manifestPath, dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileA, []byte("alpha changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(fileB); err != nil {
		t.Fatal(err)
	}
	extraFile := filepath.Join(dir, "extra.txt")
	if err := os.WriteFile(extraFile, []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := VerifyRoot(dir, manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Modified) != 1 || result.Modified[0] != "foo.txt" {
		t.Fatalf("expected modified foo.txt, got %+v", result.Modified)
	}
	if len(result.Missing) != 1 || result.Missing[0] != "bar.txt" {
		t.Fatalf("expected missing bar.txt, got %+v", result.Missing)
	}
	if len(result.Extra) != 1 || result.Extra[0] != "extra.txt" {
		t.Fatalf("expected extra extra.txt, got %+v", result.Extra)
	}
}
