package assembler

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAssembleNASM(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.asm", `
section .text
global _start
_start:
	mov eax, 1
	xor ebx, ebx
	int 0x80
`)
	obj := filepath.Join(dir, "test.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(obj); err != nil {
		t.Error("object file not created")
	}
	err = Assemble(context.Background(), src, filepath.Join(dir, "test_dbg.o"), true, false, "auto")
	if err != nil {
		t.Fatal(err)
	}
}

func TestAssembleGAS(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.s", `
.globl _start
_start:
	mov $1, %eax
	xor %ebx, %ebx
	int $0x80
`)
	obj := filepath.Join(dir, "test.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(obj); err != nil {
		t.Error("object file not created")
	}
}

func TestAssembleFASM(t *testing.T) {
	if _, err := exec.LookPath("fasm"); err != nil {
		t.Skip("fasm not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.fasm", `
format ELF64
section '.text' executable
public _start
_start:
	mov eax, 1
	xor ebx, ebx
	int 0x80
`)
	obj := filepath.Join(dir, "test.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(obj); err != nil {
		t.Error("object file not created")
	}
}

func TestAssembleUnsupported(t *testing.T) {
	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.c", "int main(){}")
	obj := filepath.Join(dir, "test.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err == nil {
		t.Error("expected error for unsupported extension")
	}
}
