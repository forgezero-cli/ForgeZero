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

func TestUpdateSelfExecutableNotAbs(t *testing.T) {
	oldExec := executablePathFunc
	defer func() { executablePathFunc = oldExec }()
	executablePathFunc = func() (string, error) {
		return "relative/fz", nil
	}
	old := httpGet
	oldURL := apiURL
	defer func() { httpGet = old; apiURL = oldURL }()
	apiURL = "https://api.github.com/repos/forgezero-cli/ForgeZero/releases/latest"
	httpGet = func(url string) (*http.Response, error) {
		if strings.Contains(url, "releases/latest") {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v1.0.0"}`))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("bin"))}, nil
	}
	if err := UpdateSelf("0.0.0"); err != nil {
		return
	}
}

func TestUpdateSelfInvalidDownloadURL(t *testing.T) {
	old := httpGet
	oldURL := apiURL
	defer func() { httpGet = old; apiURL = oldURL }()
	apiURL = "://bad-url"
	_, err := GetLatestVersion()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateSelfExceedsMaxDuringCopy(t *testing.T) {
	old := httpGet
	oldURL := apiURL
	defer func() { httpGet = old; apiURL = oldURL }()
	apiURL = "https://api.github.com/repos/forgezero-cli/ForgeZero/releases/latest"
	httpGet = func(url string) (*http.Response, error) {
		if strings.Contains(url, "releases/latest") {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v2.0.0"}`))}, nil
		}
		big := strings.Repeat("x", 200<<20+1)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(big))}, nil
	}
	if err := UpdateSelf("0.0.0"); err == nil {
		t.Fatal("expected size error")
	}
}

func TestGetLatestVersionHTTPError(t *testing.T) {
	old := httpGet
	defer func() { httpGet = old }()
	httpGet = func(string) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader("bad"))}, nil
	}
	_, err := GetLatestVersion()
	if err == nil {
		t.Fatal("expected error")
	}
}
