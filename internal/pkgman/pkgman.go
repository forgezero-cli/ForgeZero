package pkgman

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fz/internal/utils"

	"gopkg.in/yaml.v3"
)

var httpClient = &http.Client{}

var runGit = func(ctx context.Context, args ...string) (string, error) {
	return utils.RunCommand(ctx, false, os.Stdout, os.Stderr, "git", args...)
}

const vendorDir = "vendor"

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
	destRel := filepath.Join(vendorDir, repo)
	dest, err := secureVendorPath(repo)
	if err != nil {
		return err
	}
	if err := utils.SecureMkdirAll(dest); err != nil {
		return errors.New("prepare vendor dir: " + err.Error())
	}
	cloneURL := "https://" + repo
	if _, err := runGit(ctx, "clone", cloneURL, dest); err != nil {
		return errors.New("git clone " + repo + ": " + err.Error())
	}
	if tag != "" {
		if _, err := runGit(ctx, "-C", dest, "checkout", tag); err != nil {
			return errors.New("git checkout " + repo + "@" + tag + ": " + err.Error())
		}
	}
	if err := updateConfig(destRel, true); err != nil {
		return err
	}
	os.Stdout.WriteString("Package " + pkgURL + " installed.\n")
	return nil
}

func Remove(ctx context.Context, pkgURL string) error {
	_ = ctx
	repo, _, err := parsePkgURL(pkgURL)
	if err == nil {
		dest, derr := secureVendorPath(repo)
		if derr == nil {
			if _, err := os.Stat(dest); err == nil {
				return removePackage(dest)
			}
		}
	}
	dest, err := findPackagePath(pkgURL)
	if err != nil {
		return err
	}
	return removePackage(dest)
}

