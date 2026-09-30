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

import (
	"bytes"
	"testing"
)

func TestCopyBytesDisjoint(t *testing.T) {
	for _, size := range []int{0, 1, 31, 32, 33, 4095, 4096, 4097, 8191, 8192, 8193, 65539} {
		source := make([]byte, size+5)
		for index := range source {
			source[index] = byte(index*31 + 7)
		}
		destination := bytes.Repeat([]byte{0xa5}, size+7)
		got := CopyBytesDisjoint(destination[3:3+size], source[2:2+size])
		if got != size {
			t.Fatalf("size %d: copied %d bytes", size, got)
		}
		if !bytes.Equal(destination[3:3+size], source[2:2+size]) {
			t.Fatalf("size %d: copied content mismatch", size)
		}
		if destination[0] != 0xa5 || destination[2] != 0xa5 || destination[3+size] != 0xa5 {
			t.Fatalf("size %d: wrote outside destination", size)
		}
	}
}

func BenchmarkCopyBytesDisjointAVX2(b *testing.B) {
	source := make([]byte, 1<<20)
	destination := make([]byte, len(source))
	b.SetBytes(int64(len(source)))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		CopyBytesDisjoint(destination, source)
	}
}

func BenchmarkCopyBytesBuiltin(b *testing.B) {
	source := make([]byte, 1<<20)
	destination := make([]byte, len(source))
	b.SetBytes(int64(len(source)))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		copy(destination, source)
	}
}
