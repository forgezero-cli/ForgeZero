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

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	if len(left) == 0 {
		return true
	}
	return bytesEqualAsm(unsafe.Pointer(&left[0]), unsafe.Pointer(&right[0]), uintptr(len(left)))
}

func copyHashPair(dst *[64]byte, left, right *[32]byte) {
	copyHashPairAsm(unsafe.Pointer(&dst[0]), unsafe.Pointer(&left[0]), unsafe.Pointer(&right[0]))
}

func findBoomDelim(data []byte) int {
	if len(data) == 0 {
		return -1
	}
	vectorEnd := len(data) &^ 15
	if offset := findBoomDelimAsm(unsafe.Pointer(&data[0]), uintptr(len(data))); offset < vectorEnd {
		return offset
	}
	for i := vectorEnd; i < len(data); i++ {
		switch data[i] {
		case '{', '}', ';', '\n':
			return i
		}
	}
	return -1
}
