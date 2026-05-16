package linker

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func buildObjectWithAS(t *testing.T, dir, name, asmContent string) string {
	src := filepath.Join(dir, name+".s")
	err := os.WriteFile(src, []byte(asmContent), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, name+".o")
	cmd := exec.Command("as", src, "-o", obj)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("as failed: %v\n%s", err, out)
	}
	return obj
}

func TestCheckDuplicateSymbols(t *testing.T) {
	if _, err := exec.LookPath("as"); err != nil {
		t.Skip("as not installed")
	}
	if _, err := exec.LookPath("objdump"); err != nil {
		t.Skip("objdump not installed")
	}
	dir := t.TempDir()
	obj1 := buildObjectWithAS(t, dir, "a", `
.globl my_func
my_func:
	mov $1, %eax
	ret
`)
	obj2 := buildObjectWithAS(t, dir, "b", `
.globl my_func
my_func:
	mov $2, %eax
	ret
`)
	err := CheckDuplicateSymbols([]string{obj1, obj2}, true) // verbose для диагностики
	if err == nil {
		t.Error("expected duplicate symbol error")
	}
	obj3 := buildObjectWithAS(t, dir, "c", `
.globl other_func
other_func:
	ret
`)
	err = CheckDuplicateSymbols([]string{obj1, obj3}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
