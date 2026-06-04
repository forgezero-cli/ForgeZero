//go:build !windows
// +build !windows

package linker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFileHotAllocations(t *testing.T) {
	dir, err := os.MkdirTemp("", "fz_test_copy-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	defer os.RemoveAll(dir)
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	data := make([]byte, 1024)
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f := func() {
		_ = copyFileHot(src, dst)
		_ = unlinkHot(dst)
	}
	allocs := testing.AllocsPerRun(20, f)
	if allocs > 0 {
		t.Fatalf("copyFileHot allocs = %v > 0", allocs)
	}
}

func BenchmarkCopyFileHot(b *testing.B) {
	dir, err := os.MkdirTemp("", "fz_bench_copy")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	data := make([]byte, 1024*8)
	if err := os.WriteFile(src, data, 0o644); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := copyFileHot(src, dst); err != nil {
			b.Fatal(err)
		}
		_ = unlinkHot(dst)
	}
}
