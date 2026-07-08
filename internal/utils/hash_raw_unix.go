//go:build linux
// +build linux

/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package utils

import (
	"io"
	"syscall"
	"unsafe"

	fzvfs "github.com/forgezero-cli/ForgeZero/internal/fs"
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
		hasher := getKeyedHasher()
		var buf [65536]byte
		if _, err := io.CopyBuffer(hasher, f, buf[:]); err != nil {
			putKeyedHasher(hasher)
			f.Close()
			return out, err
		}
		if cerr := f.Close(); cerr != nil {
			putKeyedHasher(hasher)
			return out, cerr
		}
		digest := hasher.Digest()
		if _, err := digest.Read(out[:]); err != nil {
			putKeyedHasher(hasher)
			return out, err
		}
		putKeyedHasher(hasher)
		return out, nil
	}

	fd, err := openRawPath(path)
	if err != nil {
		return out, ErrHashOpen
	}
	hasher := getKeyedHasher()
	var buf [65536]byte
	for {
		n, readErr := syscall.Read(fd, buf[:])
		if n > 0 {
			if _, err := hasher.Write(buf[:n]); err != nil {
				syscall.Close(fd)
				putKeyedHasher(hasher)
				return out, err
			}
		}
		if readErr != nil {
			syscall.Close(fd)
			putKeyedHasher(hasher)
			return out, ErrHashRead
		}
		if n == 0 {
			break
		}
	}
	syscall.Close(fd)
	digest := hasher.Digest()
	if _, err := digest.Read(out[:]); err != nil {
		putKeyedHasher(hasher)
		return out, err
	}
	putKeyedHasher(hasher)
	return out, nil
}
