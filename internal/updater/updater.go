package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

var (
	apiURL             = "https://api.github.com/repos/forgezero-cli/ForgeZero/releases/latest"
	executablePathFunc = os.Executable
)

type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func GetLatestVersion() (string, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
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
		return err
	}
	if latest == currentVersion {
		fmt.Println("Already up to date.")
		return nil
	}
	fmt.Printf("New version available: %s (current: %s)\n", latest, currentVersion)

	asset := assetName()
	url := fmt.Sprintf("https://github.com/forgezero-cli/ForgeZero/releases/download/v%s/%s", latest, asset)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("could not download binary: %v", err)
	}
	defer resp.Body.Close()

	exePath, err := executablePathFunc()
	if err != nil {
		return err
	}
	backupPath := exePath + ".old"
	os.Rename(exePath, backupPath)
	out, err := os.Create(exePath)
	if err != nil {
		os.Rename(backupPath, exePath)
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Rename(backupPath, exePath)
		return err
	}
	os.Chmod(exePath, 0o755)
	fmt.Printf("Update successful. Backup saved as %s\n", backupPath)
	return nil
}
