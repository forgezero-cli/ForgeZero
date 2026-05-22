//go:build unix
// +build unix

package utils

import (
	"syscall"
	"unsafe"
)

func execRawMap(size int) ([]byte, error) {
	mem, err := syscall.Mmap(-1, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		return nil, err
	}
	return mem, nil
}

func execRawProtect(data []byte) error {
	if len(data) == 0 {
		return syscall.EINVAL
	}
	page := syscall.Getpagesize()
	if uintptr(unsafe.Pointer(&data[0]))%uintptr(page) != 0 {
		return syscall.EINVAL
	}
	_, _, errno := syscall.Syscall(syscall.SYS_MSYNC, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), uintptr(syscall.MS_SYNC))
	if errno != 0 {
		return errno
	}
	return syscall.Mprotect(data, syscall.PROT_READ|syscall.PROT_EXEC)
}

func execRawUnmap(data []byte) error {
	return syscall.Munmap(data)
}
