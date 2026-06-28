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

package assembler

import "unsafe"

func FastCopy(dst, src unsafe.Pointer, n uintptr) {
	for n >= 64 {
		*(*uint64)(unsafe.Pointer(uintptr(dst))) = *(*uint64)(unsafe.Pointer(uintptr(src)))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 8)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 8))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 16)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 16))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 24)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 24))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 32)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 32))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 40)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 40))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 48)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 48))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 56)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 56))

		dst = unsafe.Pointer(uintptr(dst) + 64)
		src = unsafe.Pointer(uintptr(src) + 64)
		n -= 64
	}
	for n >= 8 {
		*(*uint64)(dst) = *(*uint64)(src)
		dst = unsafe.Pointer(uintptr(dst) + 8)
		src = unsafe.Pointer(uintptr(src) + 8)
		n -= 8
	}
	for n > 0 {
		*(*byte)(dst) = *(*byte)(src)
		dst = unsafe.Pointer(uintptr(dst) + 1)
		src = unsafe.Pointer(uintptr(src) + 1)
		n--
	}
}

func Copy(dst, src unsafe.Pointer, n int) {
	if n <= 0 {
		return
	}
	FastCopy(dst, src, uintptr(n))
}
