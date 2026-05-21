package compilecommands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fz/internal/config"
)

func TestGenerateNilConfig(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)
	c := filepath.Join(dir, "main.c")
	os.WriteFile(c, []byte("int main(){}"), 0o644)
	if err := Generate(nil, dir); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateSkipsAsm(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)
	os.WriteFile(filepath.Join(dir, "a.asm"), []byte("nop"), 0o644)
	cfg := &config.Config{SourceFiles: []string{filepath.Join(dir, "a.asm")}}
	if err := Generate(cfg, dir); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile("compile_commands.json")
	if string(data) != "[]\n" && string(data) != "[]" && string(data) != "null\n" && string(data) != "null" {
		t.Fatalf("expected empty commands: %s", data)
	}
}

func TestGenerateDedup(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)
	c := filepath.Join(dir, "x.c")
	os.WriteFile(c, []byte("int x;"), 0o644)
	cfg := &config.Config{SourceFiles: []string{c, c}}
	if err := Generate(cfg, dir); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateWithDebug(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)
	c := filepath.Join(dir, "main.c")
	os.WriteFile(c, []byte("int main(){}"), 0o644)
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
