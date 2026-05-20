//go:build wasm_tests || all

package assembler

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"fz/internal/config"
	"fz/internal/sbom"
)

func TestIsWasmTarget(t *testing.T) {
	tests := []struct {
		target string
		want   bool
	}{
		{"wasm32-unknown-unknown", true},
		{"wasm32-wasi", true},
		{"wasm", true},
		{"x86_64-linux-gnu", false},
		{"arm-linux-gnueabihf", false},
	}
	oldTarget := Target
	defer func() { Target = oldTarget }()
	for _, tt := range tests {
		Target = tt.target
		if got := isWasmTarget(); got != tt.want {
			t.Errorf("isWasmTarget(%q) = %v, want %v", tt.target, got, tt.want)
		}
	}
}

func TestCCompileToWasm(t *testing.T) {
	_, err := exec.LookPath("emcc")
	if err != nil {
		if _, err := exec.LookPath("clang"); err != nil {
			t.Skip("neither emcc nor clang found, skipping wasm test")
		}
	}
	oldTarget := Target
	defer func() { Target = oldTarget }()
	Target = "wasm32-unknown-unknown"

	dir := t.TempDir()
	src := filepath.Join(dir, "test.c")
	if err := os.WriteFile(src, []byte("int main(void) { return 0; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "test.o")

	err = Assemble(context.Background(), src, obj, false, true, "auto")
	if err != nil {
		if err.Error() == "no suitable compiler found" {
			t.Skip("no wasm compiler available")
		}
		t.Fatal(err)
	}
	if _, err := os.Stat(obj); err != nil {
		t.Error("object file not created")
	}
}

func TestAssembleUnsupportedForWasm(t *testing.T) {
	oldTarget := Target
	defer func() { Target = oldTarget }()
	Target = "wasm32-wasi"

	dir := t.TempDir()
	src := filepath.Join(dir, "test.asm")
	if err := os.WriteFile(src, []byte("nop"), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "test.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err == nil {
		t.Error("expected error for .asm on wasm target")
	}
	if !strings.Contains(err.Error(), "cannot assemble") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestSBOMWithWasmTarget(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}
	_, err := sbom.Generate(dir, "vendor", "2.2.0", cfg, "wasm32-unknown-unknown")
	if err != nil {
		t.Fatal(err)
	}
}
