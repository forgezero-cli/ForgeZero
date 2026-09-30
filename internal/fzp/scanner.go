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

import "math/bits"

type scanBlock struct {
	newlines uint32
	hashes   uint32
	spaces   uint32
}

func scanFZPBlocksGeneric(data []byte, out []scanBlock, offsets []uint16) (int, int) {
	blocks := len(data) / 32
	if blocks > len(out) {
		blocks = len(out)
	}
	if offsetBlocks := len(offsets) / 32; blocks > offsetBlocks {
		blocks = offsetBlocks
	}
	newlineCount := 0
	for blockIndex := 0; blockIndex < blocks; blockIndex++ {
		var block scanBlock
		for offset, value := range data[blockIndex*32 : blockIndex*32+32] {
			bit := uint32(1) << uint(offset)
			switch value {
			case '\n':
				block.newlines |= bit
			case '#':
				block.hashes |= bit
			case ' ', '\t', '\r':
				block.spaces |= bit
			}
		}
		out[blockIndex] = block
		newlines := block.newlines
		for newlines != 0 {
			offsets[newlineCount] = uint16(blockIndex*32 + bits.TrailingZeros32(newlines))
			newlineCount++
			newlines &= newlines - 1
		}
	}
	return blocks, newlineCount
}
