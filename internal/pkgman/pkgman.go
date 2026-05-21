package pkgman

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fz/internal/utils"
	"gopkg.in/yaml.v3"
)

var httpClient = &http.Client{}

var runGit = func(ctx context.Context, args ...string) (string, error) {
	return utils.RunCommand(ctx, false, os.Stdout, os.Stderr, "git", args...)
}

const (
	vendorDir = "vendor"
)

type CatalogPackage struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Repo        string `json:"repo"`
	Tag         string `json:"tag"`
	Category    string `json:"category"`
	SourceDir   string `json:"source_dir"`
	Hash        string `json:"hash"`
}

type Catalog struct {
	Version  int              `json:"version"`
	Packages []CatalogPackage `json:"packages"`
}

func Add(ctx context.Context, pkgURL, version string) error {
	repo, tag, err := parsePkgURL(pkgURL)
	if err != nil {
		return err
	}
	if version != "" {
		tag = version
	}
	dest := filepath.Join(vendorDir, repo)
	if err := utils.SecureMkdirAll(dest); err != nil {
		return fmt.Errorf("prepare vendor dir: %w", err)
	}
	cloneURL := fmt.Sprintf("https://%s", repo)
	if _, err := runGit(ctx, "clone", cloneURL, dest); err != nil {
		return fmt.Errorf("git clone %s: %w", repo, err)
	}
	if tag != "" {
		if _, err := runGit(ctx, "-C", dest, "checkout", tag); err != nil {
			return fmt.Errorf("git checkout %s@%s: %w", repo, tag, err)
		}
	}
	if err := updateConfig(dest, true); err != nil {
		return err
	}
	fmt.Printf("Package %s installed.\n", pkgURL)
	return nil
}

func Remove(ctx context.Context, pkgURL string) error {
	_ = ctx
	repo, _, err := parsePkgURL(pkgURL)
	if err == nil {
		dest := filepath.Join(vendorDir, repo)
		if _, err := os.Stat(dest); err == nil {
			return removePackage(dest)
		}
	}
	dest, err := findPackagePath(pkgURL)
	if err != nil {
		return err
	}
	return removePackage(dest)
}

func removePackage(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove %s: %w", path, err)
	}
	dir := filepath.Dir(path)
	for dir != vendorDir && dir != "." {
		entries, err := os.ReadDir(dir)
		if err != nil {
			break
		}
		if len(entries) == 0 {
			if err := os.Remove(dir); err != nil {
				break
			}
			dir = filepath.Dir(dir)
		} else {
			break
		}
	}
	if err := cleanConfig(path); err != nil {
		return err
	}
	fmt.Printf("Package %s removed.\n", path)
	return nil
}

func cleanConfig(pkgPath string) error {
	data, err := os.ReadFile(".fz.yaml")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}
	dirsRaw, ok := cfg["source_dirs"]
	if !ok {
		return nil
	}
	dirsSlice, ok := dirsRaw.([]interface{})
	if !ok {
		return nil
	}
	newDirs := []interface{}{}
	for _, d := range dirsSlice {
		if str, ok := d.(string); ok {
			if !strings.HasPrefix(str, pkgPath) {
				newDirs = append(newDirs, str)
			}
		} else {
			newDirs = append(newDirs, d)
		}
	}
	cfg["source_dirs"] = newDirs
	newData, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return utils.SecureWriteFile(".fz.yaml", newData)
}

func List() error {
	if _, err := os.Stat(vendorDir); os.IsNotExist(err) {
		fmt.Println("No packages installed.")
		return nil
	}
	var packages []string
	err := filepath.Walk(vendorDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		gitPath := filepath.Join(path, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			rel, _ := filepath.Rel(vendorDir, path)
			packages = append(packages, rel)
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(packages) == 0 {
		fmt.Println("No packages installed.")
		return nil
	}
	for _, pkg := range packages {
		fmt.Println(pkg)
	}
	return nil
}

func Update(ctx context.Context) error {
	entries, err := os.ReadDir(vendorDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No packages to update.")
			return nil
		}
		return err
	}
	for _, entry := range entries {
		pkgPath := filepath.Join(vendorDir, entry.Name())
		if _, err := runGit(ctx, "-C", pkgPath, "pull"); err != nil {
			fmt.Printf("Warning: failed to update %s: %v\n", entry.Name(), err)
		} else {
			fmt.Printf("Updated %s\n", entry.Name())
		}
	}
	return nil
}

func getCatalogURLs() []string {
	urls := []string{
		"https://raw.githubusercontent.com/forgezero-cli/ForgeZero/refs/heads/main/catalog/catalog.json",
	}
	if envURL := os.Getenv("FZ_CATALOG_URL"); envURL != "" {
		urls = append([]string{envURL}, urls...)
	}
	urls = append(urls, "https://git.wienton.ru/alexvoste/Catalog/raw/branch/main/catalog.json")
	return urls
}

func fetchCatalogFromURL(url string) (*Catalog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "fz/2.0.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var cat Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, err
	}
	return &cat, nil
}

