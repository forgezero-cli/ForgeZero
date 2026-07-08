/*
 *   Copyright (c) 2026 ForgeZero-cli
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

package hashpool

import (
	"testing"

	"github.com/zeebo/blake3"
)

func BenchmarkBlake3New(b *testing.B) {
	data := []byte("benchmark data for blake3 new")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h := blake3.New()
		_, _ = h.Write(data)
		_ = h.Digest()
	}
}

func BenchmarkHashPoolGetPut(b *testing.B) {
	data := []byte("benchmark data for pooled blake3")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h := GetHasher()
		_, _ = h.Write(data)
		_ = h.Digest()
		PutHasher(h)
	}
}
