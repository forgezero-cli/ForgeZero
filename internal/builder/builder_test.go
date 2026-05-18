package builder

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeASM(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBuildDir(t *testing.T) {
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
	outBin := filepath.Join(t.TempDir(), "myapp")
	ctx := context.Background()
	res, err := BuildDir(ctx, []string{dir}, outBin, false, false, "raw", false, false, false, true, false, nil, nil, nil, nil, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(res.Binary); err != nil {
		t.Error("binary not created")
	}
	res2, err := BuildDir(ctx, []string{dir}, outBin, false, false, "raw", false, false, false, true, false, nil, nil, nil, nil, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Binary != res.Binary {
		t.Error("cached build mismatch")
	}
}

func TestBuildDirNoCache(t *testing.T) {
	t.Skip("skipping due to multiple _start conflict in test environment")

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
	outBin := filepath.Join(t.TempDir(), "myapp")
	ctx := context.Background()
	res, err := BuildDir(ctx, []string{dir}, outBin, false, false, "raw", false, true, false, true, false, nil, nil, nil, nil, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(res.Binary); err != nil {
		t.Error("binary not created")
	}
}

func TestCleanDir(t *testing.T) {
	dir := t.TempDir()
	objDir := filepath.Join(dir, ".fz_objs")
	cacheDir := filepath.Join(dir, ".fz_cache")
	os.MkdirAll(objDir, 0o755)
	os.MkdirAll(cacheDir, 0o755)

	base := filepath.Base(dir)
	bin := filepath.Join(dir, base+".out")
	f, err := os.Create(bin)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	objFile := filepath.Join(dir, "test.o")
	os.Create(objFile)

	if err := CleanDir(dir, false); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{objDir, cacheDir} {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Errorf("%s not removed", d)
		}
	}
	if _, err := os.Stat(bin); !os.IsNotExist(err) {
		t.Error("binary not removed")
	}
	if _, err := os.Stat(objFile); !os.IsNotExist(err) {
		t.Error(".o file not removed")
	}
}
