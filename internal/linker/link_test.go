package linker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	expected := []string{"test.o", "-o", "bin", "-Wl,-T,script.ld", "-Wl,-Ttext=0x1000"}
	if !equalSlices(got, expected) {
		t.Errorf("ApplyGccLdFlags = %v, want %v", got, expected)
	}
	got = ApplyGccLdFlags(args, "", "")
	if !equalSlices(got, args) {
		t.Error("should not modify args when empty")
	}
}

func TestApplyLdFlags(t *testing.T) {
	args := []string{"test.o", "-o", "bin"}
	got := ApplyLdFlags(args, "script.ld", "0x1000")
	expected := []string{"test.o", "-o", "bin", "-T", "script.ld", "-Ttext", "0x1000"}
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
	t.Skip("skipping auto link noclang")
	oldRunner := runner
	defer func() { runner = oldRunner }()
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			if name == "gcc" {
				return "", nil
			}
			return "", nil
		},
	}
	ctx := context.Background()
	err := tryAutoLink(ctx, "obj.o", "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
}

// ----- Test linkMultipleWithGcc -----
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

// ----- Test linkMultipleWithClang -----
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

// ----- Test tryAutoLinkMultiple with clang fail then gcc -----
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
		t.Errorf("expected 2 calls (clang fail, gcc success), got %d", callCount)
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

// Test linkWithClang fallback to -no-pie
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

func TestTryAutoLinkMultipleClangSuccess(t *testing.T) {
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
