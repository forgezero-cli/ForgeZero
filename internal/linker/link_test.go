package linker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"fz/internal/assembler"
	"fz/internal/utils"
)

type MockRunner struct {
	RunFunc func(ctx context.Context, verbose bool, name string, args ...string) (string, error)
}

func (m *MockRunner) Run(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, verbose, name, args...)
	}
	return "", nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func buildObject(t *testing.T, dir, name, asmContent string) string {
	src := filepath.Join(dir, name+".s")
	err := os.WriteFile(src, []byte(asmContent), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, name+".o")
	cmd := exec.Command("gcc", "-c", src, "-o", obj)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("gcc -c failed: %v\n%s", err, out)
	}
	return obj
}

func buildNasmObject(t *testing.T, dir, name, asmContent string) string {
	src := filepath.Join(dir, name+".s")
	err := os.WriteFile(src, []byte(asmContent), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, name+".o")
	cmd := exec.Command("nasm", "-f", "elf64", src, "-o", obj)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("nasm failed: %v\n%s", err, out)
	}
	return obj
}

func TestLink(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	obj := buildObject(t, dir, "test", `
.globl _start
_start:
	mov $60, %eax
	xor %edi, %edi
	syscall
`)
	bin := filepath.Join(dir, "test")
	err := Link(context.Background(), obj, bin, false, "raw", false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created")
	}
}

func TestApplyGccLdFlags(t *testing.T) {
	args := []string{"test.o", "-o", "bin"}
	got := ApplyGccLdFlags(args, "script.ld", "0x1000")
	expected := []string{"test.o", "-o", "bin", "-Wl,-T,script.ld", "-Wl,-Ttext=0x1000", "-Wl,--build-id=none"}
	if !equalSlices(got, expected) {
		t.Errorf("ApplyGccLdFlags = %v, want %v", got, expected)
	}
	got = ApplyGccLdFlags(args, "", "")
	expectedEmpty := []string{"test.o", "-o", "bin", "-Wl,--build-id=none"}
	if !equalSlices(got, expectedEmpty) {
		t.Errorf("should add deterministic build flag when empty, got %v", got)
	}
}

func TestApplyLdFlags(t *testing.T) {
	args := []string{"test.o", "-o", "bin"}
	got := ApplyLdFlags(args, "script.ld", "0x1000")
	expected := []string{"test.o", "-o", "bin", "-T", "script.ld", "-Ttext", "0x1000", "--build-id=none"}
	if !equalSlices(got, expected) {
		t.Errorf("ApplyLdFlags = %v, want %v", got, expected)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLinkWithGccMockSuccess(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", nil
		},
	}
	ctx := context.Background()
	err := linkWithGcc(ctx, "obj.o", "bin", false, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLinkWithGccMockSanitize(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	ctx := context.Background()
	err := linkWithGcc(ctx, "obj.o", "bin", false, false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-fsanitize=address") || !contains(capturedArgs, "-fsanitize=undefined") {
		t.Error("sanitizer flags missing")
	}
	if !contains(capturedArgs, "-fsanitize-address-use-after-scope") {
		t.Error("strict flag missing")
	}
}

func TestTryAutoLinkNoClang(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	oldCheck := utils.CheckToolFunc
	defer func() { utils.CheckToolFunc = oldCheck }()

	utils.CheckToolFunc = func(name string) error {
		if name == "gcc" {
			return nil
		}
		return fmt.Errorf("not found")
	}
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			if name == "gcc" {
				return "", nil
			}
			return "", fmt.Errorf("unexpected")
		},
	}
	ctx := context.Background()
	err := tryAutoLink(ctx, "obj.o", "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLinkMultipleWithGccMock(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	objFiles := []string{"a.o", "b.o"}
	libs := []string{"m"}
	err := linkMultipleWithGcc(context.Background(), objFiles, "bin", false, false, true, true, libs)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-fsanitize=address") {
		t.Error("missing -fsanitize=address")
	}
	if !contains(capturedArgs, "-fsanitize-address-use-after-scope") {
		t.Error("missing -fsanitize-address-use-after-scope (strict)")
	}
	for _, lib := range libs {
		if !contains(capturedArgs, "-l"+lib) {
			t.Errorf("missing -l%s", lib)
		}
	}
}

