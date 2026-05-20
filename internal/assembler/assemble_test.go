package assembler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestAssembleNASMFailure(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "bad.asm", "invalid asm content")
	obj := filepath.Join(dir, "bad.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err == nil {
		t.Error("expected error for invalid asm")
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

func TestAssembleGASFailure(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "bad.s", "invalid asm content")
	obj := filepath.Join(dir, "bad.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err == nil {
		t.Error("expected error for invalid asm")
	}
}

func TestAssembleFASMFailure(t *testing.T) {
	if _, err := exec.LookPath("fasm"); err != nil {
		t.Skip("fasm not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "bad.fasm", "invalid fasm")
	obj := filepath.Join(dir, "bad.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err == nil {
		t.Error("expected error for invalid fasm")
	}
}

func TestAssembleC(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.c", "int main() { return 0; }")
	obj := filepath.Join(dir, "test.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(obj); err != nil {
		t.Error("object file not created")
	}
}

func TestAssembleCFailure(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "bad.c", "int main() { return ")
	obj := filepath.Join(dir, "bad.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err == nil {
		t.Error("expected error for invalid C code")
	}
}

func TestAssembleCpp(t *testing.T) {
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("g++ not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.cpp", `
#include <cstdio>
int main() {
    printf("C++ works\n");
    return 0;
}
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

func TestAssembleCppFailure(t *testing.T) {
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("g++ not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "bad.cpp", "invalid c++")
	obj := filepath.Join(dir, "bad.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err == nil {
		t.Error("expected error for invalid C++")
	}
}

func TestCrossCompilerNotFound(t *testing.T) {
	oldTarget := Target
	defer func() { Target = oldTarget }()
	Target = "arm-linux-gnueabihf"
	if _, err := exec.LookPath("arm-linux-gnueabihf-gcc"); err == nil {
		t.Skip("cross-compiler present, cannot test missing tool")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "test.c")
	if err := os.WriteFile(src, []byte("int main(){}"), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "test.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err == nil {
		t.Error("expected error because cross-compiler missing")
		return
	}
	if !strings.Contains(err.Error(), "arm-linux-gnueabihf-gcc") {
		t.Errorf("error should mention missing compiler: %v", err)
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
    mov eax, 60
    xor edi, edi
    syscall
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

func TestAssembleFASMDebug(t *testing.T) {
	if _, err := exec.LookPath("fasm"); err != nil {
		t.Skip("fasm not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.fasm", `
format ELF64
section '.text' executable
public _start
_start:
    mov eax, 60
    xor edi, edi
    syscall
`)
	obj := filepath.Join(dir, "test.o")
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
		w.Close()
		r.Close()
	}()

	err = Assemble(context.Background(), src, obj, true, true, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "-dDEBUG") {
		t.Error("debug flag -dDEBUG not added to fasm command")
	}
}

func TestAssembleFASMError(t *testing.T) {
	if _, err := exec.LookPath("fasm"); err != nil {
		t.Skip("fasm not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "bad.fasm", `
format ELF64
section '.text' executable
public _start
_start:
    invalid_instruction
`)
	obj := filepath.Join(dir, "bad.o")
	err := Assemble(context.Background(), src, obj, false, true, "auto")
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "error") {
		t.Errorf("error message should contain 'error', got: %v", err)
	}
}

func TestAssembleFASMDebugWarning(t *testing.T) {
	if _, err := exec.LookPath("fasm"); err != nil {
		t.Skip("fasm not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.fasm", "format ELF64\nsection '.text' executable\n_start:\n mov eax,60\n xor edi,edi\n syscall\n")
	obj := filepath.Join(dir, "test.o")
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
		w.Close()
		r.Close()
	}()

	err = Assemble(context.Background(), src, obj, true, true, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "note: FASM debug flag") {
		t.Error("expected debug note on stderr")
	}
}

func TestAssembleFASMInjectsFormat(t *testing.T) {
	if _, err := exec.LookPath("fasm"); err != nil {
		t.Skip("fasm not installed")
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.fasm", `
section '.text' executable
public _start
_start:
    mov eax, 60
    xor edi, edi
    syscall
`)
	obj := filepath.Join(dir, "test.o")
	if err := Assemble(context.Background(), src, obj, false, false, "auto"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(obj); err != nil {
		t.Fatal(err)
	}
}
