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
		if err := json.NewEncoder(w).Encode(cat); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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
