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
	"io"
	"os"
	"path/filepath"
	"strings"
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
	if err := os.WriteFile(".fz.toml", []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, local = FindConfigs()
	if local != ".fz.toml" {
		t.Errorf("expected .fz.toml, got %s", local)
	}
}

func TestLoadTOMLConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".fz.toml")
	content := `output = "mybin"
mode = "raw"
debug = true
[flags]
cc = ["-O2"]`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "mybin" {
		t.Errorf("Output = %q, want mybin", cfg.Output)
	}
	if cfg.Mode != "raw" {
		t.Errorf("Mode = %q, want raw", cfg.Mode)
	}
	if !cfg.Debug {
		t.Error("Debug should be true")
	}
	if len(cfg.Flags.Cc) != 1 || cfg.Flags.Cc[0] != "-O2" {
		t.Error("Flags.Cc not parsed")
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

func TestLoadTOMLConfigIncludesRelativeFiles(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.toml")
	if err := os.WriteFile(basePath, []byte("output = \"from-base\"\nmode = \"raw\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "app.toml")
	if err := os.WriteFile(cfgPath, []byte("include = [\"base.toml\"]\noutput = \"from-app\"\n[flags]\ncc = [\"-O2\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "from-app" {
		t.Fatalf("Output = %q, want from-app", cfg.Output)
	}
	if cfg.Mode != "raw" {
		t.Fatalf("Mode = %q, want raw", cfg.Mode)
	}
	if len(cfg.Flags.Cc) != 1 || cfg.Flags.Cc[0] != "-O2" {
		t.Fatalf("Flags.Cc = %v, want [-O2]", cfg.Flags.Cc)
	}
}

func TestLoadTOMLEnumValues(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".fz.toml")
	content := "isolation = \"strict\"\ncache_mode = \"ram\"\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Isolation != IsolationStrict {
		t.Fatalf("Isolation = %q, want %q", cfg.Isolation, IsolationStrict)
	}
	if cfg.CacheMode != CacheModeRAM {
		t.Fatalf("CacheMode = %q, want %q", cfg.CacheMode, CacheModeRAM)
	}
}

func TestLoadTOMLEnvironmentVariables(t *testing.T) {
	t.Setenv("FZ_TEST_OUTPUT", "mars")
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".fz.toml")
	content := "output = \"${FZ_TEST_OUTPUT}\"\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "mars" {
		t.Fatalf("Output = %q, want mars", cfg.Output)
	}
}

func TestLoadYAMLDeprecationWarning(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("output: app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()
	_, err = Load(cfgPath)
	_ = w.Close()
	data, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected deprecation warning")
	}
}

func TestApplySetOverrides(t *testing.T) {
	cfg := &Config{}
	if err := cfg.ApplySetOverrides([]string{"output=release", "mode=raw", "debug=true", "optimization_level=3"}); err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "release" {
		t.Fatalf("Output = %q, want release", cfg.Output)
	}
	if cfg.Mode != "raw" {
		t.Fatalf("Mode = %q, want raw", cfg.Mode)
	}
	if !cfg.Debug {
		t.Fatal("Debug should be true")
	}
	if cfg.OptimizationLevel != 3 {
		t.Fatalf("OptimizationLevel = %d, want 3", cfg.OptimizationLevel)
	}
}

func TestApplySetOverridesNestedKeys(t *testing.T) {
	cfg := &Config{}
	err := cfg.ApplySetOverrides([]string{
		"variables.DEBUG=1",
		"toolchain_opts.search_priority=clang,gcc",
		"toolchain_opts.env_allow=CC,CXX",
		"toolchain_opts.tool_paths.gcc=/usr/bin/gcc",
		"hooks.on_failure=echo fail",
		"iso.enabled=true",
		"iso.output=boot.iso",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Variables["DEBUG"] != "1" {
		t.Fatalf("Variables.DEBUG = %q, want 1", cfg.Variables["DEBUG"])
	}
	if len(cfg.ToolchainSettings.SearchPriority) != 2 || cfg.ToolchainSettings.SearchPriority[0] != "clang" {
		t.Fatalf("SearchPriority = %v, want [clang gcc]", cfg.ToolchainSettings.SearchPriority)
	}
	if len(cfg.ToolchainSettings.EnvAllow) != 2 || cfg.ToolchainSettings.EnvAllow[1] != "CXX" {
		t.Fatalf("EnvAllow = %v, want [CC CXX]", cfg.ToolchainSettings.EnvAllow)
	}
	if cfg.ToolchainSettings.ToolPaths["gcc"] != "/usr/bin/gcc" {
		t.Fatalf("ToolPaths[gcc] = %q, want /usr/bin/gcc", cfg.ToolchainSettings.ToolPaths["gcc"])
	}
	if cfg.Hooks.OnFailure != "echo fail" {
		t.Fatalf("Hooks.OnFailure = %q, want echo fail", cfg.Hooks.OnFailure)
	}
	if !cfg.ISO.Enabled {
		t.Fatal("ISO.Enabled should be true")
	}
	if cfg.ISO.Output != "boot.iso" {
		t.Fatalf("ISO.Output = %q, want boot.iso", cfg.ISO.Output)
	}
}

func TestGenerateConfigHeader(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "config.h.in")
	outputPath := filepath.Join(dir, "config.h")
	if err := os.WriteFile(templatePath, []byte("#define FZ_OUTPUT \"${OUTPUT}\"\n#define FZ_MODE \"${MODE}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Output: "release", Mode: "raw"}
	if err := GenerateConfigH(templatePath, outputPath, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `#define FZ_OUTPUT "release"`) {
		t.Fatalf("header missing output substitution: %s", content)
	}
	if !strings.Contains(content, `#define FZ_MODE "raw"`) {
		t.Fatalf("header missing mode substitution: %s", content)
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