func removePackage(path string) error {
	if err := ensureVendorSubpath(path); err != nil {
		return err
	}
	if err := os.RemoveAll(path); err != nil {
		return errors.New("failed to remove " + path + ": " + err.Error())
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
	os.Stdout.WriteString("Package " + path + " removed.\n")
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
	absPkgPath, _ := filepath.Abs(pkgPath)
	newDirs := []interface{}{}
	for _, d := range dirsSlice {
		if str, ok := d.(string); ok {
			remove := false
			if absDir, err := filepath.Abs(str); err == nil {
				if absDir == absPkgPath || strings.HasPrefix(absDir, absPkgPath+string(filepath.Separator)) {
					remove = true
				}
			}
			if !remove {
				if str == pkgPath || strings.HasPrefix(str, pkgPath+string(filepath.Separator)) {
					remove = true
				}
			}
			if !remove {
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
		os.Stdout.WriteString("No packages installed.\n")
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
		os.Stdout.WriteString("No packages installed.\n")
		return nil
	}
	for _, pkg := range packages {
		os.Stdout.WriteString(pkg + "\n")
	}
	return nil
}

func Update(ctx context.Context) error {
	entries, err := os.ReadDir(vendorDir)
	if err != nil {
		if os.IsNotExist(err) {
			os.Stdout.WriteString("No packages to update.\n")
			return nil
		}
		return err
	}
	for _, entry := range entries {
		pkgPath := filepath.Join(vendorDir, entry.Name())
		if _, err := runGit(ctx, "-C", pkgPath, "pull"); err != nil {
			os.Stderr.WriteString("Warning: failed to update " + entry.Name() + ": " + err.Error() + "\n")
		} else {
			os.Stdout.WriteString("Updated " + entry.Name() + "\n")
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
	urls = append(urls, "https://github.com/forgezero-cli/pkgman/raw/branch/main/catalog.json")
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
		return nil, errors.New("HTTP " + strconv.Itoa(resp.StatusCode))
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
		os.Stderr.WriteString("Warning: failed to fetch catalog from " + url + ": " + err.Error() + "\n")
	}
	return nil, errors.New("all catalog URLs failed: " + lastErr.Error())
}

func ListCatalog(ctx context.Context) error {
	cat, err := fetchCatalog()
	if err != nil {
		return err
	}
	os.Stdout.WriteString("Available packages (catalog version " + strconv.Itoa(cat.Version) + "):\n")
	for _, p := range cat.Packages {
		os.Stdout.WriteString("  " + p.Name + " (" + p.Category + ") - " + p.Description + "\n")
	}
	return nil
}

func SearchCatalog(ctx context.Context, keyword string) error {
	cat, err := fetchCatalog()
	if err != nil {
		return err
	}
	os.Stdout.WriteString("Search results for '" + keyword + "':\n")
	found := false
	kw := strings.ToLower(keyword)
	for _, p := range cat.Packages {
		if strings.Contains(strings.ToLower(p.Name), kw) ||
			strings.Contains(strings.ToLower(p.Description), kw) ||
			strings.Contains(strings.ToLower(p.Category), kw) {
			os.Stdout.WriteString("  " + p.Name + " (" + p.Category + ") - " + p.Description + "\n")
			found = true
		}
	}
	if !found {
		os.Stdout.WriteString("No matching packages found.\n")
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
		return errors.New("package " + pkgName + " not found in catalog")
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
			os.Stderr.WriteString("Warning: failed to compute hash for " + pkgName + ": " + err.Error() + "\n")
		} else if actualHash != pkg.Hash {
			_ = Remove(ctx, pkgName)
			return errors.New("hash mismatch for package " + pkgName + " (expected " + pkg.Hash + ", got " + actualHash + ")")
		} else {
			os.Stdout.WriteString("Hash verification passed for " + pkgName + "\n")
		}
	}
	if pkg.SourceDir != "" {
		rootPath := filepath.Join(vendorDir, pkg.Repo)
		subPath := filepath.Join(rootPath, pkg.SourceDir)
		if err := updateConfig(subPath, true); err != nil {
			return err
		}
	}
	os.Stdout.WriteString("Installed catalog package " + pkgName + "\n")
	return nil
}

func secureVendorPath(repo string) (string, error) {
	if strings.Contains(repo, "..") || strings.Contains(repo, "\\") {
		return "", errors.New("invalid repository path: " + repo)
	}
	dest := filepath.Join(vendorDir, repo)
	absVendor, err := filepath.Abs(vendorDir)
	if err != nil {
		return "", err
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absVendor, absDest)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("package path escapes vendor directory: " + repo)
	}
	return absDest, nil
}

func ensureVendorSubpath(resolved string) error {
	absVendor, err := filepath.Abs(vendorDir)
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absVendor, absPath)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("path outside vendor directory: " + resolved)
	}
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
	if repo == "." || repo == "/" || strings.HasPrefix(repo, "/") || strings.Contains(repo, "..") || strings.Contains(repo, "\\") {
		return "", "", errors.New("invalid repository format: " + raw)
	}
	repo = path.Clean(repo)
	if repo == "." || repo == "/" || strings.HasPrefix(repo, "/") || strings.Contains(repo, "..") || strings.Contains(repo, "\\") {
		return "", "", errors.New("invalid repository format: " + raw)
	}
	if strings.Contains(repo, "//") {
		return "", "", errors.New("invalid repository format: " + raw)
	}
	parts := strings.Split(repo, "/")
	if len(parts) < 2 {
		return "", "", errors.New("invalid repository format: " + raw)
	}
	for _, p := range parts {
		if p == "" {
			return "", "", errors.New("invalid repository format: " + raw)
		}
		if err := utils.ValidateCLIArg(p); err != nil {
			return "", "", errors.New("invalid repository segment: " + err.Error())
		}
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
	if strings.Contains(name, "..") || strings.Contains(name, "\\") {
		return "", errors.New("invalid package name: " + name)
	}
	clean := strings.TrimPrefix(name, "github.com/")
	var found string
	err := filepath.Walk(vendorDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		gitPath := filepath.Join(path, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			rel, _ := filepath.Rel(vendorDir, path)
			if strings.HasSuffix(rel, clean) || rel == clean {
				resolved, rerr := filepath.Abs(path)
				if rerr != nil {
					return rerr
				}
				if suberr := ensureVendorSubpath(resolved); suberr != nil {
					return suberr
				}
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
		return "", errors.New("package " + name + " not found")
	}
	return found, nil
}