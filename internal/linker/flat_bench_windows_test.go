//go:build windows
// +build windows

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

package linker

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestCopyFileHotAllocations(t *testing.T) {
	dir, err := ioutil.TempDir("", "fz_test_copy")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	data := make([]byte, 1024)
	if err := ioutil.WriteFile(src, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f := func() {
		_ = copyFileHot(src, dst)
		_ = syscall.DeleteFile(dst)
	}
	allocs := testing.AllocsPerRun(20, f)
	if allocs > 0 {
		t.Fatalf("copyFileHot allocs = %v > 0", allocs)
	}
}

func BenchmarkCopyFileHot(b *testing.B) {
	dir, err := ioutil.TempDir("", "fz_bench_copy")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	data := make([]byte, 1024*8)
	if err := ioutil.WriteFile(src, data, 0o644); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := copyFileHot(src, dst); err != nil {
			b.Fatal(err)
		}
		_ = syscall.DeleteFile(dst)
	}
}
