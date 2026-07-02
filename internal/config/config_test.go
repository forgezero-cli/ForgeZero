/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

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
	cfg := &Config{SourceDir: "src", Mode: "auto"}
	err := cfg.Validate()
	if err != nil {
		t.Error(err)
	}
	cfg.Mode = "invalid"
	err = cfg.Validate()
	if err == nil {
		t.Error("expected invalid mode error")
	}
	cfg.Mode = "auto"
	cfg.SourceDir = ""
	cfg.SourceFile = ""
	err = cfg.Validate()
	if err != nil {
		t.Error("empty config should be valid (partial)")
	}
	cfg.CacheMode = CacheModeRAM
	err = cfg.Validate()
	if err != nil {
		t.Error(err)
	}
	cfg.CacheMode = "invalid"
	err = cfg.Validate()
	if err == nil {
		t.Error("expected invalid cache_mode error")
	}
}

func TestMergeFromFlags(t *testing.T) {
	cfg := &Config{SourceDir: "orig", Mode: "auto"}
	cfg.MergeFromFlags("file.asm", "", "newbin", "new.o", true, true, true, true, "raw", "auto", "standard")
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
	if cfg.Isolation != IsolationStandard {
		t.Error("isolation not merged")
	}
}

func TestMerge(t *testing.T) {
	base := &Config{SourceDir: "base", Mode: "auto"}
	other := &Config{SourceDir: "other", Output: "out", Debug: true}
	base.Merge(other)
	if base.SourceDir != "other" {
		t.Error("SourceDir not overwritten")
	}
	if base.Output != "out" {
		t.Error("Output not merged")
	}
	if !base.Debug {
		t.Error("Debug not merged")
	}
}

func TestFindConfigs(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change to test directory: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	system, user, local := FindConfigs()
	if system != "" || user != "" || local != "" {
		t.Errorf("found unexpected configs: system=%s user=%s local=%s", system, user, local)
	}
	if err := os.WriteFile(".fz.yaml", []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, local = FindConfigs()
	if local != ".fz.yaml" {
		t.Errorf("expected .fz.yaml, got %s", local)
	}
}

func TestLoadMerged(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change to test directory: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	cfg, err := LoadMerged("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceDir != "" {
		t.Error("expected empty config")
	}
	if err := os.WriteFile(".fz.yaml", []byte("source_dir: ./src"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadMerged("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceDir != "./src" {
		t.Errorf("expected source_dir ./src, got %s", cfg.SourceDir)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change to test directory: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	if path := DefaultConfigPath(); path != "" {
		t.Errorf("expected empty, got %s", path)
	}
	if err := os.WriteFile(".fz.yaml", []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if path := DefaultConfigPath(); path != ".fz.yaml" {
		t.Errorf("expected .fz.yaml, got %s", path)
	}
}
