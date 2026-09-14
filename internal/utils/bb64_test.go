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
	"hash/fnv"
	"math/rand"
	"testing"

	"github.com/zeebo/blake3"
	"golang.org/x/sys/cpu"
)

func TestHashBB64AssemblyStable(t *testing.T) {
	if !hasBB64AVX2() {
		t.Skip("AVX2 unavailable")
	}
	rng := rand.New(rand.NewSource(7))
	for size := 0; size < 257; size++ {
		data := make([]byte, size)
		_, _ = rng.Read(data)
		seed := uint64(size)*0x9e3779b97f4a7c15 + 1
		first := HashBB64Asm(data, seed)
		second := HashBB64Asm(data, seed)
		if first != second {
			t.Fatalf("size %d: unstable result", size)
		}
	}
}

func TestHashBB64Deterministic(t *testing.T) {
	data := bytes.Repeat([]byte("forgezero"), 4096)
	first := HashBB64(data, 0x123456789abcdef0)
	second := HashBB64(data, 0x123456789abcdef0)
	if first != second {
		t.Fatal("BB64 is not deterministic")
	}
	data[10] ^= 1
	if first == HashBB64(data, 0x123456789abcdef0) {
		t.Fatal("BB64 ignored input change")
	}
	short := []byte("int second(void) { return 2; }\n")
	shortFirst := HashBB64(short, 7)
	short[24] = '3'
	if shortFirst == HashBB64(short, 7) {
		t.Fatal("BB64 ignored short input change")
	}
	var firstExpanded, secondExpanded [32]byte
	expandBB64Seed([]byte("int second(void) { return 2; }\n"), 2, 1, 7, &firstExpanded)
	expandBB64Seed([]byte("int second(void) { return 3; }\n"), 2, 1, 7, &secondExpanded)
	if firstExpanded == secondExpanded {
		t.Fatal("BB64 expansion ignored input change")
	}
	var left, right [32]byte
	firstPair := hashBB64Pair(left, right)
	right[0] = 1
	if firstPair == hashBB64Pair(left, right) {
		t.Fatal("BB64 pair ignored input change")
	}
}

func TestHashBB64AssemblyMatchesScalar(t *testing.T) {
	if !hasBB64AVX2() {
		t.Skip("AVX2 unavailable")
	}
	rng := rand.New(rand.NewSource(11))
	for size := 0; size < 513; size++ {
		data := make([]byte, size)
		_, _ = rng.Read(data)
		seed := uint64(size)*0xd6e8feb86659fd93 + 7
		if got, want := HashBB64Asm(data, seed), hashBB64Scalar(data, seed); got != want {
			t.Fatalf("size %d: got %016x want %016x", size, got, want)
		}
	}
}

func BenchmarkHashBB64(b *testing.B) {
	data := bytes.Repeat([]byte("forgezero source block\n"), 4096)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		_ = HashBB64(data, 0x9e3779b97f4a7c15)
	}
}

func BenchmarkHashBB64Scalar(b *testing.B) {
	data := bytes.Repeat([]byte("forgezero source block\n"), 4096)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		_ = hashBB64Scalar(data, 0x9e3779b97f4a7c15)
	}
}

func BenchmarkHashFNV1a(b *testing.B) {
	data := bytes.Repeat([]byte("forgezero source block\n"), 4096)
	hasher := fnv.New64a()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		hasher.Reset()
		_, _ = hasher.Write(data)
		_ = hasher.Sum64()
	}
}

func BenchmarkHashBLAKE3(b *testing.B) {
	data := bytes.Repeat([]byte("forgezero source block\n"), 4096)
	hasher := blake3.New()
	var output [32]byte
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		hasher.Reset()
		_, _ = hasher.Write(data)
		_ = hasher.Sum(output[:0])
	}
}

func hasBB64AVX2() bool {
	return cpu.X86.HasAVX2
}
