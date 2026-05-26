//go:build darwin
// +build darwin

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

func lockFileShared(fd int) error {
	return nil
}

func unlockFile(fd int) error {
	return nil
}
