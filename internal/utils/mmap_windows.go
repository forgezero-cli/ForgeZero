//go:build windows

package utils

import (
	"syscall"
	"unsafe"
)

var (
	kernel32    = syscall.NewLazyDLL("kernel32.dll")
	mapFile     = kernel32.NewProc("MapViewOfFile")
	unmapAll    = kernel32.NewProc("UnmapViewOfFile")
	mapHandle   = kernel32.NewProc("CreateFileMappingW")
	closeHandle = kernel32.NewProc("CloseHandle")
)

func mmapFile(fd int, size int64) ([]byte, error) {
	hFile := syscall.Handle(uintptr(fd))
	hMapping, _, err := mapHandle.Call(
		uintptr(hFile),
		0,
		uintptr(4),
		uint32(size>>32),
		uint32(size),
	)
	if hMapping == 0 {
		return nil, syscall.Errno(err.(syscall.Errno))
	}
	defer closeHandle.Call(hMapping)

	ptr, _, err := mapFile.Call(hMapping, 0, 0, uint32(size>>32), uint32(size))
	if ptr == 0 {
		return nil, syscall.Errno(err.(syscall.Errno))
	}

	return unsafeByteSlice(unsafe.Pointer(ptr), int(size)), nil
}

func unmapFile(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	ptr := uintptr(unsafe.Pointer(&data[0]))
	_, _, err := unmapAll.Call(ptr)
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

func madviseNormal(data []byte) {
}

func getFileDescriptor(f interface {
	Fd() uintptr
},
) int {
	return int(f.Fd())
}

func unsafeByteSlice(ptr unsafe.Pointer, len int) []byte {
	return unsafe.Slice((*byte)(ptr), len)
}
