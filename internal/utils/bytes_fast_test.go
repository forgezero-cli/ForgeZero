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

import (
	"bytes"
	"testing"
)

func TestBytesEqualAsm(t *testing.T) {
	for size := 0; size < 97; size++ {
		left := bytes.Repeat([]byte{0x5a}, size)
		right := append([]byte(nil), left...)
		if !bytesEqual(left, right) {
			t.Fatalf("equal buffers rejected at size %d", size)
		}
		if size > 0 {
			right[size-1] ^= 1
			if bytesEqual(left, right) {
				t.Fatalf("different buffers accepted at size %d", size)
			}
		}
	}
}

func TestCopyHashPairAsm(t *testing.T) {
	var left, right [32]byte
	for i := range left {
		left[i] = byte(i)
		right[i] = byte(255 - i)
	}
	var got [64]byte
	copyHashPair(&got, &left, &right)
	if !bytes.Equal(got[:32], left[:]) || !bytes.Equal(got[32:], right[:]) {
		t.Fatal("hash pair copy mismatch")
	}
}

func TestFindBoomDelimAsm(t *testing.T) {
	data := bytes.Repeat([]byte{'a'}, 64)
	data[37] = '{'
	if got := findBoomDelim(data); got != 37 {
		t.Fatalf("delimiter offset = %d, want 37", got)
	}
	if got := findBoomDelim(bytes.Repeat([]byte{'a'}, 31)); got != -1 {
		t.Fatalf("delimiter tail offset = %d, want -1", got)
	}
}
