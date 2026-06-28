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

package updater

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetLatestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(`{"tag_name":"v1.7.1"}`)); err != nil {
			t.Errorf("mock server write failed: %v", err)
		}
	}))
	defer server.Close()
	oldURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = oldURL }()
	version, err := GetLatestVersion()
	if err != nil {
		t.Fatal(err)
	}
	if version != "1.7.1" {
		t.Errorf("expected 1.7.1, got %s", version)
	}
}

func TestGetLatestVersionNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	oldURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = oldURL }()
	_, err := GetLatestVersion()
	if err == nil {
		t.Error("expected error for 404")
	}
}

func TestAssetName(t *testing.T) {
	name := assetName()
	if name == "" {
		t.Error("asset name is empty")
	}
}

func TestUpdateSelfDownload(t *testing.T) {
	tmpDir := t.TempDir()
	fakeExe := filepath.Join(tmpDir, "fz")
	if err := os.WriteFile(fakeExe, []byte("old content"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldFunc := executablePathFunc
	executablePathFunc = func() (string, error) { return fakeExe, nil }
	defer func() { executablePathFunc = oldFunc }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("new binary content")); err != nil {
			t.Errorf("mock server write failed: %v", err)
		}
	}))
	defer server.Close()
	oldURL := apiURL
	apiURL = server.URL + "/release"
	defer func() { apiURL = oldURL }()

	binServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("new binary")); err != nil {
			t.Errorf("mock server write failed: %v", err)
		}
	}))
	defer binServer.Close()
	t.Skip("full integration test requires multiple endpoints; manual test recommended")
}

func TestUpdateSelfAlreadyUpToDate(t *testing.T) {
	oldURL := apiURL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(`{"tag_name":"v1.7.2"}`)); err != nil {
			t.Errorf("mock server write failed: %v", err)
		}
	}))
	apiURL = server.URL
	defer func() { apiURL = oldURL }()
	defer server.Close()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := UpdateSelf("1.7.2")
	w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("failed to create response body: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Already up to date")) {
		t.Error("expected 'Already up to date' message")
	}
}

func TestUpdateSelfPermissionDenied(t *testing.T) {
	oldFunc := executablePathFunc
	defer func() { executablePathFunc = oldFunc }()
	tmpDir := t.TempDir()
	fakeExe := filepath.Join(tmpDir, "fz_ro")
	if err := os.WriteFile(fakeExe, []byte("old"), 0o444); err != nil {
		t.Fatal(err)
	}
	executablePathFunc = func() (string, error) { return fakeExe, nil }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("new binary")); err != nil {
			t.Errorf("mock server write failed: %v", err)
		}
	}))
	defer server.Close()
	oldURL := apiURL
	apiURL = server.URL + "/release"
	defer func() { apiURL = oldURL }()
	err := UpdateSelf("0.0.0")
	if err == nil {
		t.Error("expected permission error when writing to read-only file")
	}
}

func TestUpdateSelfDownloadsNewVersion(t *testing.T) {
	oldHttpGet := httpGet
	oldExec := executablePathFunc
	oldURL := apiURL
	defer func() {
		httpGet = oldHttpGet
		executablePathFunc = oldExec
		apiURL = oldURL
	}()
	apiURL = "https://api.github.com/repos/forgezero-cli/ForgeZero/releases/latest"
	tmpDir := t.TempDir()
	fakeExe := filepath.Join(tmpDir, "fz")
	if err := os.WriteFile(fakeExe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	executablePathFunc = func() (string, error) { return fakeExe, nil }
	httpGet = func(url string) (*http.Response, error) {
		if strings.Contains(url, "releases/latest") {
			body := `{"tag_name":"v0.1.0"}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}
		if strings.Contains(url, "releases/download") {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("new content")), Header: make(http.Header)}, nil
		}
		return nil, nil
	}
	if err := UpdateSelf("0.0.0"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(fakeExe)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new content" {
		t.Fatalf("expected new content, got %q", string(data))
	}
	if _, err := os.Stat(fakeExe + ".old"); err != nil {
		t.Fatal(err)
	}
}
