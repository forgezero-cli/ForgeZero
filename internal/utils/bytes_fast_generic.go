//go:build !amd64

/*
 * Copyright (c) 2026 forgezero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package utils

import "unsafe"

func bytesEqualAsm(left, right unsafe.Pointer, n uintptr) bool {
	if n == 0 {
		return true
	}
	a := unsafe.Slice((*byte)(left), n)
	b := unsafe.Slice((*byte)(right), n)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func copyHashPairAsm(dst, left, right unsafe.Pointer) {
	d := unsafe.Slice((*byte)(dst), 64)
	l := unsafe.Slice((*byte)(left), 32)
	r := unsafe.Slice((*byte)(right), 32)
	copy(d, l)
	copy(d[32:], r)
}

func findBoomDelimAsm(data unsafe.Pointer, n uintptr) int {
	b := unsafe.Slice((*byte)(data), n)
	for i, value := range b {
		switch value {
		case '{', '}', ';', '\n':
			return i
		}
	}
	return -1
}
