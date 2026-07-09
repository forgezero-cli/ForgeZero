/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package pkgman

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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

func TestParsePkgURLRejectsTraversal(t *testing.T) {
	repo, _, err := parsePkgURL("github.com/user/../evil")
	if err == nil {
		t.Fatalf("expected error for traversal repo, got repo %q", repo)
	}
}

func TestRemovePackageOutsideVendor(t *testing.T) {
	tmpDir := t.TempDir()
	defer chdirTemp(t, tmpDir)()

	bad := filepath.Join("..", "evil")
	if err := removePackage(bad); err == nil {
		t.Fatal("expected removePackage to reject outside vendor path")
	}
}

func TestUpdateConfigRejectsTraversalPath(t *testing.T) {
	tmpDir := t.TempDir()
	defer chdirTemp(t, tmpDir)()

	if err := updateConfig(filepath.Join("..", "evil"), true); err == nil {
		t.Fatal("expected traversal path error")
	}
}

func TestUpdateConfig(t *testing.T) {
	tmpDir := t.TempDir()
	defer chdirTemp(t, tmpDir)()

	initialYAML := "source_dirs: []\noutput: test\n"
	if err := os.WriteFile(".fz.yaml", []byte(initialYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	pkgPath := "vendor/github.com/user/lib"

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
	defer chdirTemp(t, tmpDir)()

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
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(oldWd))
	}()
	require.NoError(t, os.Chdir(tmpDir))
	vendor := "vendor"
	if err := os.MkdirAll(filepath.Join(vendor, "github.com", "user", "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
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
	oldWd, err := os.Getwd()

	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(oldWd))
	}()
	require.NoError(t, os.Chdir(tmpDir))

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := List(); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No packages installed.") {
		t.Error("expected 'No packages installed.'")
	}
	r.Close()

	vendor := "vendor"
	if err := os.MkdirAll(filepath.Join(vendor, "github.com", "user", "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(vendor, "github.com", "user", "lib", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	r, w, _ = os.Pipe()
	os.Stdout = w
	if err := List(); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdout = oldStdout
	buf.Reset()
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "github.com/user/lib") {
		t.Errorf("list output missing package: %s", out)
	}
	r.Close()
}

func TestAddAndRemove(t *testing.T) {
	t.Skip("integration test requires git and network; run manually if needed")
}

func TestGetCatalogURLs(t *testing.T) {
	urls := getCatalogURLs()
	if len(urls) == 0 {
		t.Error("getCatalogURLs returned empty slice")
	}
	os.Setenv("FZ_CATALOG_URL", "https://example.com/custom.json")
	defer os.Unsetenv("FZ_CATALOG_URL")
	urls2 := getCatalogURLs()
	if len(urls2) == 0 || urls2[0] != "https://example.com/custom.json" {
		t.Error("FZ_CATALOG_URL not respected")
	}
}

func TestFetchCatalogFromURLInvalid(t *testing.T) {
	_, err := fetchCatalogFromURL("http://localhost:9999/nonexistent")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestListCatalogWithMockHTTP(t *testing.T) {
	oldClient := httpClient
	defer func() { httpClient = oldClient }()
	httpClient = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"version":1,"packages":[{"name":"pkg","description":"desc","category":"cat"}]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	err := ListCatalog(context.Background())
	w.Close()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Available packages") {
		t.Fatal("expected catalog output")
	}
}

func TestSearchCatalogWithMockHTTP(t *testing.T) {
	oldClient := httpClient
	defer func() { httpClient = oldClient }()
	httpClient = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"version":1,"packages":[{"name":"pkg","description":"desc","category":"category"}]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	err := SearchCatalog(context.Background(), "pkg")
	w.Close()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "pkg (category)") {
		t.Fatal("expected search result output")
	}
}
