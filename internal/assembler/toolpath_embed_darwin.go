//go:build darwin
// +build darwin

package assembler

import (
	"os"
	"unsafe"
)

func materializeEmbedded(data []byte, tool string) (string, error) {
	return materializeEmbeddedFile(data, tool)
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
