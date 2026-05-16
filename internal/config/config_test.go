package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".fz.yaml")
	content := `
source_dir: ./src
output: mybin
mode: raw
debug: true
verbose: false
keep_obj: true
no_cache: false
exclude:
  - "test_*"
flags:
  asm: ["-felf64"]
  ld: ["-T", "linker.ld"]
`
	err := os.WriteFile(cfgPath, []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceDir != "./src" {
		t.Errorf("SourceDir = %q, want ./src", cfg.SourceDir)
	}
	if cfg.Mode != "raw" {
		t.Errorf("Mode = %q, want raw", cfg.Mode)
	}
	if !cfg.Debug {
		t.Error("Debug should be true")
	}
	if len(cfg.Flags.Asm) != 1 || cfg.Flags.Asm[0] != "-felf64" {
		t.Error("Flags.Asm not parsed")
	}
}

func TestValidate(t *testing.T) {
	cfg := &Config{}
	err := cfg.Validate()
	if err == nil {
		t.Error("expected error: no source_dir or source_file")
	}
	cfg.SourceDir = "src"
	err = cfg.Validate()
	if err != nil {
		t.Error(err)
	}
	cfg.Mode = "invalid"
	err = cfg.Validate()
	if err == nil {
		t.Error("expected invalid mode error")
	}
}

func TestMergeFromFlags(t *testing.T) {
	cfg := &Config{SourceDir: "orig", Mode: "auto"}
	cfg.MergeFromFlags("file.asm", "", "newbin", "new.o", true, true, true, true, "raw")
	if cfg.SourceFile != "file.asm" {
		t.Error("SourceFile not merged")
	}
	if cfg.SourceDir != "" {
		t.Error("SourceDir should be cleared when -asm used")
	}
	if cfg.Output != "newbin" {
		t.Error("Output not merged")
	}
	if !cfg.Debug || !cfg.Verbose || !cfg.KeepObj || !cfg.NoCache {
		t.Error("bool flags not merged")
	}
	if cfg.Mode != "raw" {
		t.Error("Mode not merged")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(dir)
	if path := DefaultConfigPath(); path != "" {
		t.Errorf("expected empty, got %s", path)
	}
	os.WriteFile(".fz.yaml", []byte{}, 0o644)
	if path := DefaultConfigPath(); path != ".fz.yaml" {
		t.Errorf("expected .fz.yaml, got %s", path)
	}
}
