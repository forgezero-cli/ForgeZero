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
	"syscall"
)

func mmapFile(fd int, size int64) ([]byte, error) {
	sizeInt := int(size)
	data, err := syscall.Mmap(fd, 0, sizeInt, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func unmapFile(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return syscall.Munmap(data)
}

func madviseNormal(data []byte) {
	if len(data) > 0 {
		_ = syscall.Madvise(data, syscall.MADV_NORMAL)
	}
}

func getFileDescriptor(f interface {
	Fd() uintptr
}) int {
	return int(f.Fd())
}

func lockFileShared(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_SH)
}

func unlockFile(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_UN)
}
