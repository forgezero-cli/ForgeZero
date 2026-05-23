//go:build darwin
// +build darwin

package utils

import (
    "io"
    "os"

    fzvfs "fz/internal/fs"
    "github.com/zeebo/blake3"
)

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

    f, err := os.Open(path)
    if err != nil {
        return out, ErrHashOpen
    }
    defer f.Close()

    hasher := hasherPool.Get().(*blake3.Hasher)
    var buf [65536]byte
    if _, err := io.CopyBuffer(hasher, f, buf[:]); err != nil {
        hasher.Reset()
        hasherPool.Put(hasher)
        return out, err
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
