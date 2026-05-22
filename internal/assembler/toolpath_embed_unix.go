//go:build unix

package assembler

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func materializeEmbedded(data []byte, tool string) (string, error) {
	fd, err := memfdCreate(tool)
	if err == nil {
		if p, werr := writeMemfdPath(fd, data); werr == nil {
			return p, nil
		}
		syscall.Close(fd)
	}
	return materializeEmbeddedFile(data, tool)
}

func writeMemfdPath(fd int, data []byte) (string, error) {
	off := 0
	for off < len(data) {
		n, werr := syscall.Write(fd, data[off:])
		if werr != nil {
			syscall.Close(fd)
			return "", werr
		}
		if n == 0 {
			break
		}
		off += n
	}
	_ = syscall.Fchmod(fd, 0o755)
	var pathBuf [64]byte
	prefix := []byte("/proc/self/fd/")
	n := copy(pathBuf[:], prefix)
	nb := intToDec(pathBuf[n:], fd)
	n += nb
	syscall.Close(fd)
	path := unsafe.String(&pathBuf[0], n)
	if !validatePathLen(path) {
		rejectPathLen()
	}
	return path, nil
}

func memfdCreate(name string) (int, error) {
	return unix.MemfdCreate(name, 0)
}

func intToDec(buf []byte, v int) int {
	if v == 0 {
		buf[0] = '0'
		return 1
	}
	var tmp [20]byte
	i := 0
	for v > 0 {
		tmp[i] = byte('0' + v%10)
		i++
		v /= 10
	}
	for j := 0; j < i; j++ {
		buf[j] = tmp[i-1-j]
	}
	return i
}

func materializeEmbeddedFile(data []byte, tool string) (string, error) {
	var nameBuf [512]byte
	n := copy(nameBuf[:], os.TempDir())
	if n > 0 && nameBuf[n-1] != '/' {
		nameBuf[n] = '/'
		n++
	}
	pfx := []byte("fz-iron-")
	copy(nameBuf[n:], pfx)
	n += len(pfx)
	n += copy(nameBuf[n:], tool)
	path := unsafe.String(&nameBuf[0], n)
	if !validatePathLen(path) {
		rejectPathLen()
	}
	if err := os.WriteFile(path, data, 0o755); err != nil {
		return "", err
	}
	return path, nil
}
