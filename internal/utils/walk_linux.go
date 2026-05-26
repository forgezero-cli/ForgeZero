//go:build linux

package utils

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func Walk(root string, fn func(path string, info os.FileInfo, err error) error) error {
	root = filepath.Clean(root)
	fi, err := os.Lstat(root)
	if err != nil {
		return fn(root, nil, err)
	}
	if err := fn(root, fi, nil); err != nil {
		return err
	}
	stack := []string{root}
	var buf [8192]byte
	for len(stack) > 0 {
		dir := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		fd, err := syscall.Open(dir, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
		if err != nil {
			return err
		}
		for {
			n, _, errno := syscall.Syscall(syscall.SYS_GETDENTS64, uintptr(fd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
			if errno != 0 {
				syscall.Close(fd)
				if errno == syscall.EINTR {
					continue
				}
				return os.NewSyscallError("getdents64", errno)
			}
			if n == 0 {
				break
			}
			_, _, names := syscall.ParseDirent(buf[:n], -1, nil)
			for _, name := range names {
				if name == "." || name == ".." {
					continue
				}
				path := filepath.Join(dir, name)
				info, err := os.Lstat(path)
				if err != nil {
					if err := fn(path, nil, err); err != nil {
						if err == filepath.SkipDir {
							continue
						}
						syscall.Close(fd)
						return err
					}
					continue
				}

				if err := fn(path, info, nil); err != nil {
					if err == filepath.SkipDir {
						continue
					}
					syscall.Close(fd)
					return err
				}

				if info.IsDir() {
					stack = append(stack, path)
				}

			}
		}
		syscall.Close(fd)
	}
	return nil
}
