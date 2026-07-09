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

package fzp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessBasic(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.fz")
	if err := os.WriteFile(cfgPath, []byte("#define OUTPUT app\n#define MODE raw\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proc := NewProcessor(Options{RootDir: dir})
	out, err := proc.Process(cfgPath, Options{RootDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Fatalf("unexpected output %q", out)
	}
	defs, err := proc.ParseDefinitions("#define OUTPUT app\n#define MODE raw\n")
	if err != nil {
		t.Fatal(err)
	}
	if defs["OUTPUT"] != "app" {
		t.Fatalf("OUTPUT = %q, want app", defs["OUTPUT"])
	}
}

func TestProcessConditionAndInclude(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "base.fz")
	if err := os.WriteFile(includePath, []byte("#define EXTRA enabled\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.fz")
	if err := os.WriteFile(cfgPath, []byte("#define ENABLED 1\n#ifdef ENABLED\n#define OUTPUT app\n#endif\n#include \"base.fz\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proc := NewProcessor(Options{RootDir: dir})
	_, err := proc.Process(cfgPath, Options{RootDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defs, err := proc.ParseDefinitions("#define OUTPUT app\n#define EXTRA enabled\n")
	if err != nil {
		t.Fatal(err)
	}
	if defs["OUTPUT"] != "app" {
		t.Fatalf("OUTPUT = %q, want app", defs["OUTPUT"])
	}
}

func TestEvaluateExpression(t *testing.T) {
	parser := newParser("(2 + 3) * 4", map[string]macro{})
	got := parser.parse()
	if got != 20 {
		t.Fatalf("got %d, want 20", got)
	}
	parser = newParser("1 + 2", map[string]macro{"ENABLE": {value: "1"}})
	if parser.parse() != 3 {
		t.Fatal("expected arithmetic parser to evaluate")
	}
}

func TestParserHandlesDefinedOperator(t *testing.T) {
	parser := newParser("defined(__linux__)", map[string]macro{"__linux__": {value: "1"}})
	if parser.parse() != 1 {
		t.Fatal("expected defined(__linux__) to evaluate to true")
	}
	parser = newParser("defined(_WIN32)", map[string]macro{})
	if parser.parse() != 0 {
		t.Fatal("expected defined(_WIN32) to evaluate to false")
	}
}

func TestConvertToConfig(t *testing.T) {
	proc := NewProcessor(Options{})
	defs, err := proc.ConvertToConfig(map[string]string{"OUTPUT": "app", "MODE": "raw"})
	if err != nil {
		t.Fatal(err)
	}
	if defs["OUTPUT"] != "app" {
		t.Fatalf("OUTPUT = %q, want app", defs["OUTPUT"])
	}
}

func TestProcessConditionals(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.fz")
	if err := os.WriteFile(cfgPath, []byte("#define FEATURE 1\n#ifdef FEATURE\n#define OUTPUT app\n#elif FEATURE == 2\n#define OUTPUT alt\n#else\n#define OUTPUT fallback\n#endif\n#if FEATURE > 0\n#define MODE fast\n#endif\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proc := NewProcessor(Options{RootDir: dir})
	_, err := proc.Process(cfgPath, Options{RootDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := proc.macros["OUTPUT"]; got != "app" {
		t.Fatalf("OUTPUT macro = %q, want app", got)
	}
	if got := proc.macros["MODE"]; got != "fast" {
		t.Fatalf("MODE macro = %q, want fast", got)
	}
}

func TestProcessIncludeCycleDetection(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.fz")
	second := filepath.Join(dir, "second.fz")
	if err := os.WriteFile(first, []byte("#include \"second.fz\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("#include \"first.fz\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proc := NewProcessor(Options{RootDir: dir})
	_, err := proc.Process(first, Options{RootDir: dir})
	if err == nil {
		t.Fatal("expected include cycle error")
	}
}

func TestProcessUsesCache(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.fz")
	if err := os.WriteFile(cfgPath, []byte("#define OUTPUT app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proc := NewProcessor(Options{RootDir: dir})
	if _, err := proc.Process(cfgPath, Options{RootDir: dir}); err != nil {
		t.Fatal(err)
	}
	if len(proc.cache) != 1 {
		t.Fatalf("expected one cached entry, got %d", len(proc.cache))
	}
	if _, err := proc.Process(cfgPath, Options{RootDir: dir}); err != nil {
		t.Fatal(err)
	}
	if len(proc.cache) != 1 {
		t.Fatalf("expected cache entry to remain one, got %d", len(proc.cache))
	}
}
