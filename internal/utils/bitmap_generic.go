//go:build !amd64

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

import "sync/atomic"

func (b *AtomicBitmap) SetAtomic(index int) {
	word, mask, ok := b.bitWord(index)
	if !ok {
		return
	}
	for {
		current := atomic.LoadUint64(word)
		if atomic.CompareAndSwapUint64(word, current, current|1<<mask) {
			return
		}
	}
}

func (b *AtomicBitmap) TestAndSet(index int) bool {
	word, mask, ok := b.bitWord(index)
	if !ok {
		return false
	}
	bit := uint64(1) << mask
	for {
		current := atomic.LoadUint64(word)
		if atomic.CompareAndSwapUint64(word, current, current|bit) {
			return current&bit != 0
		}
	}
}

func (b *AtomicBitmap) ClearAtomic(index int) {
	word, mask, ok := b.bitWord(index)
	if !ok {
		return
	}
	bit := ^(uint64(1) << mask)
	for {
		current := atomic.LoadUint64(word)
		if atomic.CompareAndSwapUint64(word, current, current&bit) {
			return
		}
	}
}
