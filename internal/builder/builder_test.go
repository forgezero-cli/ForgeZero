package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNoSourceFiles(t *testing.T) {
	dir := t.TempDir()
	_, err := BuildDir(context.Background(), dir, "", false, false, "auto", false, false)
	if err == nil {
		t.Error("expected error for empty directory")
	}
}

func TestCleanDir(t *testing.T) {
	dir := t.TempDir()
	objDir := filepath.Join(dir, ".fz_objs")
	cacheDir := filepath.Join(dir, ".fz_cache")
	os.MkdirAll(objDir, 0755)
	os.MkdirAll(cacheDir, 0755)

	base := filepath.Base(dir)
	bin := filepath.Join(dir, base+".out")
	f, err := os.Create(bin)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	objFile := filepath.Join(dir, "test.o")
	f2, _ := os.Create(objFile)
	f2.Close()

	if err := CleanDir(dir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(objDir); !os.IsNotExist(err) {
		t.Error(".fz_objs not removed")
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Error(".fz_cache not removed")
	}
	if _, err := os.Stat(bin); !os.IsNotExist(err) {
		t.Error("binary not removed")
	}
	if _, err := os.Stat(objFile); !os.IsNotExist(err) {
		t.Error(".o file not removed")
	}
}
