package updater

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLatestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v1.7.1"}`))
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