func TestLinkMultipleWithGccNoFallback(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("gcc error")
		},
	}
	objFiles := []string{"a.o"}
	err := linkMultipleWithGcc(context.Background(), objFiles, "bin", false, false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

func TestLinkMultipleWithClangMock(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	objFiles := []string{"a.o"}
	libs := []string{"c"}
	err := linkMultipleWithClang(context.Background(), objFiles, "bin", false, true, true, libs)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-fsanitize=address") {
		t.Error("missing sanitizer flags")
	}
	if !contains(capturedArgs, "-l"+libs[0]) {
		t.Error("missing library flag")
	}
}

func TestLinkMultipleWithClangNoFallback(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("clang error")
		},
	}
	objFiles := []string{"a.o"}
	err := linkMultipleWithClang(context.Background(), objFiles, "bin", false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

func TestLinkMultipleWithLdMock(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	objFiles := []string{"a.o", "b.o"}
	libs := []string{"m"}
	err := linkMultipleWithLd(context.Background(), objFiles, "bin", false, libs)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-o") {
		t.Error("missing -o flag")
	}
	for _, lib := range libs {
		if !contains(capturedArgs, "-l"+lib) {
			t.Errorf("missing -l%s", lib)
		}
	}
}

func TestTryAutoLinkMultipleWithClangSuccess(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	clangCalled := false
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			if name == "clang" {
				clangCalled = true
				return "", nil
			}
			return "", nil
		},
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed, cannot test strict mode branch")
	}
	objFiles := []string{"a.o"}
	err := tryAutoLinkMultiple(context.Background(), objFiles, "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !clangCalled {
		t.Error("clang not called in strict mode")
	}
}

func TestTryAutoLinkMultipleClangFailFallbackToGcc(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if name == "clang" {
				return "", fmt.Errorf("clang fails")
			}
			if name == "gcc" {
				return "", nil
			}
			return "", nil
		},
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed, cannot test clang failure path")
	}
	objFiles := []string{"a.o"}
	err := tryAutoLinkMultiple(context.Background(), objFiles, "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestLinkWithGccFallbackToNoPie(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first gcc fails")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	err := linkWithGcc(context.Background(), "obj", "bin", false, true, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkWithClangFallbackToNoPie(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first clang fails")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	err := linkWithClang(context.Background(), "obj", "bin", false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkWithGccVerboseError(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("gcc error")
		},
	}
	err := linkWithGcc(context.Background(), "obj", "bin", true, false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "gcc error") {
		t.Errorf("error should contain command output: %v", err)
	}
}

func TestLinkWithGccFallbackNoPieVerbose(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first fail")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	err := linkWithGcc(context.Background(), "obj", "bin", true, true, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestTryAutoLinkMultipleNoLinker(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	objFiles := []string{"a.o"}
	err := tryAutoLinkMultiple(context.Background(), objFiles, "bin", false, false, false, nil)
	if err == nil {
		t.Error("expected error when no linker")
	}
	if err.Error() != "auto linking failed: no suitable linker" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLinkMultipleWithGccFallbackNoPie(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first fail")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	objFiles := []string{"a.o"}
	err := linkMultipleWithGcc(context.Background(), objFiles, "bin", false, true, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkWithLdMissingLib(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	libs := []string{"m", "c"}
	err := linkWithLd(context.Background(), "obj.o", "bin", false, libs)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-lm") || !contains(capturedArgs, "-lc") {
		t.Error("library flags missing")
	}
}

func TestValidateLinkCallErrors(t *testing.T) {
	if err := validateLinkCall(nil, "out"); err == nil {
		t.Error("expected invalid linking context error")
	}
	if err := validateLinkCall(context.Background(), ""); err == nil {
		t.Error("expected output file name required error")
	}
}

func TestShouldUseResponseFile(t *testing.T) {
	args := make([]string, 200)
	for i := range args {
		args[i] = "arg"
	}
	if !shouldUseResponseFile(args) {
		t.Fatal("expected response file for many args")
	}
}

func TestCreateResponseFileWritesArgs(t *testing.T) {
	path, err := createResponseFile([]string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "a") || !strings.Contains(string(data), "b") {
		t.Fatal("response file content missing args")
	}
}

func TestRunLinkerCommandNoName(t *testing.T) {
	if _, err := runLinkerCommand(context.Background(), false, "", []string{"a"}); err == nil {
		t.Fatal("expected error when linker name is empty")
	}
}

func TestLinkMultipleWithClangNoFallbackError(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("clang error")
		},
	}
	objFiles := []string{"a.o"}
	err := linkMultipleWithClang(context.Background(), objFiles, "bin", false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

func TestTryAutoLinkMultipleStrictClangSuccess(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	clangCalled := false
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			if name == "clang" {
				clangCalled = true
				return "", nil
			}
			return "", nil
		},
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed, cannot test strict mode branch")
	}
	objFiles := []string{"a.o"}
	err := tryAutoLinkMultiple(context.Background(), objFiles, "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !clangCalled {
		t.Error("clang not called in strict mode")
	}
}

