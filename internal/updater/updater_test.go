package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLatestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := Release{TagName: "v1.7.1"}
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()
	oldURL := apiURL
	defer func() { apiURL = oldURL }()
	apiURL = server.URL

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
	defer func() { apiURL = oldURL }()
	apiURL = server.URL

	version, err := GetLatestVersion()
	if err != nil {
		t.Fatal(err)
	}
	if version != "" {
		t.Errorf("expected empty version on 404, got %s", version)
	}
}
