package fs

import (
	"io"
	"os"
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
	return os.Rename(oldpath, newpath)
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
