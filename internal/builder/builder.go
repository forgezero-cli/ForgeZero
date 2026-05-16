package builder

import (
	"context"
	"fmt"
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
}

func BuildDir(ctx context.Context, dir, outBin string, debug, verbose bool, mode string, keepObj bool) (*BuildResult, error) {
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
	if err := os.MkdirAll(objDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create object temp dir: %w", err)
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
		objName := uniqueName + ".o"
		objPath := filepath.Join(objDir, objName)
		if err := os.MkdirAll(filepath.Dir(objPath), 0755); err != nil {
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
		if verbose {
			fmt.Printf("Assembling %s -> %s\n", p.src, p.obj)
		}
		if err := assembler.Assemble(ctx, p.src, p.obj, debug, verbose, mode); err != nil {
			return nil, fmt.Errorf("assemble %s: %w", p.src, err)
		}
	}

	if verbose {
		fmt.Printf("Linking %d object files -> %s (mode: %s)\n", len(objFiles), outBin, mode)
	}
	if err := linker.LinkMultiple(ctx, objFiles, outBin, verbose, mode); err != nil {
		return nil, fmt.Errorf("link failed: %w", err)
	}

	return &BuildResult{
		ObjectFiles: objFiles,
		Binary:      outBin,
		ObjDir:      objDir,
	}, nil
}


func CleanDir(dir string, verbose bool) error {
	objDir := filepath.Join(dir, ".fz_objs")
	if _, err := os.Stat(objDir); err == nil {
		if verbose {
			fmt.Printf("Removing %s\n", objDir)
		}
		if err := os.RemoveAll(objDir); err != nil {
			return fmt.Errorf("failed to remove %s: %w", objDir, err)
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
		if info.Mode()&0111 != 0 {
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
