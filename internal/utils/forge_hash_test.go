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
	"math/rand"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/forge"
)

func TestHashBB64ForgeDispatchMatchesAssembly(t *testing.T) {
	rng := rand.New(rand.NewSource(29))
	for _, size := range []int{4095, 4096, 4097, 8192, 16387} {
		data := make([]byte, size)
		_, _ = rng.Read(data)
		seed := uint64(size)*0x9e3779b97f4a7c15 + 5
		if got, want := HashBB64(data, seed), HashBB64Asm(data, seed); got != want {
			t.Fatalf("size %d: Forge hash = %016x, assembly hash = %016x", size, got, want)
		}
	}
}

func TestForgeScalarHashMatchesBB64(t *testing.T) {
	for size := 0; size <= 257; size++ {
		data := make([]byte, size)
		for index := range data {
			data[index] = byte(index*37 + size)
		}
		for _, seed := range []uint64{0, 1, 0x9e3779b97f4a7c15} {
			if got, want := forge.HashBytes(data, seed), HashBB64Asm(data, seed); got != want {
				t.Fatalf("size %d seed %x: Forge hash = %x, BB64 = %x", size, seed, got, want)
			}
		}
	}
}

func TestHashBB64ForgeDispatchDoesNotAllocate(t *testing.T) {
	data := make([]byte, forgeHashThreshold+17)
	for index := range data {
		data[index] = byte(index*41 + 3)
	}
	allocs := testing.AllocsPerRun(100, func() {
		_ = HashBB64(data, 0x123456789abcdef0)
	})
	if allocs != 0 {
		t.Fatalf("Forge-routed BB64 allocations = %v, want 0", allocs)
	}
}
