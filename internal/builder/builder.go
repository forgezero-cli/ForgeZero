package builder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fz/internal/assembler"
	"fz/internal/linker"
	"fz/internal/utils"
)

type BuildResult struct {
	ObjectFiles []string
	Binary      string
	ObjDir      string
	CacheDir    string
}

func BuildDir(ctx context.Context, dir, outBin string, debug, verbose bool, mode string, keepObj bool, noCache bool, noSymbolCheck bool, sanitize bool, strict bool) (*BuildResult, error) {
	if outBin == "" {
		base := filepath.Base(dir)
		if utils.IsWindows() {
			outBin = base + ".exe"
		} else {
			outBin = base + ".out"
		}
	}
	if info, err := os.Stat(outBin); err == nil && info.IsDir() {
		return nil, fmt.Errorf("output path %s is a directory, cannot write binary", outBin)
	}
	if err := utils.EnsureDir(outBin); err != nil {
		return nil, fmt.Errorf("cannot create output directory: %w", err)
	}

	var srcFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if utils.SupportedExtension(ext) {
			srcFiles = append(srcFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}
	if len(srcFiles) == 0 {
		return nil, fmt.Errorf("no supported assembly files found in %s", dir)
	}

	objDir := filepath.Join(filepath.Dir(outBin), ".fz_objs")
	cacheDir := filepath.Join(filepath.Dir(outBin), ".fz_cache")
	if err := os.MkdirAll(objDir, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create object temp dir: %w", err)
	}
	if !noCache {
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return nil, fmt.Errorf("cannot create cache dir: %w", err)
		}
	}
	if !keepObj {
		defer func() {
			os.RemoveAll(objDir)
		}()
	}

	type pair struct {
		src string
		obj string
	}
	pairs := make([]pair, len(srcFiles))
	objFilesSet := make(map[string]bool)

	for i, src := range srcFiles {
		rel, err := filepath.Rel(dir, src)
		if err != nil {
			rel = filepath.Base(src)
		}
		ext := filepath.Ext(rel)
		baseNoExt := strings.TrimSuffix(rel, ext)
		uniqueName := strings.ReplaceAll(baseNoExt, string(filepath.Separator), "_")
		srcExt := strings.TrimPrefix(ext, ".")
		objName := uniqueName + "_" + srcExt + ".o"
		objPath := filepath.Join(objDir, objName)
		if err := os.MkdirAll(filepath.Dir(objPath), 0o755); err != nil {
			return nil, fmt.Errorf("cannot create subdir for object: %w", err)
		}
		pairs[i] = pair{src: src, obj: objPath}
		objFilesSet[objPath] = true
	}

	var objFiles []string
	for obj := range objFilesSet {
		objFiles = append(objFiles, obj)
	}

	for _, p := range pairs {
		needAssemble := true
		if !noCache {
			cachedObj, err := checkCache(p.src, cacheDir, debug, verbose, mode)
			if err == nil && cachedObj != "" {
				if verbose {
					fmt.Printf("Cache hit for %s\n", p.src)
				}
				if err := copyFile(cachedObj, p.obj); err == nil {
					needAssemble = false
				}
			}
		}
		if needAssemble {
			if verbose {
				fmt.Printf("Assembling %s -> %s\n", p.src, p.obj)
			}
			if err := assembler.Assemble(ctx, p.src, p.obj, debug, verbose, mode); err != nil {
				return nil, fmt.Errorf("assemble %s: %w", p.src, err)
			}
			if !noCache {
				if err := storeCache(p.src, p.obj, cacheDir, debug, verbose, mode); err != nil && verbose {
					fmt.Printf("Warning: cache store failed: %v\n", err)
				}
			}
		}
	}

	if verbose {
		fmt.Printf("Linking %d object files -> %s (mode: %s)\n", len(objFiles), outBin, mode)
	}
	if err := linker.LinkMultiple(ctx, objFiles, outBin, verbose, mode, noSymbolCheck, sanitize, strict); err != nil {
		return nil, fmt.Errorf("link failed: %w", err)
	}

	return &BuildResult{
		ObjectFiles: objFiles,
		Binary:      outBin,
		ObjDir:      objDir,
		CacheDir:    cacheDir,
	}, nil
}

func checkCache(src, cacheDir string, debug, verbose bool, mode string) (string, error) {
	h, err := hashFile(src)
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("%s_%v_%s", h, debug, mode)
	cacheObj := filepath.Join(cacheDir, key+".o")
	if _, err := os.Stat(cacheObj); err == nil {
		return cacheObj, nil
	}
	return "", os.ErrNotExist
}

func storeCache(src, obj, cacheDir string, debug, verbose bool, mode string) error {
	h, err := hashFile(src)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s_%v_%s", h, debug, mode)
	cacheObj := filepath.Join(cacheDir, key+".o")
	return copyFile(obj, cacheObj)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func CleanDir(dir string, verbose bool) error {
	objDir := filepath.Join(dir, ".fz_objs")
	cacheDir := filepath.Join(dir, ".fz_cache")
	for _, d := range []string{objDir, cacheDir} {
		if _, err := os.Stat(d); err == nil {
			if verbose {
				fmt.Printf("Removing %s\n", d)
			}
			if err := os.RemoveAll(d); err != nil {
				return fmt.Errorf("failed to remove %s: %w", d, err)
			}
		}
	}
	base := filepath.Base(dir)
	patterns := []string{base + ".out", base + ".exe"}
	for _, p := range patterns {
		path := filepath.Join(dir, p)
		if _, err := os.Stat(path); err == nil {
			if verbose {
				fmt.Printf("Removing %s\n", path)
			}
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove %s: %w", path, err)
			}
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("cannot read directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join(dir, name)
		if strings.HasSuffix(name, ".o") {
			if verbose {
				fmt.Printf("Removing object file %s\n", path)
			}
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove %s: %w", path, err)
			}
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Mode()&0o111 != 0 {
			ext := strings.ToLower(filepath.Ext(name))
			if !utils.SupportedExtension(ext) && ext != "" {
				if verbose {
					fmt.Printf("Removing executable %s\n", path)
				}
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove %s: %w", path, err)
				}
			} else if ext == "" {
				if verbose {
					fmt.Printf("Removing executable (no extension) %s\n", path)
				}
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove %s: %w", path, err)
				}
			}
		}
	}
	return nil
}
