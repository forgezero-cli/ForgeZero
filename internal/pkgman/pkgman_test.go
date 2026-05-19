package pkgman

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePkgURL(t *testing.T) {
	tests := []struct {
		input    string
		repo     string
		tag      string
		hasError bool
	}{
		{"github.com/user/repo", "github.com/user/repo", "", false},
		{"github.com/user/repo@v1.0.0", "github.com/user/repo", "v1.0.0", false},
		{"https://github.com/user/repo", "github.com/user/repo", "", false},
		{"git@github.com:user/repo.git", "github.com/user/repo", "", false},
		{"invalid", "", "", true},
	}
	for _, tt := range tests {
		repo, tag, err := parsePkgURL(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("parsePkgURL(%q) expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parsePkgURL(%q) unexpected error: %v", tt.input, err)
		}
		if repo != tt.repo {
			t.Errorf("repo: got %q, want %q", repo, tt.repo)
		}
		if tag != tt.tag {
			t.Errorf("tag: got %q, want %q", tag, tt.tag)
		}
	}
}

func TestUpdateConfig(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Write initial .fz.yaml
	initialYAML := "source_dirs: []\noutput: test\n"
	if err := os.WriteFile(".fz.yaml", []byte(initialYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	pkgPath := "vendor/github.com/user/lib"
	// Add
	if err := updateConfig(pkgPath, true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(".fz.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), pkgPath) {
		t.Errorf("config does not contain %s", pkgPath)
	}
	// Remove
	if err := updateConfig(pkgPath, false); err != nil {
		t.Fatal(err)
	}
	data2, err := os.ReadFile(".fz.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data2), pkgPath) {
		t.Errorf("config still contains %s after removal", pkgPath)
	}
}

func TestCleanConfig(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	initialYAML := `
source_dirs:
  - vendor/github.com/user/lib
  - vendor/github.com/user/lib/subdir
  - src
output: test
`
	if err := os.WriteFile(".fz.yaml", []byte(initialYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgPath := "vendor/github.com/user/lib"
	if err := cleanConfig(pkgPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(".fz.yaml")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Contains(content, pkgPath) {
		t.Error("root path still present")
	}
	if strings.Contains(content, "subdir") {
		t.Error("subdir still present")
	}
	if !strings.Contains(content, "src") {
		t.Error("src missing")
	}
}

func TestFindPackagePath(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create fake vendor structure
	vendor := "vendor"
	os.MkdirAll(filepath.Join(vendor, "github.com", "user", "lib"), 0o755)
	gitPath := filepath.Join(vendor, "github.com", "user", "lib", ".git")
	if err := os.MkdirAll(gitPath, 0o755); err != nil {
		t.Fatal(err)
	}

	path, err := findPackagePath("user/lib")
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(vendor, "github.com", "user", "lib")
	if path != expected {
		t.Errorf("got %s, want %s", path, expected)
	}

	_, err = findPackagePath("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent package")
	}
}

func TestList(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// No packages
	List()
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No packages installed.") {
		t.Error("expected 'No packages installed.'")
	}
	r.Close()

	vendor := "vendor"
	os.MkdirAll(filepath.Join(vendor, "github.com", "user", "lib"), 0o755)
	os.MkdirAll(filepath.Join(vendor, "github.com", "user", "lib", ".git"), 0o755)

	r, w, _ = os.Pipe()
	os.Stdout = w
	List()
	w.Close()
	os.Stdout = oldStdout
	buf.Reset()
	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "github.com/user/lib") {
		t.Errorf("list output missing package: %s", out)
	}
	r.Close()
}

func TestAddAndRemove(t *testing.T) {
	t.Skip("integration test requires git and network; run manually if needed")
}
