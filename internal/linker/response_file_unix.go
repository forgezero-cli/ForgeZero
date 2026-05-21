//go:build !windows
// +build !windows

package linker

import (
	"os"
	"syscall"
)

func mmapWritableFile(f *os.File, size int) ([]byte, error) {
	return syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
}

func unmapWritableFile(data []byte) error {
	return syscall.Munmap(data)
}
