package fs

import (
	"io"
	"os"
)

type FileSystem interface {
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(path string, data []byte, perm os.FileMode) error
	ReadFile(path string) ([]byte, error)
	Open(path string) (io.ReadCloser, error)
	CreateTemp(dir, pattern string) (*os.File, error)
	Remove(name string) error
	RemoveAll(path string) error
	Rename(oldpath, newpath string) error
	Stat(name string) (os.FileInfo, error)
	Lstat(name string) (os.FileInfo, error)
	ReadDir(name string) ([]os.DirEntry, error)
	Chmod(name string, mode os.FileMode) error
}

var Default FileSystem = &Unix{}
