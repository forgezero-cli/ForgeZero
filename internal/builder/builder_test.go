package builder

import (
	"context"
	"fmt"
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
	t.Logf("Source dir: %s", dir)
	srcFile := writeASM(t, dir, "main.asm", `
section .text
global _start
_start:
	mov eax, 60
	xor edi, edi
	syscall
`)
	t.Logf("Source file: %s", srcFile)
	if _, err := os.Stat(srcFile); err != nil {
		t.Fatal("source file not created")
	}
	outBin := filepath.Join(t.TempDir(), "myapp")
	t.Logf("Output binary: %s", outBin)
	ctx := context.Background()
	res, err := BuildDir(ctx, []string{dir}, outBin, false, true, "raw", false, true, false, true, false, nil, nil, nil, nil, nil, 1, "executable")
	if err != nil {
		t.Fatalf("BuildDir error: %v", err)
	}
	if _, err := os.Stat(res.Binary); err != nil {
		t.Errorf("binary not created: %v", err)
	}
}

func TestBuildDirNoCache(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	dir := t.TempDir()
	t.Logf("Source dir: %s", dir)
	srcFile := writeASM(t, dir, "main.asm", `
section .text
global _start
_start:
	mov eax, 60
	xor edi, edi
	syscall
`)
	t.Logf("Source file: %s", srcFile)
	outBin := filepath.Join(t.TempDir(), "myapp")
	t.Logf("Output binary: %s", outBin)
	ctx := context.Background()
	res, err := BuildDir(ctx, []string{dir}, outBin, false, true, "raw", false, true, false, true, false, nil, nil, nil, nil, nil, 1, "executable")
	if err != nil {
		t.Fatalf("BuildDir error: %v", err)
	}
	if _, err := os.Stat(res.Binary); err != nil {
		t.Errorf("binary not created: %v", err)
	}
}

func TestUniqueObjectNames(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	dir := t.TempDir()
	t.Logf("Source dir: %s", dir)
	writeASM(t, dir, "hello.asm", "")
	writeASM(t, dir, "hello.s", "")
	writeASM(t, dir, "sub/hello.asm", "")
	outBin := filepath.Join(t.TempDir(), "app")
	t.Logf("Output binary: %s", outBin)
	_, _ = BuildDir(context.Background(), []string{dir}, outBin, false, true, "auto", true, true, true, true, false, nil, nil, nil, nil, nil, 1, "executable")
	objDir := filepath.Join(filepath.Dir(outBin), ".fz_objs")
	t.Logf("Object dir: %s", objDir)
	entries, err := os.ReadDir(objDir)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	t.Logf("Object files: %v", names)
	expectedPatterns := []string{"hello_asm", "hello_s", "sub_hello_asm"}
	for _, pattern := range expectedPatterns {
		found := false
		for _, n := range names {
			if strings.Contains(n, pattern) && strings.HasSuffix(n, ".o") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing object name containing %q", pattern)
		}
	}
}

func TestCleanDir(t *testing.T) {
	dir := t.TempDir()
	objDir := filepath.Join(dir, ".fz_objs")
	cacheDir := filepath.Join(dir, ".fz_cache")
	if err := os.MkdirAll(objDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) failed: %v", objDir, err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("failed to create cache directory: %v", err)
	}

	base := filepath.Base(dir)
	bin := filepath.Join(dir, base+".out")
	f, err := os.Create(bin)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	objFile := filepath.Join(dir, "test.o")
	f, err = os.Create(objFile)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
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

func TestThreeMainFiles(t *testing.T) {
	dir := t.TempDir()
	mainContent := `
#include <stdio.h>
void a(void);
void b(void);
void c(void);
int main() { a(); b(); c(); return 0; }
`
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b", "c"} {
		sub := filepath.Join(dir, name)
		if err := os.Mkdir(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		content := fmt.Sprintf(`
#include <stdio.h>
void %s(void) { printf("%%s\\n", __FILE__); }
`, name)
		if err := os.WriteFile(filepath.Join(sub, name+".c"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	outBin := filepath.Join(t.TempDir(), "three_main")
	ctx := context.Background()
	_, err := BuildDir(ctx, []string{dir}, outBin, false, true, "c", false, false, true, false, false, nil, nil, nil, nil, nil, 1, "executable")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outBin); err != nil {
		t.Error("binary not created")
	}
}
