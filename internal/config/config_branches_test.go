package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeAllFields(t *testing.T) {
	base := &Config{}
	other := &Config{
		Name:          "proj",
		SourceDirs:    []string{"d1"},
		SourceFiles:   []string{"f1.c"},
		Output:        "out",
		OutObj:        "obj",
		Mode:          "raw",
		Debug:         true,
		Verbose:       true,
		KeepObj:       true,
		NoCache:       true,
		Exclude:       []string{"*.o"},
		Include:       []string{"*.c"},
		Libs:          []string{"m"},
		IgnoreFile:    ".fzignore",
		AuditIgnore:   []string{"vendor"},
		ToolChecksums: map[string]string{"gcc": "abc"},
		Flags:         Flags{Asm: []string{"-felf64"}, Cc: []string{"-O2"}, Ld: []string{"-T"}},
	}
	base.Merge(other)
	if base.Name != "proj" || base.Output != "out" || len(base.SourceDirs) != 1 {
		t.Fatal("merge incomplete")
	}
	if base.ToolChecksums["gcc"] != "abc" {
		t.Fatal("checksums not merged")
	}
	other2 := &Config{SourceFile: "main.asm"}
	base.Merge(other2)
	if base.SourceFile != "main.asm" || base.SourceDir != "" {
		t.Fatal("source file merge failed")
	}
	base.Merge(nil)
}

func TestLoadMergedExplicitInvalid(t *testing.T) {
	_, err := LoadMerged("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected load error")
	}
}

func TestLoadMergedExplicitOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	os.WriteFile(path, []byte("source_dir: ./x\noutput: bin\n"), 0o644)
	cfg, err := LoadMerged(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceDir != "./x" {
		t.Fatal(cfg.SourceDir)
	}
}

func TestMergeFromFlagsSourceDirs(t *testing.T) {
	cfg := &Config{SourceDirs: []string{"a"}}
	cfg.MergeFromFlags("", "dir", "", "", false, false, false, false, "", "", "")
	if cfg.SourceDir != "dir" {
		t.Fatal(cfg.SourceDir)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte(":\n\tbad"), 0o644)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected yaml error")
	}
}

func TestValidateToolchain(t *testing.T) {
	cfg := &Config{SourceDir: "src", Toolchain: "invalid"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected toolchain error")
	}
}
