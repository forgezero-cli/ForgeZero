package fs

import (
	"errors"
	"io"
	"os"
)

var (
	ErrDiskFull      = errors.New("disk full")
	ErrPermission    = errors.New("permission denied")
	ErrTimeout       = errors.New("i/o timeout")
	ErrInterrupted   = errors.New("interrupted system call")
	ErrSymlink       = errors.New("symlink not permitted")
	ErrPathChanged   = errors.New("path changed during open")
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

var Default FileSystem = Unix{}
