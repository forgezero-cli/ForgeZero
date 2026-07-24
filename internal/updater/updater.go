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
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	apiURL             = "https://api.github.com/repos/forgezero-cli/ForgeZero/releases/latest"
	executablePathFunc = os.Executable
	httpClient         = &http.Client{Timeout: 30 * time.Second}
	httpGet            = func(u string) (*http.Response, error) {
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, errors.New("create request: " + err.Error())
		}
		return httpClient.Do(req)
	}
)

type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func GetLatestVersion() (string, error) {
	resp, err := httpGet(apiURL)
	if err != nil {
		return "", errors.New("fetch latest version: " + err.Error())
	}
	if resp == nil {
		return "", errors.New("empty response from upstream")
	}
	if resp.Body == nil {
		return "", errors.New("empty response body")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("GitHub API returned " + resp.Status)
	}
	var release Release
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := dec.Decode(&release); err != nil {
		return "", errors.New("decode release json: " + err.Error())
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}

func assetName() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	if osName == "windows" {
		return "fz-" + osName + "-" + arch + ".exe"
	}
	return "fz-" + osName + "-" + arch
}

func UpdateSelf(currentVersion string) error {
	latest, err := GetLatestVersion()
	if err != nil {
		return errors.New("get latest version: " + err.Error())
	}
	if latest == currentVersion {
		_, _ = os.Stdout.WriteString("Already up to date.\n")
		return nil
	}
	_, _ = os.Stdout.WriteString("New version available: " + latest + " (current: " + currentVersion + ")\n")
	return installAsset(latest)
}

func InstallVersion(version string) error {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if version == "" {
		return errors.New("version required")
	}
	return installAsset(version)
}

func installAsset(latest string) error {
	asset := assetName()
	dl := "https://github.com/forgezero-cli/ForgeZero/releases/download/v" + latest + "/" + asset
	u, err := url.Parse(dl)
	if err != nil {
		return errors.New("invalid download URL: " + err.Error())
	}
	if u.Scheme != "https" || !strings.Contains(u.Host, "github.com") {
		return errors.New("unsafe download host: " + u.Host)
	}
	resp, err := httpGet(u.String())
	if err != nil {
		return errors.New("download binary: " + err.Error())
	}
	if resp == nil {
		return errors.New("empty download response")
	}
	if resp.Body == nil {
		return errors.New("empty download body")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("download failed: " + resp.Status)
	}
	maxSize := int64(200 << 20)
	if resp.ContentLength > 0 && resp.ContentLength > maxSize {
		return errors.New("asset too large: " + strconv.FormatInt(resp.ContentLength, 10) + " bytes")
	}
	exePath, err := resolveExecutable()
	if err != nil {
		return err
	}
	dir := filepath.Dir(exePath)
	tmp, err := os.CreateTemp(dir, "fz_update_*")
	if err != nil {
		return errors.New("create temp file: " + err.Error())
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()
	lr := io.LimitReader(resp.Body, maxSize+1)
	if _, err := io.Copy(tmp, lr); err != nil {
		return errors.New("write temp file: " + err.Error())
	}
	if fi, err := tmp.Stat(); err == nil {
		if fi.Size() > maxSize {
			return errors.New("download exceeded max size: " + strconv.FormatInt(fi.Size(), 10))
		}
	}
	if err := tmp.Sync(); err != nil {
		return errors.New("sync temp file: " + err.Error())
	}
	mode := os.FileMode(0o755)
	if st, err := os.Stat(exePath); err == nil {
		mode = st.Mode()
	}
	if err := tmp.Chmod(mode); err != nil {
		return errors.New("chmod temp file: " + err.Error())
	}
	if err := tmp.Close(); err != nil {
		return errors.New("close temp file: " + err.Error())
	}
	backupPath := exePath + ".old"
	if _, err := os.Stat(exePath); err == nil {
		if err := os.Rename(exePath, backupPath); err != nil {
			return errors.New("create backup: " + err.Error())
		}
	}
	if err := os.Rename(tmpPath, exePath); err != nil {
		if _, statErr := os.Stat(backupPath); statErr == nil {
			_ = os.Rename(backupPath, exePath)
		}
		return errors.New("install update: " + err.Error())
	}
	if err := os.Chmod(exePath, mode); err != nil {
		return errors.New("set executable mode: " + err.Error())
	}
	_, _ = os.Stdout.WriteString("Update successful: " + exePath + "\n")
	return nil
}

func resolveExecutable() (string, error) {
	exePath, err := executablePathFunc()
	if err != nil {
		return "", errors.New("locate executable: " + err.Error())
	}
	exePath = filepath.Clean(exePath)
	if !filepath.IsAbs(exePath) {
		exePath, err = filepath.Abs(exePath)
		if err != nil {
			return "", errors.New("executable path abs: " + err.Error())
		}
	}
	return exePath, nil
}

func RestoreBackup() error {
	exePath, err := resolveExecutable()
	if err != nil {
		return err
	}
	backupPath := exePath + ".old"
	if _, err := os.Stat(backupPath); err != nil {
		return errors.New("no backup available to roll back to")
	}
	swap := exePath + ".swap"
	if err := os.Rename(exePath, swap); err != nil {
		return errors.New("stage current binary: " + err.Error())
	}
	if err := os.Rename(backupPath, exePath); err != nil {
		_ = os.Rename(swap, exePath)
		return errors.New("restore backup: " + err.Error())
	}
	if err := os.Rename(swap, backupPath); err != nil {
		return errors.New("rotate backup: " + err.Error())
	}
	_, _ = os.Stdout.WriteString("Rolled back: " + exePath + "\n")
	return nil
}
