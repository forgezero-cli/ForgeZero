//go:build !windows

package fs

import (
	"io"
	"os"
	"path/filepath"
)

type Unix struct{}

func (Unix) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (Unix) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (Unix) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (Unix) Open(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func (Unix) OpenVerified(path string) (io.ReadCloser, error) {
	pre, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if pre.Mode()&os.ModeSymlink != 0 {
		return nil, ErrSymlink
	}
	f, err := os.Open(path)
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

func (Unix) CreateTemp(dir, pattern string) (*os.File, error) {
	return os.CreateTemp(dir, pattern)
}

func (Unix) Remove(name string) error {
	return os.Remove(name)
}

func (Unix) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (Unix) Rename(oldpath, newpath string) error {
	return renameAtomic(oldpath, newpath)
}

func (Unix) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (Unix) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
}

func (Unix) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}

func (Unix) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

func (Unix) Readlink(name string) (string, error) {
	return os.Readlink(name)
}

func (Unix) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

func (Unix) SameFile(a, b os.FileInfo) bool {
	return os.SameFile(a, b)
}
