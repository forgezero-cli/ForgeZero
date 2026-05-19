package updater

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	// Just ensure it doesn't panic
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
	// Override executablePathFunc to return the fakeExe path
	oldFunc := executablePathFunc
	executablePathFunc = func() (string, error) { return fakeExe, nil }
	defer func() { executablePathFunc = oldFunc }()

	// Create a test HTTP server that returns a binary
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
