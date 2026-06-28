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

package fs

import (
	"errors"
	"io"
	"os"
	"sync/atomic"
)

var isolationMode atomic.Value

func init() {
	isolationMode.Store("none")
}

func SetIsolationMode(mode string) {
	switch mode {
	case "none", "standard", "strict":
		isolationMode.Store(mode)
	default:
		isolationMode.Store("none")
	}
}

func GetIsolationMode() string {
	v := isolationMode.Load()
	if v == nil {
		return "none"
	}
	return v.(string)
}

func IsStrictIsolation() bool {
	return GetIsolationMode() == "strict"
}

var (
	ErrDiskFull    = errors.New("disk full")
	ErrPermission  = errors.New("permission denied")
	ErrTimeout     = errors.New("i/o timeout")
	ErrInterrupted = errors.New("interrupted system call")
	ErrSymlink     = errors.New("symlink not permitted")
	ErrPathChanged = errors.New("path changed during open")
)

type FileSystem interface {
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(path string, data []byte, perm os.FileMode) error
	ReadFile(path string) ([]byte, error)
	Open(path string) (io.ReadCloser, error)
	OpenVerified(path string) (io.ReadCloser, error)
	CreateTemp(dir, pattern string) (*os.File, error)
	Remove(name string) error
	RemoveAll(path string) error
	Rename(oldpath, newpath string) error
	Stat(name string) (os.FileInfo, error)
	Lstat(name string) (os.FileInfo, error)
	ReadDir(name string) ([]os.DirEntry, error)
	Chmod(name string, mode os.FileMode) error
	Readlink(name string) (string, error)
	EvalSymlinks(path string) (string, error)
	SameFile(a, b os.FileInfo) bool
}
