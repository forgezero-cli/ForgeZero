package musl

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestToolchainPrepareSuccess(t *testing.T) {
	tc := NewToolchain("x86_64")
	defer tc.Close()

	dir, err := tc.Prepare()
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	if dir == "" {
		t.Error("Prepare returned empty directory")
	}
	if tc.tmpDir != dir {
		t.Error("tmpDir not set correctly")
	}

	files := []string{"crt1.o", "crti.o", "crtn.o", "libc.a"}
	for _, f := range files {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing file: %s", f)
		}
	}
}

func TestToolchainPrepareInvalidArch(t *testing.T) {
	tc := NewToolchain("invalid-arch")
	defer tc.Close()

	_, err := tc.Prepare()
	if err == nil {
		t.Error("expected error for invalid architecture")
	}
}

func TestToolchainGetLinkerArgsWithoutPrepare(t *testing.T) {
	tc := NewToolchain("x86_64")
	_, err := tc.GetLinkerArgs([]string{"a.o"}, "out")
	if err == nil {
		t.Error("expected error when not prepared")
	}
}

func TestToolchainGetLinkerArgsAfterPrepare(t *testing.T) {
	tc := NewToolchain("x86_64")
	dir, err := tc.Prepare()
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	defer tc.Close()

	userObjs := []string{"main.o", "utils.o"}
	output := "test_bin"

	args, err := tc.GetLinkerArgs(userObjs, output)
	if err != nil {
		t.Fatalf("GetLinkerArgs failed: %v", err)
	}

	expected := []string{
		"-static",
		"-nostdlib",
		filepath.Join(dir, "crt1.o"),
		filepath.Join(dir, "crti.o"),
		"main.o",
		"utils.o",
		"-L" + dir,
		"-lc",
		filepath.Join(dir, "crtn.o"),
		"-o",
		output,
	}

	if len(args) != len(expected) {
		t.Errorf("expected %d args, got %d", len(expected), len(args))
	}
	for i := range expected {
		if i >= len(args) {
			break
		}
		if args[i] != expected[i] {
			t.Errorf("arg[%d]: expected %q, got %q", i, expected[i], args[i])
		}
	}
}

func TestToolchainClose(t *testing.T) {
	tc := NewToolchain("x86_64")
	dir, err := tc.Prepare()
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("temp dir should exist: %v", err)
	}

	err = tc.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if _, err := os.Stat(dir); err == nil {
		t.Error("temp dir still exists after Close")
	}
	if tc.tmpDir != "" {
		t.Error("tmpDir not cleared")
	}
}

func TestToolchainIntegration(t *testing.T) {
	tc := NewToolchain("x86_64")
	dir, err := tc.Prepare()
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	defer tc.Close()

	cFile := filepath.Join(dir, "test.c")
	cContent := `#include <stdio.h>
int main() {
    printf("Hello from musl\n");
    return 0;
}`
	if err := os.WriteFile(cFile, []byte(cContent), 0644); err != nil {
		t.Fatalf("write c file: %v", err)
	}

	objFile := filepath.Join(dir, "test.o")
	compileCmd := exec.Command("gcc", "-c", cFile, "-o", objFile)
	if out, err := compileCmd.CombinedOutput(); err != nil {
		t.Skipf("gcc not available or compile failed: %v\n%s", err, out)
	}

	outputBin := filepath.Join(dir, "static_bin")
	args, err := tc.GetLinkerArgs([]string{objFile}, outputBin)
	if err != nil {
		t.Fatalf("GetLinkerArgs failed: %v", err)
	}

	linker := "ld.lld"
	if _, err := exec.LookPath("ld.lld"); err != nil {
		linker = "ld"
	}
	linkCmd := exec.Command(linker, args...)
	if out, err := linkCmd.CombinedOutput(); err != nil {
		t.Fatalf("link failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(outputBin); err != nil {
		t.Fatalf("binary not created: %v", err)
	}

	runCmd := exec.Command(outputBin)
	if out, err := runCmd.CombinedOutput(); err != nil {
		t.Fatalf("run failed: %v\n%s", err, out)
	}
}
