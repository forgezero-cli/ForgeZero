//go:build unix
// +build unix

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
	"unsafe"
)

func execRawMap(size int) ([]byte, error) {
	mem, err := syscall.Mmap(-1, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		return nil, err
	}
	return mem, nil
}

func execRawProtect(data []byte) error {
	if len(data) == 0 {
		return syscall.EINVAL
	}
	page := syscall.Getpagesize()
	if uintptr(unsafe.Pointer(&data[0]))%uintptr(page) != 0 {
		return syscall.EINVAL
	}
	_, _, errno := syscall.Syscall(syscall.SYS_MSYNC, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), uintptr(syscall.MS_SYNC))
	if errno != 0 {
		return errno
	}
	return syscall.Mprotect(data, syscall.PROT_READ|syscall.PROT_EXEC)
}

func execRawUnmap(data []byte) error {
	return syscall.Munmap(data)
}
