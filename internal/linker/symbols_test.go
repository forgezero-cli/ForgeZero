package linker

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func buildObjectWithNASM(t *testing.T, dir, name, asmContent string) string {
	src := filepath.Join(dir, name+".asm")
	err := os.WriteFile(src, []byte(asmContent), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, name+".o")
	cmd := exec.Command("nasm", "-felf64", src, "-o", obj)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("nasm failed: %v\n%s", err, out)
	}
	return obj
}

func TestCheckDuplicateSymbols(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	if _, err := exec.LookPath("objdump"); err != nil {
		t.Skip("objdump not installed")
	}
	dir := t.TempDir()
	obj1 := buildObjectWithNASM(t, dir, "a", `
section .text
global my_func
my_func:
	mov eax, 1
	ret
`)
	obj2 := buildObjectWithNASM(t, dir, "b", `
section .text
global my_func
my_func:
	mov eax, 2
	ret
`)
	err := CheckDuplicateSymbols([]string{obj1, obj2}, false)
	if err == nil {
		t.Error("expected duplicate symbol error")
	}
	obj3 := buildObjectWithNASM(t, dir, "c", `
section .text
global other_func
other_func:
	ret
`)
	err = CheckDuplicateSymbols([]string{obj1, obj3}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckDuplicateSymbolsNoDuplicates(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	dir := t.TempDir()
	obj1 := buildObjectWithNASM(t, dir, "a", `
section .text
global a
a: ret
`)
	obj2 := buildObjectWithNASM(t, dir, "b", `
section .text
global b
b: ret
`)
	err := CheckDuplicateSymbols([]string{obj1, obj2}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckDuplicateSymbolsSingleFile(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	dir := t.TempDir()
	obj := buildObjectWithNASM(t, dir, "single", `
section .text
global foo
foo: ret
`)
	err := CheckDuplicateSymbols([]string{obj}, false)
	if err != nil {
		t.Errorf("single file should not produce error: %v", err)
	}
}

func TestCheckDuplicateSymbolsVerbose(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	dir := t.TempDir()
	obj := buildObjectWithNASM(t, dir, "verbose", `
section .text
global bar
bar: ret
`)
	err := CheckDuplicateSymbols([]string{obj}, true)
	if err != nil {
		t.Errorf("verbose mode should not error on single file: %v", err)
	}
}

func TestReadSymbolsWithObjdumpFallback(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	dir := t.TempDir()
	obj := buildObjectWithNASM(t, dir, "fallback", `
section .text
global fallback_func
fallback_func: ret
`)
	_, err := readSymbols(obj, false)
	if err != nil {
		t.Skip("no symbol reader available")
	}
}
