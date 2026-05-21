//go:build windows
// +build windows

package utils

import (
	"syscall"
	"unsafe"
)

func execRawMap(size int) ([]byte, error) {
	addr, err := syscall.VirtualAlloc(0, uintptr(size), syscall.MEM_COMMIT|syscall.MEM_RESERVE, syscall.PAGE_READWRITE)
	if err != nil {
		return nil, err
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(addr)), size), nil
}

func execRawProtect(data []byte) error {
	old := uint32(0)
	return syscall.VirtualProtect(uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), syscall.PAGE_EXECUTE_READ, &old)
}

func execRawUnmap(data []byte) error {
	return syscall.VirtualFree(uintptr(unsafe.Pointer(&data[0])), 0, syscall.MEM_RELEASE)
}
