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

package assembler

import (
	"sync"
	"testing"
)

func TestEncoderConcurrencyStress(t *testing.T) {
	const goroutines = 8
	const iterations = 5000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				e := GetEncoder()
				_ = e.WriteByte(byte(i))
				e.Write([]byte("stress-test-encoder"))
				if len(e.Bytes()) == 0 {
					PutEncoder(e)
					continue
				}
				PutEncoder(e)
			}
		}()
	}

	wg.Wait()

}

func TestEncoderGrowthPath(t *testing.T) {
	e := GetEncoder()
	defer PutEncoder(e)

	total := 200000
	chunk := 1024
	data := make([]byte, chunk)
	for i := 0; i < total/chunk; i++ {
		e.Write(data)
	}
	expected := (total / chunk) * chunk
	if len(e.Bytes()) != expected {
		t.Fatalf("expected buffer length %d, got %d", expected, len(e.Bytes()))
	}
}
