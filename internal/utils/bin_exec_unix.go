//go:build unix
// +build unix

package utils

import (
	"syscall"
)

func execRawMap(size int) ([]byte, error) {
	mem, err := syscall.Mmap(-1, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		return nil, err
	}
	return mem, nil
}

func execRawProtect(data []byte) error {
	return syscall.Mprotect(data, syscall.PROT_READ|syscall.PROT_EXEC)
}

func execRawUnmap(data []byte) error {
	return syscall.Munmap(data)
}
