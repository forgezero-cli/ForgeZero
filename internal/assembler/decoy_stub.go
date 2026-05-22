//go:build !linux

package assembler

import "os"

func emitDecoyObject(path string) error {
    const size = 4096
    data := make([]byte, size)
    copy(data[:4], []byte{0x7f, 'E', 'L', 'F'})
    for i := 4; i < size; i++ {
        data[i] = byte((i*37)^0x55)
    }
    return os.WriteFile(path, data, 0o600)
}
