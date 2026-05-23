//go:build linux
// +build linux

package utils

import (
	"syscall"
)

func mmapFile(fd int, size int64) ([]byte, error) {
	sizeInt := int(size)
	data, err := syscall.Mmap(fd, 0, sizeInt, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func unmapFile(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return syscall.Munmap(data)
}

func madviseNormal(data []byte) {
	if len(data) > 0 {
		_ = syscall.Madvise(data, syscall.MADV_NORMAL)
	}
}

func getFileDescriptor(f interface {
	Fd() uintptr
}) int {
	return int(f.Fd())
}

func lockFileShared(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_SH)
}

func unlockFile(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_UN)
}
