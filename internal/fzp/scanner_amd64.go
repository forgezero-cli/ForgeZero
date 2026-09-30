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

package fzp

import (
	"unsafe"

	"golang.org/x/sys/cpu"
)

func scanFZPBlocks(data []byte, out []scanBlock, offsets []uint16) (int, int) {
	blocks := len(data) / 32
	if blocks > len(out) {
		blocks = len(out)
	}
	if offsetBlocks := len(offsets) / 32; blocks > offsetBlocks {
		blocks = offsetBlocks
	}
	if blocks == 0 {
		return 0, 0
	}
	if !cpu.X86.HasAVX2 || !cpu.X86.HasBMI1 {
		return scanFZPBlocksGeneric(data[:blocks*32], out[:blocks], offsets)
	}
	scanned, newlines := scanFZPBlocksAVX2(unsafe.Pointer(unsafe.SliceData(data)), uintptr(blocks), unsafe.Pointer(unsafe.SliceData(out)), unsafe.Pointer(unsafe.SliceData(offsets)))
	return int(scanned), int(newlines)
}

//go:noescape
func scanFZPBlocksAVX2(data unsafe.Pointer, blocks uintptr, out unsafe.Pointer, offsets unsafe.Pointer) (uintptr, uintptr)
