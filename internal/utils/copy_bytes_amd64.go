//go:build amd64

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

package utils

import (
	"unsafe"

	"golang.org/x/sys/cpu"
)

func copyBytesDisjointPlatform(dst, src []byte) {
	if len(src) < 4096 || !cpu.X86.HasAVX2 {
		copy(dst, src)
		return
	}
	copyBytesAVX2(unsafe.Pointer(unsafe.SliceData(dst)), unsafe.Pointer(unsafe.SliceData(src)), uintptr(len(src)))
}

//go:noescape
func copyBytesAVX2(dst, src unsafe.Pointer, size uintptr)
