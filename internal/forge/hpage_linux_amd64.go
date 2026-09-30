//go:build linux && amd64

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

import "unsafe"

const (
	mapPrivate    = 0x02
	mapAnonymous  = 0x20
	mapHugeTLB    = 0x40000
	protReadWrite = 0x03
	HugePageSize  = 2 << 20
)

func MmapHugepages(size uint64) unsafe.Pointer {
	if size == 0 || size > uint64(^uintptr(0)) {
		return nil
	}
	baseFlags := uintptr(mapPrivate | mapAnonymous)
	if size%HugePageSize == 0 {
		if address := mmapSyscall(uintptr(size), baseFlags|mapHugeTLB); int64(uintptr(address)) >= 0 {
			return address
		}
	}
	address := mmapSyscall(uintptr(size), baseFlags)
	if int64(uintptr(address)) < 0 {
		return nil
	}
	return address
}

func MunmapHugepages(pointer unsafe.Pointer, size uint64) {
	if pointer == nil || size == 0 || size > uint64(^uintptr(0)) {
		return
	}
	munmapSyscall(pointer, uintptr(size))
}

//go:noescape
func mmapSyscall(size uintptr, flags uintptr) unsafe.Pointer

//go:noescape
func munmapSyscall(pointer unsafe.Pointer, size uintptr)
