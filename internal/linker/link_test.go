package linker

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLink(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "test.s")
	err := os.WriteFile(src, []byte(`
.globl _start
_start:
	mov $60, %eax
	xor %edi, %edi
	syscall
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "test.o")
	cmd := exec.Command("gcc", "-c", src, "-o", obj)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("gcc -c failed: %v\n%s", err, out)
	}
	bin := filepath.Join(dir, "test")
	err = Link(context.Background(), obj, bin, false, "raw", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created")
	}
	err = Link(context.Background(), obj, filepath.Join(dir, "test2"), false, "raw", true, true)
	if err != nil {
		t.Error(err)
	}
}
