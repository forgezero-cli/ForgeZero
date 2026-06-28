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
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetLatestVersionNilBody(t *testing.T) {
	old := httpGet
	defer func() { httpGet = old }()
	httpGet = func(string) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: nil}, nil
	}
	_, err := GetLatestVersion()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetLatestVersionBadJSON(t *testing.T) {
	old := httpGet
	defer func() { httpGet = old }()
	httpGet = func(string) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("not-json")),
		}, nil
	}
	_, err := GetLatestVersion()
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestGetLatestVersionEmptyTag(t *testing.T) {
	old := httpGet
	defer func() { httpGet = old }()
	httpGet = func(string) (*http.Response, error) {
		body := `{"tag_name":"v"}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	v, err := GetLatestVersion()
	if err != nil {
		t.Fatal(err)
	}
	if v != "" {
		t.Fatalf("got %q", v)
	}
}

func TestUpdateSelfDownloadHTTPError(t *testing.T) {
	old := httpGet
	oldURL := apiURL
	defer func() {
		httpGet = old
		apiURL = oldURL
	}()
	apiURL = "https://api.github.com/repos/forgezero-cli/ForgeZero/releases/latest"
	httpGet = func(url string) (*http.Response, error) {
		if strings.Contains(url, "releases/latest") {
			body := `{"tag_name":"v9.9.9"}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
		}
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("fail")),
		}, nil
	}
	if err := UpdateSelf("0.0.0"); err == nil {
		t.Fatal("expected download error")
	}
}

func TestUpdateSelfAssetTooLarge(t *testing.T) {
	old := httpGet
	oldURL := apiURL
	defer func() {
		httpGet = old
		apiURL = oldURL
	}()
	apiURL = "https://api.github.com/repos/forgezero-cli/ForgeZero/releases/latest"
	httpGet = func(url string) (*http.Response, error) {
		if strings.Contains(url, "releases/latest") {
			body := `{"tag_name":"v0.2.0"}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
		}
		h := make(http.Header)
		return &http.Response{
			StatusCode:    http.StatusOK,
			Body:          io.NopCloser(strings.NewReader("x")),
			Header:        h,
			ContentLength: int64(300 << 20),
		}, nil
	}
	err := UpdateSelf("0.0.0")
	if err == nil {
		t.Fatal("expected size error")
	}
}

func TestUpdateSelfNilDownloadBody(t *testing.T) {
	old := httpGet
	oldURL := apiURL
	defer func() { httpGet = old; apiURL = oldURL }()
	apiURL = "https://api.github.com/repos/forgezero-cli/ForgeZero/releases/latest"
	httpGet = func(url string) (*http.Response, error) {
		if strings.Contains(url, "releases/latest") {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v0.3.0"}`))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: nil}, nil
	}
	if err := UpdateSelf("0.0.0"); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateSelfGetVersionError(t *testing.T) {
	old := httpGet
	defer func() { httpGet = old }()
	httpGet = func(string) (*http.Response, error) {
		return nil, io.ErrUnexpectedEOF
	}
	if err := UpdateSelf("0.0.0"); err == nil {
		t.Fatal("expected error")
	}
}

func TestAssetNameWindows(t *testing.T) {
	name := assetName()
	if name == "" {
		t.Fatal("empty")
	}
}
