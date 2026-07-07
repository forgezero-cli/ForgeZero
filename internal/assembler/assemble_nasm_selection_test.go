package assembler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type runCmdCapture struct {
	nasmCalled bool
}

func TestAsm_DefaultUsesInternalAssembler_NoNasm(t *testing.T) {
	oldForceInternal := ForceInternalAsm
	oldUseNasm := UseNasm
	defer func() {
		ForceInternalAsm = oldForceInternal
		UseNasm = oldUseNasm
	}()

	ForceInternalAsm = true
	UseNasm = false

	cap := &runCmdCapture{}
	oldRun := runCommand
	defer func() { runCommand = oldRun }()
	SetRunCommand(func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if strings.Contains(name, "nasm") {
			cap.nasmCalled = true
		}
		return "", nil
	})

	dir := t.TempDir()
	src := filepath.Join(dir, "t.asm")
	if err := os.WriteFile(src, []byte("section .text\nstart:\n  nop\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "t.o")

	_ = Assemble(context.Background(), src, obj, false, false, "raw")

	if cap.nasmCalled {
		t.Fatal("nasm was invoked, expected internal assembler by default")
	}
}

func TestAsm_UseNasmRoutesToNasm(t *testing.T) {
	oldForceInternal := ForceInternalAsm
	oldUseNasm := UseNasm
	defer func() {
		ForceInternalAsm = oldForceInternal
		UseNasm = oldUseNasm
	}()

	ForceInternalAsm = false
	UseNasm = true

	cap := &runCmdCapture{}
	oldRun := runCommand
	defer func() { runCommand = oldRun }()
	SetRunCommand(func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if strings.Contains(name, "nasm") {
			cap.nasmCalled = true
		}
		return "", nil
	})

	dir := t.TempDir()
	src := filepath.Join(dir, "t.asm")
	if err := os.WriteFile(src, []byte("section .text\nstart:\n  nop\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "t.o")

	_ = Assemble(context.Background(), src, obj, false, false, "raw")

	if !cap.nasmCalled {
		t.Fatal("nasm was not invoked, expected NASM routing when UseNasm=true")
	}
}

