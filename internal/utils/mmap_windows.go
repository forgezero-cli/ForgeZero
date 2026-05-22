//go:build windows

package utils

import (
	"syscall"
	"unsafe"
)

const (
	pageReadOnly = 0x02
	fileMapRead  = 4
)

var (
	kernel32    = syscall.NewLazyDLL("kernel32.dll")
	mapFile     = kernel32.NewProc("MapViewOfFile")
	unmapAll    = kernel32.NewProc("UnmapViewOfFile")
	mapHandle   = kernel32.NewProc("CreateFileMappingW")
	closeHandle = kernel32.NewProc("CloseHandle")
)

func mmapFile(fd int, size int64) ([]byte, error) {
	if size <= 0 {
		return nil, nil
	}
	hFile := syscall.Handle(uintptr(fd))
	high := uintptr(uint32(uint64(size) >> 32))
	low := uintptr(uint32(uint64(size)))
	hMapping, _, err := mapHandle.Call(
		uintptr(hFile),
		0,
		uintptr(pageReadOnly),
		high,
		low,
		0,
	)
	if hMapping == 0 {
		return nil, syscall.Errno(err.(syscall.Errno))
	}
	defer closeHandle.Call(hMapping)

	ptr, _, err := mapFile.Call(
		uintptr(hMapping),
		uintptr(fileMapRead),
		0,
		0,
		uintptr(size),
	)
	if ptr == 0 {
		return nil, syscall.Errno(err.(syscall.Errno))
	}

	return unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(size)), nil
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

func lockFileShared(fd int) error {
	h := syscall.Handle(uintptr(fd))
	var ol syscall.Overlapped
	const lockRangeLow = 0xffffffff
	const lockRangeHigh = 0xffffffff
	return syscall.LockFileEx(h, 0, 0, lockRangeLow, lockRangeHigh, &ol)
}

func unlockFile(fd int) error {
	h := syscall.Handle(uintptr(fd))
	var ol syscall.Overlapped
	const lockRangeLow = 0xffffffff
	const lockRangeHigh = 0xffffffff
	return syscall.UnlockFileEx(h, 0, lockRangeLow, lockRangeHigh, &ol)
}

func madviseNormal(data []byte) {
}

func getFileDescriptor(f interface {
	Fd() uintptr
},
) int {
	return int(f.Fd())
}
