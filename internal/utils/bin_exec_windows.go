//go:build windows
// +build windows

package utils

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func execRawMap(size int) ([]byte, error) {
	addr, err := windows.VirtualAlloc(0, uintptr(size), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		return nil, err
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(addr)), size), nil
}

func execRawProtect(data []byte) error {
	old := uint32(0)
	return windows.VirtualProtect(uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), windows.PAGE_EXECUTE_READ, &old)
}

func execRawUnmap(data []byte) error {
	return windows.VirtualFree(uintptr(unsafe.Pointer(&data[0])), 0, windows.MEM_RELEASE)
}
