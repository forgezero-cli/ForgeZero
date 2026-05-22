//go:build linux

package assembler

import (
    "os"
    "syscall"
)

func emitDecoyObject(path string) error {
    const size = 4096
    buf := make([]byte, size)
    copy(buf[:4], []byte{0x7f, 'E', 'L', 'F'})
    for i := 4; i < size; i++ {
        buf[i] = byte((i*31)^0xAA)
    }
    f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC|syscall.O_CLOEXEC, 0o600)
    if err != nil {
        return err
    }
    defer f.Close()
    if err := f.Truncate(int64(size)); err != nil {
        return err
    }
    data, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
    if err != nil {
        return err
    }
    copy(data, buf)
    return syscall.Munmap(data)
}
