package musl

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMuslIntegration(t *testing.T) {
	tmpDir, err := ExtractMusl("x86_64")
	if err != nil {
		t.Fatalf("ExtractMusl failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testC := `#include <stdio.h>
int main() {
    printf("Hello from Musl static binary!\n");
    return 0;
}
`

	cFile := filepath.Join(tmpDir, "test_musl.c")
	if err := os.WriteFile(cFile, []byte(testC), 0644); err != nil {
		t.Fatalf("write c file: %v", err)
	}

	objFile := filepath.Join(tmpDir, "test_musl.o")
	compileCmd := exec.Command("gcc", "-c", cFile, "-o", objFile)
	if out, err := compileCmd.CombinedOutput(); err != nil {
		t.Fatalf("gcc compile failed: %v\n%s", err, out)
	}

	outputBin := "test_static_bin"
	flags := GetMuslLinkerFlags(tmpDir, []string{objFile}, outputBin)

	ldPath := "ld"
	if _, err := exec.LookPath("ld.lld"); err == nil {
		ldPath = "ld.lld"
	}

	linkCmd := exec.Command(ldPath, flags...)
	if out, err := linkCmd.CombinedOutput(); err != nil {
		t.Fatalf("link failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(outputBin); err != nil {
		t.Fatalf("binary not created: %v", err)
	}

	t.Logf("Static binary created: %s", outputBin)
}
