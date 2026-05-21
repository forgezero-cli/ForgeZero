package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	apiURL             = "https://api.github.com/repos/forgezero-cli/ForgeZero/releases/latest"
	executablePathFunc = os.Executable
	httpClient         = &http.Client{Timeout: 30 * time.Second}
	httpGet            = func(u string) (*http.Response, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
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
		return "", fmt.Errorf("fetch latest version: %w", err)
	}
	if resp == nil || resp.Body == nil {
		if resp != nil {
			_ = resp.Body.Close()
		}
		return "", errors.New("empty response from upstream")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var release Release
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := dec.Decode(&release); err != nil {
		return "", fmt.Errorf("decode release json: %w", err)
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}

func assetName() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	if osName == "windows" {
		return fmt.Sprintf("fz-%s-%s.exe", osName, arch)
	}
	return fmt.Sprintf("fz-%s-%s", osName, arch)
}

func UpdateSelf(currentVersion string) error {
	latest, err := GetLatestVersion()
	if err != nil {
		return fmt.Errorf("get latest version: %w", err)
	}
	if latest == currentVersion {
		fmt.Println("Already up to date.")
		return nil
	}
	fmt.Printf("New version available: %s (current: %s)\n", latest, currentVersion)
	asset := assetName()
	dl := fmt.Sprintf("https://github.com/forgezero-cli/ForgeZero/releases/download/v%s/%s", latest, asset)
	u, err := url.Parse(dl)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	if u.Scheme != "https" || !strings.Contains(u.Host, "github.com") {
		return fmt.Errorf("unsafe download host: %s", u.Host)
	}
	resp, err := httpGet(u.String())
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}
	if resp == nil || resp.Body == nil {
		if resp != nil {
			_ = resp.Body.Close()
		}
		return errors.New("empty download response")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	maxSize := int64(200 << 20)
	if resp.ContentLength > 0 && resp.ContentLength > maxSize {
		return fmt.Errorf("asset too large: %d bytes", resp.ContentLength)
	}
	exePath, err := executablePathFunc()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	exePath = filepath.Clean(exePath)
	if !filepath.IsAbs(exePath) {
		exePath, err = filepath.Abs(exePath)
		if err != nil {
			return fmt.Errorf("executable path abs: %w", err)
		}
	}
	dir := filepath.Dir(exePath)
	tmp, err := os.CreateTemp(dir, "fz_update_*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()
	lr := io.LimitReader(resp.Body, maxSize+1)
	if _, err := io.Copy(tmp, lr); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if fi, err := tmp.Stat(); err == nil {
		if fi.Size() > maxSize {
			return fmt.Errorf("download exceeded max size: %d", fi.Size())
		}
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	mode := os.FileMode(0o755)
	if st, err := os.Stat(exePath); err == nil {
		mode = st.Mode()
	}
	if err := tmp.Chmod(mode); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	backupPath := exePath + ".old"
	if _, err := os.Stat(exePath); err == nil {
		if err := os.Rename(exePath, backupPath); err != nil {
			return fmt.Errorf("create backup: %w", err)
		}
	}
	if err := os.Rename(tmpPath, exePath); err != nil {
		if _, statErr := os.Stat(backupPath); statErr == nil {
			_ = os.Rename(backupPath, exePath)
		}
		return fmt.Errorf("install update: %w", err)
	}
	if err := os.Chmod(exePath, mode); err != nil {
		return fmt.Errorf("set executable mode: %w", err)
	}
	fmt.Printf("Update successful: %s\n", exePath)
	return nil
}
