package linker

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
	err := Link(context.Background(), obj, bin, false, "raw", false, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created")
	}
	err = Link(context.Background(), obj, filepath.Join(dir, "test2"), false, "raw", true, true, false)
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
	err := LinkMultiple(context.Background(), []string{obj1, obj2}, bin, false, "raw", false, true, false)
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
	err := Link(context.Background(), obj, bin, false, "c", false, true, false)
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
	err := Link(context.Background(), obj, bin, false, "auto", false, true, false)
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
	err := Link(context.Background(), obj, bin, false, "c", false, true, false)
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
	err := Link(context.Background(), obj, bin, false, "auto", false, true, true)
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
	err := Link(context.Background(), obj, bin, false, "raw", true, true, false)
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
	err = Link(context.Background(), emptyObj, bin, false, "raw", false, true, false)
	if err == nil {
		t.Error("expected error for empty object file")
	}
}
