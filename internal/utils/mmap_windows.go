//go:build windows

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
	"syscall"
	"unsafe"
)

const (
	pageReadOnly = 0x02
	fileMapRead  = 4
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	mapFile          = kernel32.NewProc("MapViewOfFile")
	unmapAll         = kernel32.NewProc("UnmapViewOfFile")
	mapHandle        = kernel32.NewProc("CreateFileMappingW")
	closeHandle      = kernel32.NewProc("CloseHandle")
	lockFileExProc   = kernel32.NewProc("LockFileEx")
	unlockFileExProc = kernel32.NewProc("UnlockFileEx")
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
	ret, _, err := lockFileExProc.Call(
		uintptr(h),
		0,
		0,
		lockRangeLow,
		lockRangeHigh,
		uintptr(unsafe.Pointer(&ol)),
	)
	if ret == 0 {
		return err.(syscall.Errno)
	}
	return nil
}

func unlockFile(fd int) error {
	h := syscall.Handle(uintptr(fd))
	var ol syscall.Overlapped
	const lockRangeLow = 0xffffffff
	const lockRangeHigh = 0xffffffff
	ret, _, err := unlockFileExProc.Call(
		uintptr(h),
		0,
		lockRangeLow,
		lockRangeHigh,
		uintptr(unsafe.Pointer(&ol)),
	)
	if ret == 0 {
		return err.(syscall.Errno)
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
