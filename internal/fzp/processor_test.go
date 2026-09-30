/*
 *   Copyright (c) 2026 forgezero-cli
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
	"strings"
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

func BenchmarkProcessLargeSource(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "source.fz")
	data := strings.Repeat("int value(void) { return 1; }\n", 1024)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		b.Fatal(err)
	}
	proc := NewProcessor(Options{RootDir: dir})
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for b.Loop() {
		clear(proc.cache)
		if _, err := proc.Process(path, Options{RootDir: dir}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFZPBlockScannerAVX2(b *testing.B) {
	data := []byte(strings.Repeat("int value(void) { return 1; }\n", 1024))
	masks := make([]scanBlock, len(data)/32)
	offsets := make([]uint16, len(data))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		scanFZPBlocks(data, masks, offsets)
	}
}

func BenchmarkFZPBlockScannerGeneric(b *testing.B) {
	data := []byte(strings.Repeat("int value(void) { return 1; }\n", 1024))
	masks := make([]scanBlock, len(data)/32)
	offsets := make([]uint16, len(data))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		scanFZPBlocksGeneric(data, masks, offsets)
	}
}

func TestFZPBlockScanner(t *testing.T) {
	data := make([]byte, 64)
	data[3] = '\n'
	data[6] = '#'
	data[10] = ' '
	data[11] = '\t'
	data[12] = '\r'
	data[32] = '\n'
	data[63] = '#'
	var got [2]scanBlock
	var gotOffsets [64]uint16
	blocks, newlineCount := scanFZPBlocks(data, got[:], gotOffsets[:])
	if blocks != len(got) {
		t.Fatalf("scanned blocks = %d, want %d", blocks, len(got))
	}
	var want [2]scanBlock
	var wantOffsets [64]uint16
	wantBlocks, wantNewlines := scanFZPBlocksGeneric(data, want[:], wantOffsets[:])
	if wantBlocks != len(want) {
		t.Fatalf("generic blocks = %d, want %d", wantBlocks, len(want))
	}
	if got != want {
		t.Fatalf("SIMD masks = %#v, want %#v", got, want)
	}
	if newlineCount != wantNewlines {
		t.Fatalf("SIMD newline count = %d, want %d", newlineCount, wantNewlines)
	}
	for index := 0; index < newlineCount; index++ {
		if gotOffsets[index] != wantOffsets[index] {
			t.Fatalf("SIMD newline offset %d = %d, want %d", index, gotOffsets[index], wantOffsets[index])
		}
	}
}

func TestFZPBlockScannerAllocs(t *testing.T) {
	data := make([]byte, 32*64)
	var masks [64]scanBlock
	var offsets [32 * 64]uint16
	allocs := testing.AllocsPerRun(100, func() {
		scanFZPBlocks(data, masks[:], offsets[:])
	})
	if allocs != 0 {
		t.Fatalf("scanner allocations = %v, want 0", allocs)
	}
}
