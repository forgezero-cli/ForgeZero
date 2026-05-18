package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const (
	repoOwner = "alexvoste"
	repoName  = "ForgeZero"
	apiURL    = "https://api.github.com/repos/%s/%s/releases/latest"
)

type Release struct {
	TagName string `json:"tag_name"`
}

func GetLatestVersion() (string, error) {
	url := fmt.Sprintf(apiURL, repoOwner, repoName)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", nil // no releases, treat as no update available
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}

func UpdateSelf(currentVersion string) error {
	latest, err := GetLatestVersion()
	if err != nil {
		return err
	}
	if latest != "" && latest == currentVersion {
		fmt.Println("Already up to date.")
		return nil
	}
	if latest != "" {
		fmt.Printf("New version available: %s (current: %s)\n", latest, currentVersion)
	} else {
		fmt.Printf("Current version: %s. No release tags found, will update to latest via 'go install'.\n", currentVersion)
	}

	goPath, err := exec.LookPath("go")
	if err != nil {
		fmt.Println("Go is not installed. Please update manually using:")
		fmt.Printf("  go install github.com/%s/%s/cmd/fz@latest\n", repoOwner, repoName)
		return nil
	}
	fmt.Print("Updating via 'go install'... ")
	cmd := exec.Command(goPath, "install", fmt.Sprintf("github.com/%s/%s/cmd/fz@latest", repoOwner, repoName))
	cmd.Env = append(
		os.Environ(),
		"GOOS="+runtime.GOOS,
		"GOARCH="+runtime.GOARCH,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("failed.")
		return err
	}
	fmt.Println("done.")
	fmt.Println("Update successful. Please restart fz.")
	return nil
}
