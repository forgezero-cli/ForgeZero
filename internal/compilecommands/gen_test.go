package compilecommands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fz/internal/config"
)

func TestGenerateWithSpacesAndMixedFiles(t *testing.T) {
	dir := t.TempDir()
	spaceDir := filepath.Join(dir, "sub dir")
	if err := os.MkdirAll(spaceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cFile := filepath.Join(spaceDir, "main.c")
	if err := os.WriteFile(cFile, []byte("int main() { return 0; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		SourceFiles: []string{cFile},
	}
	oldWd, _ := os.Getwd()
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := Generate(cfg, "."); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("compile_commands.json")
	if err != nil {
		t.Fatal(err)
	}
	var cmds []CompileCommand
	if err := json.Unmarshal(data, &cmds); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(cmds) != 1 {
		t.Errorf("expected 1 command, got %d", len(cmds))
	}
	if !strings.Contains(string(data), "sub dir") {
		t.Error("path with space not preserved")
	}
}
