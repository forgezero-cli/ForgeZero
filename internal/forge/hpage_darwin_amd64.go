//go:build darwin && amd64

/*
 *   Copyright (c) 2026 forgezero-cli
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

package forge

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

const HugePageSize = 2 << 20

func MmapHugepages(size uint64) unsafe.Pointer {
	if size == 0 || size > uint64(^uint(0)>>1) {
		return nil
	}
	mapping, err := unix.Mmap(-1, 0, int(size), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANON)
	if err != nil || len(mapping) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.SliceData(mapping))
}

func MunmapHugepages(pointer unsafe.Pointer, size uint64) {
	if pointer == nil || size == 0 || size > uint64(^uint(0)>>1) {
		return
	}
	_ = unix.Munmap(unsafe.Slice((*byte)(pointer), int(size)))
}
