//go:build linux
// +build linux

package utils

import (
    "io"
    "syscall"
    "unsafe"

    fzvfs "fz/internal/fs"
    "github.com/zeebo/blake3"
)

func openRawPath(path string) (int, error) {
    const atFDCWD = ^uintptr(0) - 99
    var buf [4096]byte
    if len(path) >= len(buf) {
        return -1, syscall.ENAMETOOLONG
    }
    n := copy(buf[:], path)
    buf[n] = 0
    r0, _, errno := syscall.Syscall(syscall.SYS_OPENAT, atFDCWD, uintptr(unsafe.Pointer(&buf[0])), uintptr(syscall.O_RDONLY|syscall.O_CLOEXEC))
    if errno != 0 {
        return -1, errno
    }
    return int(r0), nil
}

func hashRawFileDigest(path string) ([32]byte, error) {
    var out [32]byte
    if fileSystem() != fzvfs.Default {
        f, err := openVerified(path)
        if err != nil {
            return out, ErrHashOpen
        }
        hasher := hasherPool.Get().(*blake3.Hasher)
        var buf [65536]byte
        if _, err := io.CopyBuffer(hasher, f, buf[:]); err != nil {
            hasher.Reset()
            hasherPool.Put(hasher)
            f.Close()
            return out, err
        }
        if cerr := f.Close(); cerr != nil {
            hasher.Reset()
            hasherPool.Put(hasher)
            return out, cerr
        }
        digest := hasher.Digest()
        if _, err := digest.Read(out[:]); err != nil {
            hasher.Reset()
            hasherPool.Put(hasher)
            return out, err
        }
        hasher.Reset()
        hasherPool.Put(hasher)
        return out, nil
    }

    fd, err := openRawPath(path)
    if err != nil {
        return out, ErrHashOpen
    }
    hasher := hasherPool.Get().(*blake3.Hasher)
    var buf [65536]byte
    for {
        n, readErr := syscall.Read(fd, buf[:])
        if n > 0 {
            if _, err := hasher.Write(buf[:n]); err != nil {
                syscall.Close(fd)
                hasher.Reset()
                hasherPool.Put(hasher)
                return out, err
            }
        }
        if readErr != nil {
            syscall.Close(fd)
            hasher.Reset()
            hasherPool.Put(hasher)
            return out, ErrHashRead
        }
        if n == 0 {
            break
        }
    }
    syscall.Close(fd)
    digest := hasher.Digest()
    if _, err := digest.Read(out[:]); err != nil {
        hasher.Reset()
        hasherPool.Put(hasher)
        return out, err
    }
    hasher.Reset()
    hasherPool.Put(hasher)
    return out, nil
}
