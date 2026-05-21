package assembler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"fz/internal/utils"
	"fz/internal/zig"
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

func TestAsmCmdAndFormatFlagLogic(t *testing.T) {
	cases := []struct {
		target, wantCmd, wantFormat string
	}{
		{"x86_64-linux-gnu", "nasm", "-felf64"},
		{"i386-linux-gnu", "nasm", "-felf32"},
		{"arm-linux-gnueabihf", "arm-linux-gnueabihf-as", "-march=armv7-a"},
		{"riscv64-unknown-elf", "riscv64-unknown-elf-as", "-felf64"},
	}
	oldTarget := Target
	defer func() { Target = oldTarget }()
	for _, tt := range cases {
		Target = tt.target
		if got := asmCmdForTarget(); got != tt.wantCmd {
			t.Fatalf("asmCmdForTarget(%q) = %q, want %q", tt.target, got, tt.wantCmd)
		}
		if got := formatFlagForTarget(); got != tt.wantFormat {
			t.Fatalf("formatFlagForTarget(%q) = %q, want %q", tt.target, got, tt.wantFormat)
		}
	}
}

func TestAssembleUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.txt", "hello")
	obj := filepath.Join(dir, "test.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err == nil || !strings.Contains(err.Error(), "unsupported source extension") {
		t.Fatalf("expected unsupported extension error, got %v", err)
	}
}

func TestAssembleFASMInjectsELF64WithMockedCommand(t *testing.T) {
	oldForce := ForceFASM
	oldCheck := utils.CheckToolFunc
	oldRun := runCommand
	defer func() {
		ForceFASM = oldForce
		utils.CheckToolFunc = oldCheck
		runCommand = oldRun
	}()
	ForceFASM = true
	utils.CheckToolFunc = func(name string) error { return nil }
	invoked := false
	runCommand = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		invoked = true
		if name != "fasm" {
			t.Fatalf("expected fasm command, got %s", name)
		}
		return "", nil
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
	if !invoked {
		t.Fatal("expected fasm to be invoked")
	}
}

func TestAssembleCUsesZigWhenRequested(t *testing.T) {
	oldTarget := Target
	oldZigReq := zig.ZigRequested
	oldZigEnabled := zig.ZigEnabled
	oldCheck := utils.CheckToolFunc
	oldRun := zig.RunCommand
	defer func() {
		Target = oldTarget
		zig.ZigRequested = oldZigReq
		zig.ZigEnabled = oldZigEnabled
		utils.CheckToolFunc = oldCheck
		zig.RunCommand = oldRun
	}()
	Target = "x86_64-linux-gnu"
	zig.ZigRequested = true
	utils.CheckToolFunc = func(name string) error { return nil }
	called := false
	zig.RunCommand = func(ctx context.Context, verbose bool, args ...string) (string, error) {
		called = true
		return "", nil
	}
	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.c", "int main() { return 0; }")
	obj := filepath.Join(dir, "test.o")
	if err := Assemble(context.Background(), src, obj, false, false, "auto"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected zig compile to run")
	}
}
