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
	"encoding/binary"
	"math/bits"
)

func hashForgeScalar(data []byte, seed uint64) uint64 {
	var lanes [4]uint64
	for offset := 0; offset+32 <= len(data); offset += 32 {
		for lane := range lanes {
			value := binary.LittleEndian.Uint64(data[offset+lane*8:]) ^ seed
			mixed := value ^ value>>29 ^ value<<35
			mixed ^= uint64(uint32(mixed) * 0x7f4a7c15)
			lanes[lane] ^= mixed
		}
	}
	tail := seed
	for _, value := range data[len(data)&^31:] {
		tail ^= uint64(value)
		tail = bits.RotateLeft64(tail, 13) * 0x9e3779b97f4a7c15
	}
	hash := lanes[0] ^ lanes[1] ^ lanes[2] ^ lanes[3] ^ tail
	hash ^= hash >> 33
	hash *= 0x9e3779b97f4a7c15
	return hash ^ hash>>29
}
