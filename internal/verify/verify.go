package verify

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"fz/internal/utils"
)

type ManifestEntry struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

type Manifest struct {
	Root      string          `json:"root"`
	CreatedAt string          `json:"created_at"`
	Entries   []ManifestEntry `json:"entries"`
}

type VerifyResult struct {
	Missing  []string `json:"missing"`
	Modified []string `json:"modified"`
	Extra    []string `json:"extra"`
}

func LoadManifest(path string) (*Manifest, error) {
	if err := utils.ValidateCLIPath(path); err != nil {
		return nil, err
	}
	data, err := utils.ReadFileSecure(path)
	if err != nil {
		return nil, fmt.Errorf("load manifest %s: %w", path, err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, err)
	}
	return &manifest, nil
}

func WriteManifest(path, root string) error {
	if err := utils.ValidateCLIPath(path); err != nil {
		return err
	}
	if err := utils.ValidateCLIPath(root); err != nil {
		return err
	}
	entries, err := BuildManifest(root)
	if err != nil {
		return err
	}
	manifest := Manifest{
		Root:      filepath.Clean(root),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Entries:   entries,
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return utils.SecureWriteFile(path, data)
}

func VerifyRoot(root, manifestPath string) (*VerifyResult, error) {
	if err := utils.ValidateCLIPath(root); err != nil {
		return nil, err
	}
	if err := utils.ValidateCLIPath(manifestPath); err != nil {
		return nil, err
	}
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	entries, err := BuildManifest(root)
	if err != nil {
		return nil, err
	}
	current := map[string]string{}
	for _, entry := range entries {
		current[entry.Path] = entry.Hash
	}
	var missing, modified, extra []string
	for _, entry := range manifest.Entries {
		currentHash, ok := current[entry.Path]
		if !ok {
			missing = append(missing, entry.Path)
			continue
		}
		if !utils.ConstantTimeEqual(currentHash, entry.Hash) {
			modified = append(modified, entry.Path)
		}
	}
	recorded := make(map[string]struct{}, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		recorded[entry.Path] = struct{}{}
	}
	for path := range current {
		if _, ok := recorded[path]; !ok {
			extra = append(extra, path)
		}
	}
	sort.Strings(missing)
	sort.Strings(modified)
	sort.Strings(extra)
	return &VerifyResult{Missing: missing, Modified: modified, Extra: extra}, nil
}

func BuildManifest(root string) ([]ManifestEntry, error) {
	root = filepath.Clean(root)
	if err := utils.ValidateCLIPath(root); err != nil {
		return nil, err
	}
	files, err := collectFiles(root)
	if err != nil {
		return nil, err
	}
	entries := make([]ManifestEntry, len(files))
	errCh := make(chan error, len(files))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	for i, rel := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(index int, fileRel string) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					select {
					case errCh <- fmt.Errorf("panic hashing %s: %v", fileRel, r):
					default:
					}
				}
			}()
			fullPath := filepath.Join(root, fileRel)
			hash, err := utils.HashFile(fullPath)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("hash %s: %w", fullPath, err):
				default:
				}
				return
			}
			entries[index] = ManifestEntry{Path: fileRel, Hash: hash}
		}(i, rel)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries, nil
}

func collectFiles(root string) ([]string, error) {
	root = filepath.Clean(root)
	var files []string
	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipDir(root, path) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks not permitted: %s", path)
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if filepath.Base(rel) == "manifest.json" {
			return nil
		}
		ignoredFiles := map[string]bool{
			"blake3.manifest":                          true,
			"internal/contribute/CONTRIBUTING_USER.md": true,
		}
		if ignoredFiles[rel] {
			return nil
		}
		if rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("invalid path outside root: %s", path)
		}
		files = append(files, rel)
		return nil
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func shouldSkipDir(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return false
	}
	first := strings.Split(rel, string(os.PathSeparator))[0]
	switch first {
	case ".git", ".fz_objs", ".fz_cache", "vendor", "release", "node_modules":
		return true
	}
	return false
}
