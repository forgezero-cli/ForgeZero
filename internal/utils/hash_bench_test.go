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

package utils

import (
	"os"
	"testing"
)

func BenchmarkHashFile100MB(b *testing.B) {
	tmpFile, err := os.CreateTemp("", "bench100")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	data := make([]byte, 100*1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	if _, err := tmpFile.Write(data); err != nil {
		b.Fatal(err)
	}
	tmpFile.Close()

	filePath := tmpFile.Name()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := hashRawFileDigest(filePath)
		if err != nil {
			b.Fatal(err)
		}
	}
}
