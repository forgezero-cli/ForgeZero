package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTasks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fz.toml")
	data := "[[tasks]]\nname = \"codegen\"\nstage = \"pre-build\"\ncommand = \"python3 gen.py\"\ninputs = [\"schema.json\"]\noutputs = [\"generated.c\"]\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Tasks) != 1 || cfg.Tasks[0].Name != "codegen" || cfg.Tasks[0].Stage != "pre-build" {
		t.Fatalf("unexpected tasks: %+v", cfg.Tasks)
	}
}
