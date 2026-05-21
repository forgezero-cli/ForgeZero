package fs

import (
	"path/filepath"
	"strings"
)

func CleanPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, `\\`) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.FromSlash(path))
}

func IsUNC(path string) bool {
	return len(path) >= 2 && (path[0] == '\\' && path[1] == '\\')
}

func HasDrivePrefix(path string) bool {
	if len(path) < 2 {
		return false
	}
	return path[1] == ':' && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z'))
}

func NormalizeAbs(path string) (string, error) {
	clean := CleanPath(path)
	if IsUNC(clean) {
		return clean, nil
	}
	return filepath.Abs(clean)
}
