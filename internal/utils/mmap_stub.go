//go:build !unix && !windows

package utils

import (
	"unsafe"
)

func mmapFile(fd int, size int64) ([]byte, error) {
	return nil, ErrHashMmap
}

func unmapFile(data []byte) error {
	return nil
}

func madviseNormal(data []byte) {
}

func getFileDescriptor(f interface {
	Fd() uintptr
}) int {
	return int(f.Fd())
}

func unsafeByteSlice(ptr unsafe.Pointer, len int) []byte {
	return *(*[]byte)(unsafe.Pointer(&struct {
		data uintptr
		len  int
		cap  int
	}{uintptr(ptr), len, len}))
}
