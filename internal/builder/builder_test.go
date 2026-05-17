package builder

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	res, err := BuildDir(ctx, []string{dir}, outBin, false, false, "raw", false, false, false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(res.Binary); err != nil {
		t.Error("binary not created")
	}
	res2, err := BuildDir(ctx, []string{dir}, outBin, false, false, "raw", false, false, false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Binary != res.Binary {
		t.Error("cached build mismatch")
	}
}

func TestBuildDirNoCache(t *testing.T) {
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
	res, err := BuildDir(ctx, []string{dir}, outBin, false, false, "raw", false, true, false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(res.Binary); err != nil {
		t.Error("binary not created")
	}
}

func TestUniqueObjectNames(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	dir := t.TempDir()
	writeASM(t, dir, "hello.asm", "")
	writeASM(t, dir, "hello.s", "")
	writeASM(t, dir, "sub/hello.asm", "")
	outBin := filepath.Join(t.TempDir(), "app")
	_, err := BuildDir(context.Background(), []string{dir}, outBin, false, true, "auto", true, false, true, true, false, nil)
	if err == nil {
		// link error expected because empty files have no _start/main
	}
	objDir := filepath.Join(filepath.Dir(outBin), ".fz_objs")
	entries, _ := os.ReadDir(objDir)
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	expected := []string{"hello_asm.o", "hello_s.o", "sub_hello_asm.o"}
	for _, exp := range expected {
		found := false
		for _, n := range names {
			if strings.Contains(n, exp) && strings.HasSuffix(n, ".o") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing object name containing %q", exp)
		}
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
