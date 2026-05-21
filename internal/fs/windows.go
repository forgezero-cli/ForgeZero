//go:build windows

package fs

import (
	"io"
	"os"
	"path/filepath"
)

type Windows struct{}

func (Windows) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(CleanPath(path), perm)
}

func (Windows) WriteFile(path string, data []byte, perm os.FileMode) error {
	p := CleanPath(path)
	if err := os.WriteFile(p, data, perm); err != nil {
		return err
	}
	_ = os.Chmod(p, perm)
	return nil
}

func (Windows) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(CleanPath(path))
}

func (Windows) Open(path string) (io.ReadCloser, error) {
	return os.Open(CleanPath(path))
}

func (Windows) OpenVerified(path string) (io.ReadCloser, error) {
	p := CleanPath(path)
	pre, err := os.Lstat(p)
	if err != nil {
		return nil, err
	}
	if isSymlinkMode(pre.Mode()) {
		return nil, ErrSymlink
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	post, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !os.SameFile(pre, post) {
		f.Close()
		return nil, ErrPathChanged
	}
	return f, nil
}

func (Windows) CreateTemp(dir, pattern string) (*os.File, error) {
	return os.CreateTemp(CleanPath(dir), pattern)
}

func (Windows) Remove(name string) error {
	return os.Remove(CleanPath(name))
}

func (Windows) RemoveAll(path string) error {
	return os.RemoveAll(CleanPath(path))
}

func (Windows) Rename(oldpath, newpath string) error {
	return renameAtomic(CleanPath(oldpath), CleanPath(newpath))
}

func (Windows) Stat(name string) (os.FileInfo, error) {
	return os.Stat(CleanPath(name))
}

func (Windows) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(CleanPath(name))
}

func (Windows) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(CleanPath(name))
}

func (Windows) Chmod(name string, mode os.FileMode) error {
	_ = os.Chmod(CleanPath(name), mode)
	return nil
}

func (Windows) Readlink(name string) (string, error) {
	return os.Readlink(CleanPath(name))
}

func (Windows) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(CleanPath(path))
}

func (Windows) SameFile(a, b os.FileInfo) bool {
	return os.SameFile(a, b)
}

func isSymlinkMode(mode os.FileMode) bool {
	return mode&os.ModeSymlink != 0
}
