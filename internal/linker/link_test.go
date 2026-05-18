package linker

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
