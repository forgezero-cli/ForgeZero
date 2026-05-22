package utils

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	fzvfs "fz/internal/fs"
	"fz/internal/seal"
)

func resolveOrAbs(path string) (string, error) {
	if err := ValidateCLIPath(path); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path %s: %w", path, err)
	}
	return filepath.Clean(abs), nil
}

func ResolveSecurePath(path string) (string, error) {
	abs, err := resolveOrAbs(path)
	if err != nil {
		return "", err
	}
	if fzvfs.IsStrictIsolation() {
		root := GetExecutionRoot()
		if root != "" {
			rootAbs, rootErr := resolveOrAbs(root)
			if rootErr == nil && !pathWithinRoot(rootAbs, abs) {
				return "", fmt.Errorf("strict isolation outside root: %s", path)
			}
		}
	}
	eval, err := fileSystem().EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return abs, nil
		}
		return "", fmt.Errorf("eval symlinks %s: %w", abs, err)
	}
	if fzvfs.IsStrictIsolation() {
		root := GetExecutionRoot()
		if root != "" {
			rootAbs, rootErr := resolveOrAbs(root)
			if rootErr == nil && !pathWithinRoot(rootAbs, eval) {
				return "", fmt.Errorf("strict isolation outside root: %s", path)
			}
		}
	}
	return eval, nil
}

func openVerified(resolved string) (io.ReadCloser, error) {
	f, err := fileSystem().OpenVerified(resolved)
	if err != nil {
		if fzvfs.IsStrictIsolation() {
			os.Stderr.WriteString("strict isolation file integrity failure\n")
			os.Exit(1)
		}
		return nil, fmt.Errorf("open verified %s: %w", resolved, err)
	}
	return f, nil
}

func atomicWrite(resolved string, data []byte) error {
	dir := filepath.Dir(resolved)
	tmp, err := fileSystem().CreateTemp(dir, ".fz_write_*.tmp")
	if err != nil {
		return fmt.Errorf("create temp in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		tmp.Close()
		if cleanup {
			_ = fileSystem().Remove(tmpName)
		}
	}()
	if err := fileSystem().Chmod(tmpName, FilePerm); err != nil {
		return fmt.Errorf("chmod temp %s: %w", tmpName, err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temp %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp %s: %w", tmpName, err)
	}
	if err := renameResolved(tmpName, resolved); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpName, resolved, err)
	}
	cleanup = false
	return fileSystem().Chmod(resolved, FilePerm)
}

func renameResolved(oldpath, newpath string) error {
	return fileSystem().Rename(oldpath, newpath)
}

func resolveDest(path string) (string, error) {
	resolved, err := ResolveSecurePath(path)
	if err == nil {
		return resolved, nil
	}
	abs, absErr := resolveOrAbs(path)
	if absErr != nil {
		return "", fmt.Errorf("resolve dest %s: %w", path, err)
	}
	return abs, nil
}

func ConstantTimeEqual(a, b string) bool {
	return constantTimeEqual(a, b)
}

func StatResolved(resolved string) (os.FileInfo, error) {
	return fileSystem().Stat(resolved)
}

func LstatPath(path string) (os.FileInfo, error) {
	abs, err := resolveOrAbs(path)
	if err != nil {
		return nil, err
	}
	return fileSystem().Lstat(abs)
}

func ReadDirResolved(resolved string) ([]os.DirEntry, error) {
	return fileSystem().ReadDir(resolved)
}

func EvalSymlinksPath(path string) (string, error) {
	abs, err := resolveOrAbs(path)
	if err != nil {
		return "", err
	}
	return fileSystem().EvalSymlinks(abs)
}

func RemovePath(path string) error {
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		abs, absErr := resolveOrAbs(path)
		if absErr != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
		resolved = abs
	}
	return fileSystem().Remove(resolved)
}

func OpenVerifiedRead(path string) (io.ReadCloser, error) {
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return openVerified(resolved)
}

func ReadFileSecure(path string) ([]byte, error) {
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	f, err := openVerified(resolved)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if fzvfs.IsStrictIsolation() {
		h, herr := HashDataDigest(data)
		if herr == nil {
			hb := make([]byte, hex.EncodedLen(len(h)))
			hex.Encode(hb, h[:])
			if !seal.IsAllowedHex(string(hb)) {
				for i := range data {
					data[i] = 0
				}
				os.Exit(1)
			}
		}
	}
	return data, nil
}
