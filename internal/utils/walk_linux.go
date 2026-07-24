//go:build linux

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
				_ = syscall.Close(fd)
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
						_ = syscall.Close(fd)
						return err
					}
					continue
				}

				if err := fn(path, info, nil); err != nil {
					if err == filepath.SkipDir {
						continue
					}
					_ = syscall.Close(fd)
					return err
				}

				if info.IsDir() {
					stack = append(stack, path)
				}

			}
		}
		_ = syscall.Close(fd)
	}
	return nil
}
