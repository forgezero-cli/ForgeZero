//go:build !windows

package fs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

type Unix struct{}

func (Unix) MkdirAll(path string, perm os.FileMode) error {
	if path == "" || path == "." {
		return nil
	}
	err := syscall.Mkdir(path, uint32(perm))
	if err == nil || errors.Is(err, syscall.EEXIST) {
		return nil
	}
	if !errors.Is(err, syscall.ENOENT) {
		return err
	}
	if err := (Unix{}).MkdirAll(filepath.Dir(path), perm); err != nil {
		return err
	}
	return syscall.Mkdir(path, uint32(perm))
}

func (Unix) WriteFile(path string, data []byte, perm os.FileMode) error {
	fd, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC|syscall.O_CLOEXEC, uint32(perm))
	if err != nil {
		return err
	}
	defer syscall.Close(fd)
	for len(data) > 0 {
		n, err := syscall.Write(fd, data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return syscall.Fchmod(fd, uint32(perm))
}

func (Unix) ReadFile(path string) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(fd)
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		return nil, err
	}
	if st.Size == 0 {
		return []byte{}, nil
	}
	if st.Size < 0 || st.Size > 1<<31 {
		return nil, errors.New("invalid file size")
	}
	size := int(st.Size)
	data, err := syscall.Mmap(fd, 0, size, syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, err
	}
	out := make([]byte, size)
	copy(out, data)
	_ = syscall.Munmap(data)
	return out, nil
}

func (Unix) Open(path string) (io.ReadCloser, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

func (Unix) OpenVerified(path string) (io.ReadCloser, error) {
	var pre syscall.Stat_t
	if err := syscall.Lstat(path, &pre); err != nil {
		return nil, err
	}
	if pre.Mode&syscall.S_IFMT == syscall.S_IFLNK {
		return nil, ErrSymlink
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	var post syscall.Stat_t
	if err := syscall.Fstat(fd, &post); err != nil {
		f.Close()
		return nil, err
	}
	if pre.Dev != post.Dev || pre.Ino != post.Ino || pre.Mode != post.Mode {
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
