package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	DirPerm  os.FileMode = 0o700
	FilePerm os.FileMode = 0o600
)

func forbiddenPathChars() string {
	if runtime.GOOS == "windows" {
		return "`$&|;><*?[]{}()\"'\x00\n\r"
	}
	return "`$&|;><*?[]{}()\"'\\\x00\n\r"
}

func forbiddenArgChars() string {
	if runtime.GOOS == "windows" {
		return "`$&|;><*?[]{}()\"'\x00\n\r"
	}
	return "`$&|;><*?[]{}()\"'\\\x00\n\r"
}

func pathWithinRoot(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if filepath.VolumeName(root) != "" && filepath.VolumeName(root) != filepath.VolumeName(target) {
		return false
	}
	if strings.EqualFold(root, target) {
		return true
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func isUnsafeUNC(path string) bool {
	if !strings.HasPrefix(path, `\\`) {
		return false
	}
	rest := strings.TrimPrefix(path, `\\`)
	parts := strings.SplitN(rest, `\`, 3)
	return len(parts) < 2 || parts[0] == "" || parts[1] == ""
}