func TestTryAutoLinkMultipleStrictClangFailThenGcc(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if name == "clang" {
				return "", fmt.Errorf("clang fails")
			}
			if name == "gcc" {
				return "", nil
			}
			return "", nil
		},
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed, cannot test clang failure path")
	}
	objFiles := []string{"a.o"}
	err := tryAutoLinkMultiple(context.Background(), objFiles, "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestLinkWithGccNoFallbackError(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("gcc error")
		},
	}
	err := linkWithGcc(context.Background(), "obj", "bin", false, false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

func TestLinkWithGccFallbackBothFail(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			return "", fmt.Errorf("always fails")
		},
	}
	err := linkWithGcc(context.Background(), "obj", "bin", false, true, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestTryAutoLinkStrictClangSuccess(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	clangCalled := false
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			if name == "clang" {
				clangCalled = true
				return "", nil
			}
			return "", nil
		},
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed, cannot test strict clang path")
	}
	err := tryAutoLink(context.Background(), "obj.o", "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !clangCalled {
		t.Error("clang not called in strict mode")
	}
}

func TestTryAutoLinkStrictClangFailGccSuccess(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if name == "clang" {
				return "", fmt.Errorf("clang error")
			}
			if name == "gcc" {
				return "", nil
			}
			return "", nil
		},
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed, cannot test clang failure path")
	}
	err := tryAutoLink(context.Background(), "obj.o", "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestLinkWithClangNoFallbackError(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("clang error")
		},
	}
	err := linkWithClang(context.Background(), "obj.o", "bin", false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

func TestLinkMultipleWithClangFallbackToNoPie(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first fail")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	objFiles := []string{"a.o"}
	err := linkMultipleWithClang(context.Background(), objFiles, "bin", false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestCheckDuplicateSymbolsVerboseReal(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	if _, err := exec.LookPath("objdump"); err != nil {
		t.Skip("objdump not installed")
	}
	dir := t.TempDir()
	buildNasmObject(t, dir, "dup1", `
section .text
global dup
dup: ret
`)
	buildNasmObject(t, dir, "dup2", `
section .text
global dup
dup: ret
`)
	obj1 := filepath.Join(dir, "dup1.o")
	obj2 := filepath.Join(dir, "dup2.o")
	err := CheckDuplicateSymbols([]string{obj1, obj2}, true)
	if err == nil {
		t.Error("expected duplicate symbol error")
	}
	if !strings.Contains(err.Error(), "duplicate global symbols") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestTryAutoLinkMultipleStrictClangFailGccSuccess(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if name == "clang" {
				return "", fmt.Errorf("clang error")
			}
			if name == "gcc" {
				return "", nil
			}
			return "", nil
		},
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed")
	}
	objFiles := []string{"a.o"}
	err := tryAutoLinkMultiple(context.Background(), objFiles, "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestLinkMultipleWithGccSanitizeStrict(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	objFiles := []string{"a.o"}
	err := linkMultipleWithGcc(context.Background(), objFiles, "bin", false, false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-fsanitize=address") {
		t.Error("missing -fsanitize=address")
	}
	if !contains(capturedArgs, "-fsanitize-address-use-after-scope") {
		t.Error("missing strict flag")
	}
}

func TestLinkMultipleWithGccNoFallbackError(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("gcc error")
		},
	}
	objFiles := []string{"a.o"}
	err := linkMultipleWithGcc(context.Background(), objFiles, "bin", false, false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

func TestSetOutputFormat(t *testing.T) {
	oldFormat := assembler.OutputFormat
	defer func() { assembler.OutputFormat = oldFormat }()

	err := SetOutputFormat("elf64")
	if err != nil {
		t.Fatal(err)
	}
	if assembler.OutputFormat != "elf64" {
		t.Errorf("expected elf64, got %s", assembler.OutputFormat)
	}

	err = SetOutputFormat("elf32")
	if err != nil {
		t.Fatal(err)
	}
	if assembler.OutputFormat != "elf32" {
		t.Errorf("expected elf32, got %s", assembler.OutputFormat)
	}

	err = SetOutputFormat("bin")
	if err != nil {
		t.Fatal(err)
	}
	if assembler.OutputFormat != "bin" {
		t.Errorf("expected bin, got %s", assembler.OutputFormat)
	}

	err = SetOutputFormat("invalid")
	if err == nil {
		t.Error("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid output format") {
		t.Errorf("wrong error message: %v", err)
	}
}

func TestLinkGccNotFound(t *testing.T) {
	dir := t.TempDir()
	obj := filepath.Join(dir, "obj.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "bin")

	oldCheck := utils.CheckToolFunc
	defer func() { utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(name string) error {
		if name == "gcc" {
			return fmt.Errorf("gcc not found")
		}
		return nil
	}
	err := Link(context.Background(), obj, bin, false, "c", false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "gcc not found") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestLinkLdNotFound(t *testing.T) {
	dir := t.TempDir()
	obj := filepath.Join(dir, "obj.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "bin")

	oldCheck := utils.CheckToolFunc
	defer func() { utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(name string) error {
		if name == "ld" {
			return fmt.Errorf("ld not found")
		}
		return nil
	}
	err := Link(context.Background(), obj, bin, false, "raw", false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "ld not found") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestLinkMultipleGccNotFound(t *testing.T) {
	oldCheck := utils.CheckToolFunc
	defer func() { utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(name string) error {
		if name == "gcc" {
			return fmt.Errorf("gcc not found")
		}
		return nil
	}
	dir := t.TempDir()
	obj := filepath.Join(dir, "a.o")
	err := os.WriteFile(obj, []byte("fake"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	err = LinkMultiple(context.Background(), []string{obj}, "bin", false, "c", false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "gcc not found") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestLinkMultipleLdNotFound(t *testing.T) {
	oldCheck := utils.CheckToolFunc
	defer func() { utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(name string) error {
		if name == "ld" {
			return fmt.Errorf("ld not found")
		}
		return nil
	}

	dir := t.TempDir()
	obj := filepath.Join(dir, "a.o")
	err := os.WriteFile(obj, []byte("fake"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	err = LinkMultiple(context.Background(), []string{obj}, "bin", false, "raw", false, false, false, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "ld not found") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestTryAutoLinkNoGccNoLd(t *testing.T) {
	oldCheck := utils.CheckToolFunc
	defer func() { utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(name string) error {
		return fmt.Errorf("tool not found")
	}
	err := tryAutoLink(context.Background(), "obj.o", "bin", false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "auto linking failed") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestTryAutoLinkMultipleNoGccNoLd(t *testing.T) {
	oldCheck := utils.CheckToolFunc
	defer func() { utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(name string) error {
		return fmt.Errorf("tool not found")
	}
	objFiles := []string{"a.o"}
	err := tryAutoLinkMultiple(context.Background(), objFiles, "bin", false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "auto linking failed") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestLinkMultipleWithGccStrictSanitize(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	objFiles := []string{"a.o", "b.o"}
	libs := []string{"m"}
	err := linkMultipleWithGcc(context.Background(), objFiles, "bin", false, true, true, true, libs)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-fsanitize=address") || !contains(capturedArgs, "-fsanitize=undefined") {
		t.Error("sanitizer flags missing")
	}
	if !contains(capturedArgs, "-fsanitize-address-use-after-scope") {
		t.Error("strict flag missing")
	}
	for _, lib := range libs {
		if !contains(capturedArgs, "-l"+lib) {
			t.Errorf("missing -l%s", lib)
		}
	}
}

func TestLinkMultipleWithClangStrict(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	objFiles := []string{"a.o"}
	libs := []string{"c"}
	err := linkMultipleWithClang(context.Background(), objFiles, "bin", false, true, true, libs)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-fsanitize=address") || !contains(capturedArgs, "-fsanitize=undefined") {
		t.Error("sanitizer flags missing")
	}
	if !contains(capturedArgs, "-fsanitize-address-use-after-return=always") {
		t.Error("strict clang flag missing")
	}
	if !contains(capturedArgs, "-fsanitize-address-use-after-scope") {
		t.Error("strict clang flag missing")
	}
	for _, lib := range libs {
		if !contains(capturedArgs, "-l"+lib) {
			t.Errorf("missing -l%s", lib)
		}
	}
}

func TestTryAutoLinkNoTools(t *testing.T) {
	oldCheck := utils.CheckToolFunc
	defer func() { utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(name string) error {
		return fmt.Errorf("tool missing")
	}
	err := tryAutoLink(context.Background(), "obj.o", "bin", false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "no suitable linker") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTryAutoLinkMultipleNoTools(t *testing.T) {
	oldCheck := utils.CheckToolFunc
	defer func() { utils.CheckToolFunc = oldCheck }()
	utils.CheckToolFunc = func(name string) error {
		return fmt.Errorf("tool missing")
	}
	objFiles := []string{"a.o"}
	err := tryAutoLinkMultiple(context.Background(), objFiles, "bin", false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "no suitable linker") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLinkWithGccNoFallbackErrorVerbose(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("gcc error")
		},
	}
	err := linkWithGcc(context.Background(), "obj.o", "bin", true, false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "gcc error") {
		t.Errorf("error should contain command output: %v", err)
	}
}

func TestLinkWithClangFallbackVerbose(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first clang fail")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	err := linkWithClang(context.Background(), "obj.o", "bin", true, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkMultipleWithGccFallbackVerbose(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first fail")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	objFiles := []string{"a.o"}
	err := linkMultipleWithGcc(context.Background(), objFiles, "bin", true, true, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkMultipleWithClangFallbackVerbose(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first fail")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	objFiles := []string{"a.o"}
	err := linkMultipleWithClang(context.Background(), objFiles, "bin", true, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkWithLdMock(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	err := linkWithLd(context.Background(), "obj.o", "bin", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-o") {
		t.Error("missing -o flag")
	}
}

func TestLinkWithGccVerboseFalseError(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("gcc error")
		},
	}
	err := linkWithGcc(context.Background(), "obj.o", "bin", false, false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

func TestLinkWithLd(t *testing.T) {
	if _, err := exec.LookPath("ld"); err != nil {
		t.Skip("ld not installed")
	}
	dir := t.TempDir()
	obj := buildObject(t, dir, "test", `
.globl _start
_start:
    mov $60, %eax
    xor %edi, %edi
    syscall
`)
	bin := filepath.Join(dir, "test")
	err := linkWithLd(context.Background(), obj, bin, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created")
	}
}

func TestLinkMultipleWithLd(t *testing.T) {
	if _, err := exec.LookPath("ld"); err != nil {
		t.Skip("ld not installed")
	}
	dir := t.TempDir()
	obj1 := buildObject(t, dir, "a", `
.globl _start
_start:
    call b
    mov $60, %eax
    syscall
`)
	obj2 := buildObject(t, dir, "b", `
.globl b
b:
    ret
`)
	bin := filepath.Join(dir, "test")
	err := linkMultipleWithLd(context.Background(), []string{obj1, obj2}, bin, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created")
	}
}

func TestLinkWithLdMockNoRealLd(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	err := linkWithLd(context.Background(), "obj.o", "bin", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-o") {
		t.Error("missing -o flag")
	}
}

func TestLinkWithGccSharedFlag(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	oldShared := Shared
	Shared = true
	defer func() { Shared = oldShared }()
	err := linkWithGcc(context.Background(), "obj.o", "bin", false, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-shared") {
		t.Error("missing -shared flag")
	}
}

func TestLinkWithGccLdFlags(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	oldLdFlags := LdFlags
	LdFlags = "-Wl,-Map=output.map -pthread"
	defer func() { LdFlags = oldLdFlags }()
	err := linkWithGcc(context.Background(), "obj.o", "bin", false, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-Wl,-Map=output.map") || !contains(capturedArgs, "-pthread") {
		t.Errorf("LdFlags not injected correctly: %v", capturedArgs)
	}
}

func TestParanoidLinker(t *testing.T) {
	origRunner := runner
	origLookPath := lookPathFunc
	origCheckTool := utils.CheckToolFunc
	origLdFlags := LdFlags
	origShared := Shared
	origLdScript := LdScript
	origTextAddr := TextAddr
	defer func() {
		runner = origRunner
		lookPathFunc = origLookPath
		utils.CheckToolFunc = origCheckTool
		LdFlags = origLdFlags
		Shared = origShared
		LdScript = origLdScript
		TextAddr = origTextAddr
	}()

	resetMocks := func() {
		runner = &MockRunner{RunFunc: nil}
		lookPathFunc = exec.LookPath
		utils.CheckToolFunc = nil
		LdFlags = ""
		Shared = false
		LdScript = ""
		TextAddr = ""
	}

	objFile := func() string {
		f, err := os.CreateTemp("", "test*.o")
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte("dummy object content"))
		f.Close()
		return f.Name()
	}
	objPath := objFile()
	defer os.Remove(objPath)

	resetMocks()
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if name == "gcc" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected: %s", name)
	}}
	err := Link(context.Background(), objPath, "out", false, "c", false, false, false, nil)
	if err != nil {
		t.Errorf("mode c: %v", err)
	}

	resetMocks()
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if name == "ld" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected: %s", name)
	}}
	err = Link(context.Background(), objPath, "out", false, "raw", false, false, false, nil)
	if err != nil {
		t.Errorf("mode raw: %v", err)
	}

	resetMocks()
	callCount := 0
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		callCount++
		if name == "gcc" && !contains(args, "-no-pie") {
			return "", fmt.Errorf("gcc fails")
		}
		if name == "gcc" && contains(args, "-no-pie") {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = Link(context.Background(), objPath, "out", false, "auto", false, false, false, nil)
	if err != nil {
		t.Errorf("auto fallback to -no-pie: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}

	resetMocks()
	callCount = 0
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		callCount++
		if name == "gcc" {
			return "", fmt.Errorf("gcc fails")
		}
		if name == "ld" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = Link(context.Background(), objPath, "out", false, "auto", false, false, false, nil)
	if err != nil {
		t.Errorf("auto fallback to ld: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls (gcc, gcc -no-pie, ld), got %d", callCount)
	}

	resetMocks()
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "", fmt.Errorf("always fails")
	}}
	err = Link(context.Background(), objPath, "out", false, "auto", false, false, false, nil)
	if err == nil {
		t.Error("expected error when no linker works")
	}
	if !strings.Contains(err.Error(), "no suitable linker") {
		t.Errorf("wrong error: %v", err)
	}

	resetMocks()
	Shared = true
	var capturedArgs []string
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	}}
	err = linkWithGcc(context.Background(), objPath, "out", false, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-shared") {
		t.Error("missing -shared flag")
	}

	resetMocks()
	LdFlags = "-Wl,-Map=out.map -pthread"
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	}}
	err = linkWithGcc(context.Background(), objPath, "out", false, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-Wl,-Map=out.map") || !contains(capturedArgs, "-pthread") {
		t.Errorf("LdFlags not split: %v", capturedArgs)
	}

	resetMocks()
	LdScript = "linker.ld"
	TextAddr = "0x1000"
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	}}
	err = linkWithLd(context.Background(), objPath, "out", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-T") || !contains(capturedArgs, "linker.ld") ||
		!contains(capturedArgs, "-Ttext") || !contains(capturedArgs, "0x1000") {
		t.Errorf("linker script/address missing: %v", capturedArgs)
	}

	resetMocks()
	emptyObj, _ := os.CreateTemp("", "empty*.o")
	emptyObj.Close()
	defer os.Remove(emptyObj.Name())
	err = Link(context.Background(), emptyObj.Name(), "out", false, "raw", false, false, false, nil)
	if err == nil {
		t.Error("expected error for empty object")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("wrong error: %v", err)
	}

	resetMocks()
	utils.CheckToolFunc = func(name string) error {
		if name == "gcc" {
			return fmt.Errorf("gcc not found")
		}
		return nil
	}
	err = Link(context.Background(), objPath, "out", false, "c", false, false, false, nil)
	if err == nil || !strings.Contains(err.Error(), "gcc not found") {
		t.Errorf("missing gcc not detected: %v", err)
	}
	utils.CheckToolFunc = nil

	resetMocks()
	utils.CheckToolFunc = func(name string) error {
		if name == "ld" {
			return fmt.Errorf("ld not found")
		}
		return nil
	}
	err = Link(context.Background(), objPath, "out", false, "raw", false, false, false, nil)
	if err == nil || !strings.Contains(err.Error(), "ld not found") {
		t.Errorf("missing ld not detected: %v", err)
	}
	utils.CheckToolFunc = nil

	resetMocks()
	lookPathFunc = func(name string) (string, error) {
		if name == "clang" {
			return "/usr/bin/clang", nil
		}
		return "", fmt.Errorf("not found")
	}
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if name == "clang" && contains(args, "-fsanitize=address") {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = tryAutoLink(context.Background(), objPath, "out", false, true, true, nil)
	if err != nil {
		t.Errorf("strict mode with clang: %v", err)
	}
	lookPathFunc = exec.LookPath

	resetMocks()
	lookPathFunc = func(name string) (string, error) {
		if name == "clang" {
			return "", fmt.Errorf("not found")
		}
		return "/usr/bin/gcc", nil
	}
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if name == "gcc" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = tryAutoLink(context.Background(), objPath, "out", false, true, true, nil)
	lookPathFunc = exec.LookPath
	if err != nil {
		t.Errorf("clang not found fallback to gcc: %v", err)
	}

	resetMocks()
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if name == "ld" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = LinkMultiple(context.Background(), []string{objPath, objPath}, "out", false, "raw", false, false, false, nil)
	if err != nil {
		t.Errorf("LinkMultiple raw: %v", err)
	}

	resetMocks()
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if name == "gcc" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = LinkMultiple(context.Background(), []string{objPath}, "out", false, "c", false, false, false, nil)
	if err != nil {
		t.Errorf("LinkMultiple c: %v", err)
	}

	resetMocks()
	callCount = 0
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		callCount++
		if name == "gcc" {
			return "", fmt.Errorf("gcc fails")
		}
		if name == "ld" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = LinkMultiple(context.Background(), []string{objPath}, "out", false, "auto", false, false, false, nil)
	if err != nil {
		t.Errorf("LinkMultiple auto fallback: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls (gcc, gcc -no-pie, ld), got %d", callCount)
	}

	resetMocks()
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "", nil
	}}
	err = Link(context.Background(), objPath, "out", true, "raw", false, false, false, nil)
	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "Running: ld") {
		t.Error("verbose mode did not print command")
	}
	if err != nil {
		t.Errorf("verbose link: %v", err)
	}

	resetMocks()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			return "", nil
		}
	}}
	err = Link(ctx, objPath, "out", false, "raw", false, false, false, nil)
	if err == nil || err.Error() != "context canceled" {
		t.Errorf("expected context cancellation, got %v", err)
	}
}

func TestLinkInvalidMode(t *testing.T) {
	dir := t.TempDir()
	obj := filepath.Join(dir, "obj.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "bin")
	err := Link(context.Background(), obj, bin, false, "invalid", false, false, false, nil)
	if err == nil {
		t.Error("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "unsupported mode") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestLinkMultipleInvalidMode(t *testing.T) {
	dir := t.TempDir()
	obj := filepath.Join(dir, "obj.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := LinkMultiple(context.Background(), []string{obj}, "bin", false, "invalid", false, false, false, nil)
	if err == nil {
		t.Error("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "unsupported mode") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestLinkEmptyObjectList(t *testing.T) {
	err := LinkMultiple(context.Background(), []string{}, "bin", false, "raw", false, false, false, nil)
	if err == nil {
		t.Error("expected error for empty object list")
	}
	if !strings.Contains(err.Error(), "no object files") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestLinkNonexistentObject(t *testing.T) {
	err := Link(context.Background(), "nonexistent.o", "bin", false, "raw", false, false, false, nil)
	if err == nil {
		t.Error("expected error for nonexistent object")
	}
}

func TestResponseFileCreation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("response file test skipped on windows")
	}
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var argsUsed []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			argsUsed = args
			return "", nil
		},
	}
	longArgs := make([]string, 200)
	for i := range longArgs {
		longArgs[i] = "very_long_argument_to_force_response_file_usage"
	}
	_, err := runLinkerCommand(context.Background(), false, "testlinker", longArgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(argsUsed) != 1 || !strings.HasPrefix(argsUsed[0], "@") {
		t.Errorf("expected response file argument, got %v", argsUsed)
	}
}
