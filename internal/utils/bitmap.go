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

type AtomicBitmap struct {
	words []uint64
}

func NewAtomicBitmap(bitCount int) AtomicBitmap {
	if bitCount <= 0 {
		return AtomicBitmap{}
	}
	wordCount := bitCount / 64
	if bitCount%64 != 0 {
		wordCount++
	}
	return AtomicBitmap{words: make([]uint64, wordCount)}
}

func (b *AtomicBitmap) bitWord(index int) (*uint64, uint64, bool) {
	if b == nil || index < 0 || index/64 >= len(b.words) {
		return nil, 0, false
	}
	return &b.words[index/64], uint64(index & 63), true
}