func fetchCatalog() (*Catalog, error) {
	var lastErr error
	for _, url := range getCatalogURLs() {
		cat, err := fetchCatalogFromURL(url)
		if err == nil {
			return cat, nil
		}
		lastErr = err
		fmt.Fprintf(os.Stderr, "Warning: failed to fetch catalog from %s: %v\n", url, err)
	}
	return nil, fmt.Errorf("all catalog URLs failed: %w", lastErr)
}

func ListCatalog(ctx context.Context) error {
	cat, err := fetchCatalog()
	if err != nil {
		return err
	}
	fmt.Printf("Available packages (catalog version %d):\n", cat.Version)
	for _, p := range cat.Packages {
		fmt.Printf("  %s (%s) - %s\n", p.Name, p.Category, p.Description)
	}
	return nil
}

func SearchCatalog(ctx context.Context, keyword string) error {
	cat, err := fetchCatalog()
	if err != nil {
		return err
	}
	fmt.Printf("Search results for '%s':\n", keyword)
	found := false
	for _, p := range cat.Packages {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(keyword)) ||
			strings.Contains(strings.ToLower(p.Description), strings.ToLower(keyword)) ||
			strings.Contains(strings.ToLower(p.Category), strings.ToLower(keyword)) {
			fmt.Printf("  %s (%s) - %s\n", p.Name, p.Category, p.Description)
			found = true
		}
	}
	if !found {
		fmt.Println("No matching packages found.")
	}
	return nil
}

func InstallFromCatalog(ctx context.Context, pkgName string) error {
	cat, err := fetchCatalog()
	if err != nil {
		return err
	}
	var pkg *CatalogPackage
	for _, p := range cat.Packages {
		if p.Name == pkgName {
			pkg = &p
			break
		}
	}
	if pkg == nil {
		return fmt.Errorf("package %s not found in catalog", pkgName)
	}
	repoWithTag := pkg.Repo
	if pkg.Tag != "" {
		repoWithTag += "@" + pkg.Tag
	}
	if err := Add(ctx, repoWithTag, ""); err != nil {
		return err
	}
	hashDirPath := filepath.Join(vendorDir, pkg.Repo)
	if pkg.SourceDir != "" {
		hashDirPath = filepath.Join(hashDirPath, pkg.SourceDir)
	}
	if pkg.Hash != "" {
		actualHash, err := utils.HashDir(hashDirPath)
		if err != nil {
			fmt.Printf("Warning: failed to compute hash for %s: %v\n", pkgName, err)
		} else if actualHash != pkg.Hash {
			_ = Remove(ctx, pkgName)
			return fmt.Errorf("hash mismatch for package %s (expected %s, got %s)", pkgName, pkg.Hash, actualHash)
		} else {
			fmt.Printf("Hash verification passed for %s\n", pkgName)
		}
	}
	if pkg.SourceDir != "" {
		rootPath := filepath.Join(vendorDir, pkg.Repo)
		subPath := filepath.Join(rootPath, pkg.SourceDir)
		if err := updateConfig(subPath, true); err != nil {
			return err
		}
	}
	fmt.Printf("Installed catalog package %s\n", pkgName)
	return nil
}

func parsePkgURL(raw string) (repo, tag string, err error) {
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "git@")
	raw = strings.TrimSuffix(raw, ".git")
	if strings.Contains(raw, "@") {
		parts := strings.SplitN(raw, "@", 2)
		repo = parts[0]
		tag = parts[1]
	} else {
		repo = raw
	}
	repo = strings.Replace(repo, ":", "/", 1)
	if !strings.Contains(repo, "/") {
		return "", "", fmt.Errorf("invalid repository format: %s", raw)
	}
	return repo, tag, nil
}

func updateConfig(pkgPath string, add bool) error {
	data, err := os.ReadFile(".fz.yaml")
	if err != nil {
		if os.IsNotExist(err) {
			data = []byte("source_dirs: []\noutput: myapp\n")
		} else {
			return err
		}
	}
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}
	var dirs []interface{}
	if d, ok := cfg["source_dirs"]; ok {
		if v, ok := d.([]interface{}); ok {
			dirs = v
		}
	}
	found := false
	for i, d := range dirs {
		if d == pkgPath {
			found = true
			if !add {
				dirs = append(dirs[:i], dirs[i+1:]...)
			}
			break
		}
	}
	if add && !found {
		dirs = append(dirs, pkgPath)
	}
	cfg["source_dirs"] = dirs
	newData, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return utils.SecureWriteFile(".fz.yaml", newData)
}

func findPackagePath(name string) (string, error) {
	clean := strings.TrimPrefix(name, "github.com/")
	var found string
	err := filepath.Walk(vendorDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		gitPath := filepath.Join(path, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			rel, _ := filepath.Rel(vendorDir, path)
			if strings.HasSuffix(rel, clean) || rel == clean {
				found = path
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("package %s not found", name)
	}
	return found, nil
}
