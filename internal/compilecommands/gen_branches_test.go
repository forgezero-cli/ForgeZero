package compilecommands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fz/internal/config"
)

func chdirTemp(t *testing.T, dir string) func() {
	t.Helper()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return func() { _ = os.Chdir(oldWd) }
}

func TestGenerateNilConfig(t *testing.T) {
	dir := t.TempDir()
	defer chdirTemp(t, dir)()
	c := filepath.Join(dir, "main.c")
	if err := os.WriteFile(c, []byte("int main(){}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Generate(nil, dir); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateSkipsAsm(t *testing.T) {
	dir := t.TempDir()
	defer chdirTemp(t, dir)()
	if err := os.WriteFile(filepath.Join(dir, "a.asm"), []byte("nop"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{SourceFiles: []string{filepath.Join(dir, "a.asm")}}
	if err := Generate(cfg, dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("compile_commands.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "[]\n" && string(data) != "[]" && string(data) != "null\n" && string(data) != "null" {
		t.Fatalf("expected empty commands: %s", data)
	}
}

func TestGenerateDedup(t *testing.T) {
	dir := t.TempDir()
	defer chdirTemp(t, dir)()
	c := filepath.Join(dir, "x.c")
	if err := os.WriteFile(c, []byte("int x;"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{SourceFiles: []string{c, c}}
	if err := Generate(cfg, dir); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateWithDebug(t *testing.T) {
	dir := t.TempDir()
	defer chdirTemp(t, dir)()
	c := filepath.Join(dir, "main.c")
	if err := os.WriteFile(c, []byte("int main(){}"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{SourceFiles: []string{c}, Debug: true}
	if err := Generate(cfg, dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("compile_commands.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "-g") {
		t.Fatal("missing debug flag")
	}
}
