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

package builder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/config"
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
	dir := t.TempDir()
	writeASM(t, dir, "hello.asm", `section .text
global main
main:
    mov rax, 60
    xor rdi, rdi
    syscall
`)
	writeASM(t, dir, "hello.s", `.text
.globl hello_s
hello_s:
    ret
`)
	writeASM(t, dir, "sub/hello.asm", `section .text
global helper_func
helper_func:
    ret
`)
	outBin := filepath.Join(t.TempDir(), "app")
	_, err := BuildDir(context.Background(), []string{dir}, outBin, false, true, "raw", true, true, true, false, false, nil, nil, nil, nil, nil, 1, "executable")
	if err != nil {
		t.Fatal(err)
	}
	objDir := filepath.Join(filepath.Dir(outBin), ".fz_objs")
	entries, err := os.ReadDir(objDir)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
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

func TestMatchExcludePatterns(t *testing.T) {
	if !matchExclude("foo.o", []string{"*.o"}) {
		t.Fatal("expected foo.o to match exclude")
	}
	if !matchExclude("bar/test.s", []string{"bar/*"}) {
		t.Fatal("expected path to match exclude pattern")
	}
	if matchExclude("src/main.c", []string{"*.o"}) {
		t.Fatal("unexpected exclude match")
	}
}

func TestCollectSourceFilesUsesConfigSourceFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{SourceFiles: []string{"a.c", "b.c"}}
	files, err := CollectSourceFiles(cfg, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestCleanDirRemovesExecutablesAndObjects(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "app.out")
	if err := os.WriteFile(out, []byte("exe"), 0o755); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "a.o")
	if err := os.WriteFile(obj, []byte("obj"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CleanDir(dir, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("expected %s removed", out)
	}
	if _, err := os.Stat(obj); !os.IsNotExist(err) {
		t.Fatalf("expected %s removed", obj)
	}
}

func TestBuildDirNoSupportedFiles(t *testing.T) {
	dir := t.TempDir()
	outBin := filepath.Join(t.TempDir(), "app")
	_, err := BuildDir(context.Background(), []string{dir}, outBin, false, false, "auto", false, true, false, false, false, nil, nil, nil, nil, nil, 1, "executable")
	if err == nil || !strings.Contains(err.Error(), "no supported files found") {
		t.Fatalf("expected no supported files error, got %v", err)
	}
}

func TestRAMCacheStoreAndRestore(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.asm")
	if err := os.WriteFile(src, []byte("section .text\nglobal _start\n_start:\n  mov eax, 60\n  xor edi, edi\n  syscall\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "src.o")
	if err := os.WriteFile(obj, []byte{1, 2, 3, 4}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(obj+".syms", []byte("SYMS"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := storeRAMCache(src, obj, false, "auto"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj + ".syms"); err != nil {
		t.Fatal(err)
	}
	restored, err := restoreRAMCache(src, obj, false, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if !restored {
		t.Fatal("expected RAM cache restore")
	}
	got, err := os.ReadFile(obj)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte{1, 2, 3, 4}) {
		t.Fatalf("unexpected restored object: %v", got)
	}
	gotSyms, err := os.ReadFile(obj + ".syms")
	if err != nil {
		t.Fatal(err)
	}
	if string(gotSyms) != "SYMS" {
		t.Fatalf("unexpected restored syms: %q", string(gotSyms))
	}
}
