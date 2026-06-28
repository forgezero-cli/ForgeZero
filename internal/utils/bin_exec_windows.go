//go:build windows
// +build windows

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
	"unsafe"

	"golang.org/x/sys/windows"
)

func execRawMap(size int) ([]byte, error) {
	addr, err := windows.VirtualAlloc(0, uintptr(size), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		return nil, err
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(addr)), size), nil
}

func execRawProtect(data []byte) error {
	old := uint32(0)
	return windows.VirtualProtect(uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), windows.PAGE_EXECUTE_READ, &old)
}

func execRawUnmap(data []byte) error {
	return windows.VirtualFree(uintptr(unsafe.Pointer(&data[0])), 0, windows.MEM_RELEASE)
}
