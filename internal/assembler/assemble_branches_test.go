package assembler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fz/internal/utils"
	"fz/internal/zig"
)

func TestValidateArgsReject(t *testing.T) {
	if err := validateArgs([]string{"bad;arg"}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateArgsOK(t *testing.T) {
	if err := validateArgs([]string{"-O2"}); err != nil {
		t.Fatal(err)
	}
}

func TestCCForTargetAndGasCmd(t *testing.T) {
	old := Target
	defer func() { Target = old }()
	Target = "x86_64-linux-gnu"
	if CCForTarget() != "gcc" {
		t.Fatal(CCForTarget())
	}
	if gasCmdForTarget() != "as" {
		t.Fatal(gasCmdForTarget())
	}
	Target = "wasm32-unknown-unknown"
	if CCForTarget() != "clang" {
		t.Fatal(CCForTarget())
	}
	if gasCmdForTarget() != "clang" {
		t.Fatal(gasCmdForTarget())
	}
	Target = "arm-linux-gnueabihf"
	if CCForTarget() != "arm-linux-gnueabihf-gcc" {
		t.Fatal(CCForTarget())
	}
	Target = "riscv64-unknown-elf"
	if cxxForTarget() != "riscv64-unknown-elf-g++" {
		t.Fatal(cxxForTarget())
	}
}

func TestAssembleInvalidPaths(t *testing.T) {
	err := Assemble(context.Background(), "../escape", "out.o", false, false, "auto")
	if err == nil {
		t.Fatal("expected path error")
	}
}

func TestAssembleWasmNASM(t *testing.T) {
	old := Target
	defer func() { Target = old }()
	Target = "wasm32-unknown-unknown"
	dir := t.TempDir()
	src := filepath.Join(dir, "t.asm")
	os.WriteFile(src, []byte("nop"), 0o644)
	obj := filepath.Join(dir, "t.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err == nil || !strings.Contains(err.Error(), "wasm") {
		t.Fatalf("got %v", err)
	}
}

func TestAssembleWasmFASM(t *testing.T) {
	old := Target
	defer func() { Target = old }()
	Target = "wasm32-unknown-unknown"
	dir := t.TempDir()
	src := filepath.Join(dir, "t.fasm")
	os.WriteFile(src, []byte("nop"), 0o644)
	err := Assemble(context.Background(), src, filepath.Join(dir, "t.o"), false, false, "auto")
	if err == nil || !strings.Contains(err.Error(), "wasm") {
		t.Fatalf("got %v", err)
	}
}

func TestAssembleNASMVerboseFail(t *testing.T) {
	oldRun := runCommand
	oldCheck := utils.CheckToolFunc
	defer func() { runCommand = oldRun; utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(string) error { return nil }
	runCommand = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "line 1\nerror: bad\n", errors.New("fail")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "t.asm")
	os.WriteFile(src, []byte("nop"), 0o644)
	err := Assemble(context.Background(), src, filepath.Join(dir, "t.o"), false, true, "auto")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssembleNASMNonVerboseFail(t *testing.T) {
	oldRun := runCommand
	oldCheck := utils.CheckToolFunc
	defer func() { runCommand = oldRun; utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(string) error { return nil }
	runCommand = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "", errors.New("fail")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "t.asm")
	os.WriteFile(src, []byte("nop"), 0o644)
	err := Assemble(context.Background(), src, filepath.Join(dir, "t.o"), false, false, "auto")
	if err == nil || !strings.Contains(err.Error(), "verbose") {
		t.Fatalf("got %v", err)
	}
}

func TestAssembleNASMInvalidAsmFlags(t *testing.T) {
	oldFlags := AsmFlags
	oldCheck := utils.CheckToolFunc
	defer func() { AsmFlags = oldFlags; utils.CheckToolFunc = oldCheck }()
	AsmFlags = []string{"bad;flag"}
	utils.CheckToolFunc = func(string) error { return nil }
	dir := t.TempDir()
	src := filepath.Join(dir, "t.asm")
	os.WriteFile(src, []byte("nop"), 0o644)
	err := Assemble(context.Background(), src, filepath.Join(dir, "t.o"), false, false, "auto")
	if err == nil {
		t.Fatal("expected asm flags error")
	}
}

func TestAssembleCMocked(t *testing.T) {
	oldRun := runCommand
	oldCheck := utils.CheckToolFunc
	oldCc := CcFlags
	defer func() {
		runCommand = oldRun
		utils.CheckToolFunc = oldCheck
		CcFlags = oldCc
	}()
	utils.CheckToolFunc = func(string) error { return nil }
	runCommand = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "", nil
	}
	CcFlags = "-O2"
	dir := t.TempDir()
	src := filepath.Join(dir, "t.c")
	os.WriteFile(src, []byte("int x;"), 0o644)
	obj := filepath.Join(dir, "t.o")
	if err := Assemble(context.Background(), src, obj, true, true, "auto"); err != nil {
		t.Fatal(err)
	}
}

func TestAssembleCMockedFail(t *testing.T) {
	oldRun := runCommand
	oldCheck := utils.CheckToolFunc
	defer func() { runCommand = oldRun; utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(string) error { return nil }
	runCommand = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "err", errors.New("compile fail")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "t.c")
	os.WriteFile(src, []byte("int x;"), 0o644)
	err := Assemble(context.Background(), src, filepath.Join(dir, "t.o"), false, false, "auto")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssembleCppInvalidCcFlags(t *testing.T) {
	oldCc := CcFlags
	oldCheck := utils.CheckToolFunc
	defer func() { CcFlags = oldCc; utils.CheckToolFunc = oldCheck }()
	CcFlags = "bad;flag"
	utils.CheckToolFunc = func(string) error { return nil }
	dir := t.TempDir()
	src := filepath.Join(dir, "t.cpp")
	os.WriteFile(src, []byte("int x;"), 0o644)
	err := Assemble(context.Background(), src, filepath.Join(dir, "t.o"), false, false, "auto")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssembleWasmClangC(t *testing.T) {
	oldTarget := Target
	oldRun := runCommand
	oldCheck := utils.CheckToolFunc
	defer func() {
		Target = oldTarget
		runCommand = oldRun
		utils.CheckToolFunc = oldCheck
	}()
	Target = "wasm32-unknown-unknown"
	utils.CheckToolFunc = func(string) error { return nil }
	runCommand = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "", nil
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "t.c")
	os.WriteFile(src, []byte("int main(){}"), 0o644)
	if err := Assemble(context.Background(), src, filepath.Join(dir, "t.o"), false, true, "auto"); err != nil {
		t.Fatal(err)
	}
}

func TestAssembleForceFASMOnAsm(t *testing.T) {
	oldForce := ForceFASM
	oldRun := runCommand
	oldCheck := utils.CheckToolFunc
	defer func() {
		ForceFASM = oldForce
		runCommand = oldRun
		utils.CheckToolFunc = oldCheck
	}()
	ForceFASM = true
	utils.CheckToolFunc = func(string) error { return nil }
	runCommand = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "", nil
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "t.asm")
	os.WriteFile(src, []byte("nop"), 0o644)
	if err := Assemble(context.Background(), src, filepath.Join(dir, "t.o"), false, false, "auto"); err != nil {
		t.Fatal(err)
	}
}

func TestAssembleFASMNonVerboseFail(t *testing.T) {
	oldRun := runCommand
	oldCheck := utils.CheckToolFunc
	defer func() { runCommand = oldRun; utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(string) error { return nil }
	runCommand = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "", errors.New("fasm fail")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "t.fasm")
	os.WriteFile(src, []byte("format ELF64\n_start:\n"), 0o644)
	err := Assemble(context.Background(), src, filepath.Join(dir, "t.o"), false, false, "auto")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssembleFASMVerboseParseErrorLine(t *testing.T) {
	oldRun := runCommand
	oldCheck := utils.CheckToolFunc
	defer func() { runCommand = oldRun; utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(string) error { return nil }
	runCommand = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "fatal error on line 5\n", errors.New("fasm fail")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "t.fasm")
	os.WriteFile(src, []byte("format ELF64\n_start:\n"), 0o644)
	err := Assemble(context.Background(), src, filepath.Join(dir, "t.o"), false, true, "auto")
	if err == nil || !strings.Contains(err.Error(), "fasm error") {
		t.Fatalf("got %v", err)
	}
}

func TestAssembleCppZigEnabled(t *testing.T) {
	oldZig := zig.ZigEnabled
	oldRun := zig.RunCommand
	oldCheck := utils.CheckToolFunc
	defer func() {
		zig.ZigEnabled = oldZig
		zig.RunCommand = oldRun
		utils.CheckToolFunc = oldCheck
	}()
	zig.ZigEnabled = true
	utils.CheckToolFunc = func(string) error { return nil }
	zig.RunCommand = func(ctx context.Context, verbose bool, args ...string) (string, error) {
		return "", nil
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "t.cpp")
	os.WriteFile(src, []byte("int main(){}"), 0o644)
	if err := Assemble(context.Background(), src, filepath.Join(dir, "t.o"), false, false, "auto"); err != nil {
		t.Fatal(err)
	}
}

func TestAssembleGASWasm(t *testing.T) {
	oldTarget := Target
	oldRun := runCommand
	oldCheck := utils.CheckToolFunc
	defer func() {
		Target = oldTarget
		runCommand = oldRun
		utils.CheckToolFunc = oldCheck
	}()
	Target = "wasm32-unknown-unknown"
	utils.CheckToolFunc = func(string) error { return nil }
	runCommand = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "", nil
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "t.s")
	os.WriteFile(src, []byte(".globl _start\n_start:\n"), 0o644)
	if err := Assemble(context.Background(), src, filepath.Join(dir, "t.o"), true, true, "auto"); err != nil {
		t.Fatal(err)
	}
}

func TestFormatFlagDefaultArch(t *testing.T) {
	old := Target
	defer func() { Target = old }()
	Target = "powerpc-unknown"
	if formatFlagForTarget() != "-felf64" {
		t.Fatal(formatFlagForTarget())
	}
}

func TestAsmCmdWasm(t *testing.T) {
	old := Target
	defer func() { Target = old }()
	Target = "wasm32-unknown-unknown"
	if asmCmdForTarget() != "clang" {
		t.Fatal(asmCmdForTarget())
	}
}
