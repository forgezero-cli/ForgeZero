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
	err = Link(context.Background(), obj, filepath.Join(dir, "test2"), false, "raw", true, true, false, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestLinkMultiple(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
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
	bin := filepath.Join(dir, "multi")
	err := LinkMultiple(context.Background(), []string{obj1, obj2}, bin, false, "raw", false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created")
	}
}

func TestLinkGccMode(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	obj := buildObject(t, dir, "main", `
.globl main
main:
	mov $0, %eax
	ret
`)
	bin := filepath.Join(dir, "c_mode")
	err := Link(context.Background(), obj, bin, false, "c", false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created")
	}
}

func TestLinkAutoFallback(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	obj := buildObject(t, dir, "start", `
.globl _start
_start:
	mov $60, %eax
	syscall
`)
	bin := filepath.Join(dir, "auto")
	err := Link(context.Background(), obj, bin, false, "auto", false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created")
	}
}

func TestLinkWithSanitizers(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	obj := buildObject(t, dir, "san", `
.globl main
main:
	mov $0, %eax
	ret
`)
	bin := filepath.Join(dir, "san_bin")
	err := Link(context.Background(), obj, bin, false, "c", false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created with sanitizers")
	}
}

func TestLinkStrictMode(t *testing.T) {
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed (required for strict mode)")
	}
	dir := t.TempDir()
	obj := buildObject(t, dir, "strict", `
.globl main
main:
	mov $0, %eax
	ret
`)
	bin := filepath.Join(dir, "strict_bin")
	err := Link(context.Background(), obj, bin, false, "auto", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created with strict sanitizers")
	}
}

func TestLinkNoSymbolCheck(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	obj := buildObject(t, dir, "nocheck", `
.globl _start
_start:
	mov $60, %eax
	syscall
`)
	bin := filepath.Join(dir, "nocheck")
	err := Link(context.Background(), obj, bin, false, "raw", true, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created with no-symbol-check")
	}
}

func TestLinkEmptyObject(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	emptyObj := filepath.Join(dir, "empty.o")
	err := os.WriteFile(emptyObj, []byte{}, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "empty_bin")
	err = Link(context.Background(), emptyObj, bin, false, "raw", false, true, false, nil)
	if err == nil {
		t.Error("expected error for empty object file")
	}
}

// ----- Mock tests -----

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

func TestLinkAutoFallbackGccToNoPie(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	dir := t.TempDir()
	obj := filepath.Join(dir, "test.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "out")

	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if name == "gcc" && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first gcc fails")
			}
			if name == "gcc" && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	err := Link(context.Background(), obj, bin, false, "auto", false, false, false, nil)
	if err != nil {
		t.Fatalf("expected success after fallback, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkAutoFallbackGccToLd(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	dir := t.TempDir()
	obj := filepath.Join(dir, "test.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "out")

	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if name == "gcc" && !contains(args, "-no-pie") {
				return "", fmt.Errorf("gcc fails")
			}
			if name == "gcc" && contains(args, "-no-pie") {
				return "", fmt.Errorf("gcc -no-pie fails")
			}
			if name == "ld" {
				return "", nil
			}
			return "", nil
		},
	}
	err := Link(context.Background(), obj, bin, false, "auto", false, false, false, nil)
	if err != nil {
		t.Fatalf("expected fallback to ld, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestLinkStrictModeWithClangMock(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	dir := t.TempDir()
	obj := filepath.Join(dir, "test.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "out")

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
	err := Link(context.Background(), obj, bin, false, "auto", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !clangCalled {
		t.Error("clang was not called in strict mode")
	}
}

func TestLinkWithSanitizerFlags(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	dir := t.TempDir()
	obj := filepath.Join(dir, "test.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "out")

	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	err := Link(context.Background(), obj, bin, false, "c", false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-fsanitize=address") {
		t.Error("missing -fsanitize=address")
	}
	if !contains(capturedArgs, "-fsanitize=undefined") {
		t.Error("missing -fsanitize=undefined")
	}
}

func TestLinkWithStrictGccFlags(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	dir := t.TempDir()
	obj := filepath.Join(dir, "test.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "out")

	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	err := Link(context.Background(), obj, bin, false, "c", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-fsanitize-address-use-after-scope") {
		t.Error("missing -fsanitize-address-use-after-scope in strict mode")
	}
}

// ----- Test LinkMultiple with 'c' mode and sanitizers -----
func TestLinkMultipleCModeWithLibs(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	obj := buildObject(t, dir, "main", `
.globl main
main:
	mov $0, %eax
	ret
`)
	bin := filepath.Join(dir, "out")
	err := LinkMultiple(context.Background(), []string{obj}, bin, false, "c", false, true, false, []string{"m"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created")
	}
}

func TestLinkMultipleWithGccNoFallback(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	dir := t.TempDir()
	obj := filepath.Join(dir, "test.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "out")

	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("gcc error")
		},
	}
	err := linkMultipleWithGcc(context.Background(), []string{obj}, bin, false, false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

// ----- Test linkMultipleWithClang without fallback -----
func TestLinkMultipleWithClangNoFallback(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	dir := t.TempDir()
	obj := filepath.Join(dir, "test.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "out")

	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("clang error")
		},
	}
	err := linkMultipleWithClang(context.Background(), []string{obj}, bin, false, false, true, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

// ----- Test tryAutoLinkMultiple when no clang and gcc fails -----
func TestTryAutoLinkMultipleNoLinkerFallback(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	dir := t.TempDir()
	obj := filepath.Join(dir, "test.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "out")

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("command not found")
		},
	}
	err := tryAutoLinkMultiple(context.Background(), []string{obj}, bin, false, false, false, nil)
	if err == nil {
		t.Error("expected error when no linker available")
	}
	if !strings.Contains(err.Error(), "auto linking failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ----- Test linkMultipleWithLd with empty object -----
func TestLinkMultipleWithLdEmptyObject(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	dir := t.TempDir()
	emptyObj := filepath.Join(dir, "empty.o")
	if err := os.WriteFile(emptyObj, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "out")
	err := LinkMultiple(context.Background(), []string{emptyObj}, bin, false, "raw", false, false, false, nil)
	if err == nil {
		t.Error("expected error for empty object file")
	}
}
