package fs

import (
	"errors"
	"io"
	"os"
	"sync"
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

var readBufferPool = sync.Pool{New: func() any { b := make([]byte, 32*1024); return &b }}

func readFileBytes(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bufp := readBufferPool.Get().(*[]byte)
	buf := *bufp
	defer readBufferPool.Put(bufp)
	var result []byte
	for {
		n, err := f.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	return result, nil
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
