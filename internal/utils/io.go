/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package utils

import (
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"

	fzvfs "github.com/forgezero-cli/ForgeZero/internal/fs"
	"github.com/forgezero-cli/ForgeZero/internal/seal"
)

func resolveOrAbs(path string) (string, error) {
	if err := ValidateCLIPath(path); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", errors.New("resolve path " + path + ": " + err.Error())
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
				return "", errors.New("strict isolation outside root: " + path)
			}
		}
	}
	eval, err := fileSystem().EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return abs, nil
		}
		return "", errors.New("eval symlinks " + abs + ": " + err.Error())
	}
	if fzvfs.IsStrictIsolation() {
		root := GetExecutionRoot()
		if root != "" {
			rootAbs, rootErr := resolveOrAbs(root)
			if rootErr == nil && !pathWithinRoot(rootAbs, eval) {
				return "", errors.New("strict isolation outside root: " + path)
			}
		}
	}
	return eval, nil
}

func openVerified(resolved string) (io.ReadCloser, error) {
	f, err := fileSystem().OpenVerified(resolved)
	if err != nil {
		if fzvfs.IsStrictIsolation() {
			_, _ = os.Stderr.WriteString("strict isolation file integrity failure\n")
			os.Exit(1)
		}
		return nil, errors.New("open verified " + resolved + ": " + err.Error())
	}
	return f, nil
}

func atomicWrite(resolved string, data []byte) error {
	dir := filepath.Dir(resolved)
	tmp, err := fileSystem().CreateTemp(dir, ".fz_write_*.tmp")
	if err != nil {
		return errors.New("create temp in " + dir + ": " + err.Error())
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = fileSystem().Remove(tmpName)
		}
	}()
	if err := fileSystem().Chmod(tmpName, FilePerm); err != nil {
		return errors.New("chmod temp " + tmpName + ": " + err.Error())
	}
	if _, err := tmp.Write(data); err != nil {
		return errors.New("write temp " + tmpName + ": " + err.Error())
	}
	if err := tmp.Close(); err != nil {
		return errors.New("close temp " + tmpName + ": " + err.Error())
	}
	if err := renameResolved(tmpName, resolved); err != nil {
		return errors.New("rename " + tmpName + " to " + resolved + ": " + err.Error())
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
		return "", errors.New("resolve dest " + path + ": " + err.Error())
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
			return errors.New("remove " + path + ": " + err.Error())
		}
		resolved = abs
	}
	return fileSystem().Remove(resolved)
}

func OpenVerifiedRead(path string) (io.ReadCloser, error) {
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		return nil, errors.New("open " + path + ": " + err.Error())
	}
	return openVerified(resolved)
}

func ReadFileSecure(path string) ([]byte, error) {
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		return nil, errors.New("read " + path + ": " + err.Error())
	}
	f, err := openVerified(resolved)
	if err != nil {
		return nil, errors.New("read " + path + ": " + err.Error())
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, errors.New("read " + path + ": " + err.Error())
	}
	if fzvfs.IsStrictIsolation() {
		h, herr := HashDataDigest(data)
		if herr == nil {
			var hexBuf [64]byte
			hex.Encode(hexBuf[:], h[:])
			if !seal.IsAllowedHex(string(hexBuf[:])) {
				ZeroizeBytes(data)
				os.Exit(1)
			}
		}
	}
	return data, nil
}
