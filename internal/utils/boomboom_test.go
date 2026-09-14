/*
 *   BOOMBOOM HASH by ForgeZero CLI / AlexVoste - TEST FILE for boomboom.go
 *
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
	"os"
	"testing"

	"github.com/zeebo/blake3"
)

func TestBoomBoomContextChangesDigest(t *testing.T) {
	data := []byte("int main(void) { return 0; }\n")
	first, err := NewBoomBoomHasher(BoomBoomContext{Compiler: "gcc", Version: "14", Target: "x86_64-linux-gnu", Flags: []string{"-O3"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewBoomBoomHasher(BoomBoomContext{Compiler: "clang", Version: "18", Target: "x86_64-linux-gnu", Flags: []string{"-O3"}})
	if err != nil {
		t.Fatal(err)
	}
	left, err := first.Hash(data)
	if err != nil {
		t.Fatal(err)
	}
	right, err := second.Hash(data)
	if err != nil {
		t.Fatal(err)
	}
	if left == right {
		t.Fatal("context did not change digest")
	}
}

func TestBoomBoomHasher_Incremental(t *testing.T) {
	data := []byte("#include <stdio.h>\n\nint first(void) { return 1; }\nint second(void) { return 2; }\n")
	hasher, err := NewBoomBoomHasher(BoomBoomContext{Compiler: "gcc", Target: "x86_64-linux-gnu"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := hasher.Hash(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(hasher.Chunks()) != 3 {
		t.Fatalf("chunks = %d, want 3", len(hasher.Chunks()))
	}
	data[bytes.Index(data, []byte("return 2"))+len("return ")] = '3'
	second, err := hasher.Update(data)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("digest did not change")
	}
	if hasher.ChangedChunks() != 1 {
		t.Fatalf("changed chunks = %d, want 1", hasher.ChangedChunks())
	}
	full, err := NewBoomBoomHasher(BoomBoomContext{Compiler: "gcc", Target: "x86_64-linux-gnu"})
	if err != nil {
		t.Fatal(err)
	}
	want, err := full.Hash(data)
	if err != nil {
		t.Fatal(err)
	}
	if second != want {
		t.Fatal("incremental digest differs from full digest")
	}
}

func TestBoomBoomHasher_RepeatedInput(t *testing.T) {
	data := []byte("int main(void) { return 0; }\n")
	hasher, err := NewBoomBoomHasher(BoomBoomContext{Target: "x86_64-linux-gnu"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := hasher.Hash(data)
	if err != nil {
		t.Fatal(err)
	}
	second, err := hasher.Update(data)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("repeated input changed digest")
	}
	if hasher.ChangedChunks() != 0 {
		t.Fatalf("changed chunks = %d, want 0", hasher.ChangedChunks())
	}
}

func TestBoomBoomHasher_ParallelLeaves(t *testing.T) {
	data := []byte("int a(void) { return 1; }\nint b(void) { return 1; }\nint c(void) { return 1; }\nint d(void) { return 1; }\nint e(void) { return 1; }\nint f(void) { return 1; }\n")
	hasher, err := NewBoomBoomHasher(BoomBoomContext{Target: "x86_64-linux-gnu"})
	if err != nil {
		t.Fatal(err)
	}
	defer hasher.Close()
	if _, err := hasher.Hash(data); err != nil {
		t.Fatal(err)
	}
	var positions [5]int
	for i := 0; i < 5; i++ {
		positions[i] = bytes.Index(data, []byte("return 1")) + len("return ")
		data[positions[i]] = byte('2' + i)
	}
	got, err := hasher.Update(data)
	if err != nil {
		t.Fatal(err)
	}
	if hasher.ChangedChunks() != 5 {
		t.Fatalf("changed chunks = %d, want 5", hasher.ChangedChunks())
	}
	full, err := NewBoomBoomHasher(BoomBoomContext{Target: "x86_64-linux-gnu"})
	if err != nil {
		t.Fatal(err)
	}
	want, err := full.Hash(data)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatal("parallel digest differs from full digest")
	}
	allocs := testing.AllocsPerRun(10, func() {
		for _, position := range positions {
			data[position] ^= 1
		}
		_, _ = hasher.Update(data)
	})
	if allocs != 0 {
		t.Fatalf("parallel update allocations = %v, want 0", allocs)
	}
}

func TestBoomBoomHasher_ParallelGrowth(t *testing.T) {
	hasher, err := NewBoomBoomHasher(BoomBoomContext{Target: "x86_64-linux-gnu"})
	if err != nil {
		t.Fatal(err)
	}
	defer hasher.Close()

	initial := []byte("int a(void) { return 1; }\nint b(void) { return 1; }\nint c(void) { return 1; }\nint d(void) { return 1; }\nint e(void) { return 1; }\n")
	if _, err := hasher.Hash(initial); err != nil {
		t.Fatal(err)
	}

	updated := []byte("int a(void) { return 2; }\nint b(void) { return 3; }\nint c(void) { return 4; }\nint d(void) { return 5; }\nint e(void) { return 6; }\nint f(void) { return 7; }\n")
	got, err := hasher.Update(updated)
	if err != nil {
		t.Fatal(err)
	}

	full, err := NewBoomBoomHasher(BoomBoomContext{Target: "x86_64-linux-gnu"})
	if err != nil {
		t.Fatal(err)
	}
	want, err := full.Hash(updated)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatal("parallel growth digest differs from full digest")
	}
}

func TestBoomBoomMappedFile(t *testing.T) {
	path := t.TempDir() + "/source.c"
	data := []byte("int main(void) { return 0; }\n")
	if err := writeFileForBoomBoom(path, data); err != nil {
		t.Fatal(err)
	}
	mapped, err := HashBoomBoomMappedFile(path, BoomBoomContext{Target: "x86_64-linux-gnu"})
	if err != nil {
		t.Fatal(err)
	}
	file, err := HashBoomBoomFile(path, BoomBoomContext{Target: "x86_64-linux-gnu"})
	if err != nil {
		t.Fatal(err)
	}
	if mapped != file {
		t.Fatal("mapped digest differs from file digest")
	}
	largePath := t.TempDir() + "/large.c"
	largeData := bytes.Repeat([]byte("int value(void) { return 42; }\n"), 1024)
	if err := writeFileForBoomBoom(largePath, largeData); err != nil {
		t.Fatal(err)
	}
	largeMapped, err := HashBoomBoomMappedFile(largePath, BoomBoomContext{Target: "x86_64-linux-gnu"})
	if err != nil {
		t.Fatal(err)
	}
	largeFile, err := HashBoomBoomFile(largePath, BoomBoomContext{Target: "x86_64-linux-gnu"})
	if err != nil {
		t.Fatal(err)
	}
	if largeMapped != largeFile {
		t.Fatal("large mapped digest differs from file digest")
	}
}

func writeFileForBoomBoom(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func BenchmarkBoomBoomFullBLAKE3(b *testing.B) {
	data := bytes.Repeat([]byte("int value(void) { return 42; }\n"), 4096)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasher := blake3.New()
		_, _ = hasher.Write(data)
		_ = hasher.Sum(nil)
	}
}

func BenchmarkBoomBoomHasher_Update(b *testing.B) {
	data := bytes.Repeat([]byte("int value(void) { return 42; }\n"), 4096)
	hasher, err := NewBoomBoomHasher(BoomBoomContext{Compiler: "gcc", Version: "14", Target: "x86_64-linux-gnu", Flags: []string{"-O3", "-fPIC"}})
	if err != nil {
		b.Fatal(err)
	}
	if _, err := hasher.Hash(data); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data[len(data)-2] ^= 1
		_, _ = hasher.Update(data)
	}
}
