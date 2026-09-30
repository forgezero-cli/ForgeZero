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

package gloria

import (
	"encoding/binary"
	"unsafe"

	"golang.org/x/sys/cpu"
)

func writeBareMetalPrintHeader(dst []byte, length uint64, displacement int32) {
	var patch [32]byte
	binary.LittleEndian.PutUint64(patch[12:20], length)
	binary.LittleEndian.PutUint32(patch[23:27], uint32(displacement))
	if cpu.X86.HasAVX2 {
		writeBareMetalPrintHeaderAVX2(
			unsafe.Pointer(unsafe.SliceData(dst)),
			unsafe.Pointer(unsafe.SliceData(bareMetalPrintHeaderTemplate[:])),
			unsafe.Pointer(unsafe.SliceData(bareMetalPrintHeaderMask[:])),
			unsafe.Pointer(unsafe.SliceData(patch[:])),
		)
		return
	}
	copy(dst, bareMetalPrintHeaderTemplate[:])
	binary.LittleEndian.PutUint64(dst[12:20], length)
	binary.LittleEndian.PutUint32(dst[23:27], uint32(displacement))
}

//go:noescape
func writeBareMetalPrintHeaderAVX2(dst, template, mask, patch unsafe.Pointer)
