package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func symlinkAllowed(rootEval, path, target string) (bool, error) {
	linkTarget, err := fileSystem().Readlink(path)
	if err != nil {
		return false, fmt.Errorf("cannot read symlink %s: %w", path, err)
	}
	var targetAbs string
	if !filepath.IsAbs(linkTarget) {
		targetAbs = filepath.Clean(filepath.Join(filepath.Dir(path), linkTarget))
	} else {
		targetAbs = filepath.Clean(linkTarget)
	}
	targetEval, err := fileSystem().EvalSymlinks(targetAbs)
	if err != nil {
		return false, fmt.Errorf("cannot resolve symlink %s target %s: %w", path, targetAbs, err)
	}
	rootClean := filepath.Clean(rootEval)
	if targetEval == rootClean || strings.HasPrefix(targetEval, rootClean+string(os.PathSeparator)) {
		return true, nil
	}
	fmt.Fprintf(os.Stderr, "SECURITY WARNING: skipping symlink %s -> %s outside project root %s\n", path, targetAbs, rootClean)
	return false, nil
}
