//go:build !unix && !windows

package utils

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
