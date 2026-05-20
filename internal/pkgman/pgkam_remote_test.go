package pkgman

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchCatalogFromURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cat := Catalog{
			Version: 1,
			Packages: []CatalogPackage{
				{Name: "test-pkg", Repo: "github.com/test/pkg"},
			},
		}
		json.NewEncoder(w).Encode(cat)
	}))
	defer server.Close()
	cat, err := fetchCatalogFromURL(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Packages) != 1 || cat.Packages[0].Name != "test-pkg" {
		t.Error("unexpected catalog content")
	}
}

func TestFetchCatalogFromURLNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	_, err := fetchCatalogFromURL(server.URL)
	if err == nil {
		t.Error("expected error for 404")
	}
}
