package builder

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func TestCollectSourceFilesWalk(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.c"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := CollectSourceFiles(nil, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || !strings.HasSuffix(files[0], "a.c") {
		t.Fatalf("got %v", files)
	}
}

func TestCollectSourceFilesWalkError(t *testing.T) {
	_, err := CollectSourceFiles(nil, []string{"/nonexistent-dir-xyz"})
	if err == nil {
		t.Fatal("expected walk error")
	}
}

func TestCheckCacheHit(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.c")
	if err := os.WriteFile(src, []byte("int x;"), 0o644); err != nil {
		t.Fatal(err)
	}
	cacheDir := filepath.Join(dir, ".fz_cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "obj.o")
	if err := os.WriteFile(obj, []byte{1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := storeCache(src, obj, cacheDir, false, false, "auto"); err != nil {
		t.Fatal(err)
	}
	got, err := checkCache(src, cacheDir, false, false, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected cache hit")
	}
}

func TestStoreShadowCacheHardLink(t *testing.T) {
	dir := t.TempDir()
	if err := os.Setenv("XDG_CACHE_HOME", dir); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("XDG_CACHE_HOME")

	src := filepath.Join(dir, "src.c")
	if err := os.WriteFile(src, []byte("int x=0;"), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "build", "src.o")
	if err := os.MkdirAll(filepath.Dir(obj), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(obj, []byte{1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := storeShadowCache(src, obj, false, "auto"); err != nil {
		t.Fatal(err)
	}
	key, err := utils.ShadowCacheKey(src, []string{"debug=false", "mode=auto"})
	if err != nil {
		t.Fatal(err)
	}
	shadowObj := utils.ShadowCachePath(key)
	info1, err := os.Stat(obj)
	if err != nil {
		t.Fatal(err)
	}
	info2, err := os.Stat(shadowObj)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(info1, info2) {
		t.Fatal("expected hard link in shadow cache")
	}
}

func TestRestoreShadowCache(t *testing.T) {
	dir := t.TempDir()
	if err := os.Setenv("XDG_CACHE_HOME", dir); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("XDG_CACHE_HOME")

	src := filepath.Join(dir, "src.c")
	if err := os.WriteFile(src, []byte("int y=1;"), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "build", "src.o")
	if err := os.MkdirAll(filepath.Dir(obj), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(obj, []byte{1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := storeShadowCache(src, obj, false, "auto"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj); err != nil {
		t.Fatal(err)
	}
	restored, err := restoreShadowCache(src, obj, false, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if !restored {
		t.Fatal("expected shadow cache restore")
	}
	got, err := os.ReadFile(obj)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("unexpected obj contents: %v", got)
	}
}

func TestCheckCacheEmptyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.c")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cacheDir := filepath.Join(dir, ".fz_cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := storeCache(src, src, cacheDir, false, false, "auto"); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(cacheDir)
	for _, e := range entries {
		p := filepath.Join(cacheDir, e.Name())
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := checkCache(src, cacheDir, false, false, "auto")
		if err == nil || !strings.Contains(err.Error(), "empty") {
			t.Fatalf("got %v", err)
		}
	}
}

func TestBuildDirStaticLibrary(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	if _, err := exec.LookPath("ar"); err != nil {
		t.Skip("ar not installed")
	}
	dir := t.TempDir()
	writeASM(t, dir, "a.asm", `
section .text
global _start
_start:
	mov eax, 60
	xor edi, edi
	syscall
`)
	out := filepath.Join(t.TempDir(), "lib.a")
	_, err := BuildDir(context.Background(), []string{dir}, out, false, true, "raw", false, true, true, true, false, nil, nil, nil, nil, nil, 1, "static")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
}

func TestBuildDirWithExclude(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	dir := t.TempDir()
	writeASM(t, dir, "main.asm", `
section .text
global _start
_start:
	mov eax, 60
	xor edi, edi
	syscall
`)
	writeASM(t, dir, "skip.asm", `
section .text
global skip
skip:
	ret
`)
	out := filepath.Join(t.TempDir(), "app")
	_, err := BuildDir(context.Background(), []string{dir}, out, false, false, "raw", false, true, true, true, false, []string{"skip.asm"}, nil, nil, nil, nil, 1, "executable")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCleanDirVerboseExecutableNoExt(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "mybin")
	if err := os.WriteFile(bin, []byte{0x7f, 'E', 'L', 'F'}, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CleanDir(dir, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); !os.IsNotExist(err) {
		t.Fatal("binary not removed")
	}
}

func TestCleanDirReadDirFail(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0o755) }()
	if err := CleanDir(dir, false); err == nil {
		t.Fatal("expected readdir error")
	}
}

func TestStoreCacheHashFail(t *testing.T) {
	err := storeCache("/nonexistent", "obj", t.TempDir(), false, false, "auto")
	if err == nil {
		t.Fatal("expected hash error")
	}
}
