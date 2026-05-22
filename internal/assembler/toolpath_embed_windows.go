//go:build windows

package assembler

import (
	"os"
	"syscall"
	"unsafe"
)

func materializeEmbedded(data []byte, tool string) (string, error) {
	var nameBuf [512]byte
	n := copy(nameBuf[:], os.TempDir())
	if n > 0 && nameBuf[n-1] != '\\' && nameBuf[n-1] != '/' {
		nameBuf[n] = '\\'
		n++
	}
	pfx := []byte("fz-iron-")
	copy(nameBuf[n:], pfx)
	n += len(pfx)
	n += copy(nameBuf[n:], tool)
	ext := []byte{'.', 'e', 'x', 'e'}
	copy(nameBuf[n:], ext)
	n += len(ext)
	path := unsafe.String(&nameBuf[0], n)
	if !validatePathLen(path) {
		rejectPathLen()
	}
	if err := os.WriteFile(path, data, 0o755); err != nil {
		return "", err
	}
	return path, nil
}
