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

import "unsafe"

func (b *AtomicBitmap) SetAtomic(index int) {
	word, bit, ok := b.bitWord(index)
	if ok {
		bitmapSetAtomicAsm(unsafe.Pointer(word), bit)
	}
}

func (b *AtomicBitmap) TestAndSet(index int) bool {
	word, bit, ok := b.bitWord(index)
	if !ok {
		return false
	}
	return bitmapTestAndSetAsm(unsafe.Pointer(word), bit)
}

func (b *AtomicBitmap) ClearAtomic(index int) {
	word, bit, ok := b.bitWord(index)
	if ok {
		bitmapClearAtomicAsm(unsafe.Pointer(word), bit)
	}
}

//go:noescape
func bitmapSetAtomicAsm(word unsafe.Pointer, bit uint64)

//go:noescape
func bitmapTestAndSetAsm(word unsafe.Pointer, bit uint64) bool

//go:noescape
func bitmapClearAtomicAsm(word unsafe.Pointer, bit uint64)
