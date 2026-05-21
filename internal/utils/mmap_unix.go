//go:build unix

package utils

import (
	"syscall"
	"unsafe"
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
		syscall.Madvise(data, syscall.MADV_NORMAL)
	}
}

func getFileDescriptor(f interface {
	Fd() uintptr
}) int {
	return int(f.Fd())
}

func unsafeByteSlice(ptr unsafe.Pointer, len int) []byte {
	return unsafe.Slice((*byte)(ptr), len)
}
