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

package reverse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReverseMakefile(t *testing.T) {
	tmp := t.TempDir()
	mk := filepath.Join(tmp, "Makefile")
	data := []byte(`
CC=gcc
CFLAGS=-Wall -O2 -Iinclude -DDEBUG=1
LDFLAGS=-lm -lz
TARGET=myapp
SRCS=main.c utils.c

all: $(TARGET)

$(TARGET): $(SRCS)
	$(CC) $(CFLAGS) -o $@ $^ $(LDFLAGS)

clean:
	rm -f $(TARGET)
`)
	if err := os.WriteFile(mk, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReverseMakefile(mk)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "myapp" {
		t.Errorf("expected output myapp, got %s", cfg.Output)
	}
	if len(cfg.SourceFiles) != 2 || cfg.SourceFiles[0] != "main.c" || cfg.SourceFiles[1] != "utils.c" {
		t.Errorf("source files mismatch: %v", cfg.SourceFiles)
	}
	if len(cfg.Include) != 1 || cfg.Include[0] != "include" {
		t.Errorf("include mismatch: %v", cfg.Include)
	}
	if len(cfg.Libs) != 2 || cfg.Libs[0] != "m" || cfg.Libs[1] != "z" {
		t.Errorf("libs mismatch: %v", cfg.Libs)
	}
	if len(cfg.Flags.Cc) == 0 {
		t.Error("expected Cc flags")
	}
}

func TestReverseMakefileWithVariables(t *testing.T) {
	tmp := t.TempDir()
	mk := filepath.Join(tmp, "Makefile")
	data := []byte(`
CC = clang
CFLAGS = -g -O0
LDFLAGS = -lfoo
SRC = foo.c bar.c
OUT = prog
TARGET = $(OUT)

all: $(TARGET)
	$(CC) $(CFLAGS) -o $@ $(SRC) $(LDFLAGS)
`)
	if err := os.WriteFile(mk, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReverseMakefile(mk)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "prog" {
		t.Errorf("expected output prog, got %s", cfg.Output)
	}
	if len(cfg.SourceFiles) != 2 {
		t.Errorf("expected 2 sources, got %d", len(cfg.SourceFiles))
	}
	if len(cfg.Flags.Cc) == 0 {
		t.Error("expected Cc flags")
	}
}

func TestReverseCMake(t *testing.T) {
	tmp := t.TempDir()
	cm := filepath.Join(tmp, "CMakeLists.txt")
	data := []byte(`
project(MyApp)
add_executable(myapp main.c utils.c)
target_include_directories(myapp PRIVATE include)
target_link_libraries(myapp m z)
set(CMAKE_C_FLAGS "-Wall -O2")
`)
	if err := os.WriteFile(cm, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReverseCMake(cm)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "myapp" {
		t.Errorf("expected output myapp, got %s", cfg.Output)
	}
	if len(cfg.SourceFiles) != 2 || cfg.SourceFiles[0] != "main.c" || cfg.SourceFiles[1] != "utils.c" {
		t.Errorf("source files mismatch: %v", cfg.SourceFiles)
	}
	if len(cfg.Include) != 1 || cfg.Include[0] != "include" {
		t.Errorf("include mismatch: %v", cfg.Include)
	}
	if len(cfg.Libs) != 2 || cfg.Libs[0] != "m" || cfg.Libs[1] != "z" {
		t.Errorf("libs mismatch: %v", cfg.Libs)
	}
}

func TestReverseCMakeWithLibrary(t *testing.T) {
	tmp := t.TempDir()
	cm := filepath.Join(tmp, "CMakeLists.txt")
	data := []byte(`
project(MyLib)
add_library(mylib STATIC src1.c src2.c)
target_include_directories(mylib PUBLIC include)
target_link_libraries(mylib pthread)
`)
	if err := os.WriteFile(cm, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReverseCMake(cm)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "libmylib.a" {
		t.Errorf("expected output libmylib.a, got %s", cfg.Output)
	}
	if len(cfg.SourceFiles) != 2 {
		t.Errorf("expected 2 sources, got %d", len(cfg.SourceFiles))
	}
}

func TestReverseFileDetection(t *testing.T) {
	tmp := t.TempDir()
	mk := filepath.Join(tmp, "file.mk")
	data := []byte(`
CC=gcc
TARGET=foo
SRCS=foo.c
`)
	if err := os.WriteFile(mk, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReverseFile(mk)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "foo" {
		t.Errorf("expected output foo, got %s", cfg.Output)
	}

	cm := filepath.Join(tmp, "CMakeLists.txt")
	data2 := []byte(`project(Bar)`)
	if err := os.WriteFile(cm, data2, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg2, err := ReverseFile(cm)
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Name != "Bar" {
		t.Errorf("expected name Bar, got %s", cfg2.Name)
	}
}

func TestReverseFileUnknown(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "unknown.txt")
	data := []byte(`some random text`)
	if err := os.WriteFile(f, data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReverseFile(f)
	if err == nil {
		t.Error("expected error for unknown file")
	}
}

func TestSplitFlags(t *testing.T) {
	cases := []struct {
		in  string
		out []string
	}{
		{"-Wall -O2", []string{"-Wall", "-O2"}},
		{`-D"FOO=bar" -Iinclude`, []string{`-D"FOO=bar"`, "-Iinclude"}},
		{"", nil},
		{"-lfoo", []string{"-lfoo"}},
		{`-D'ABC=123'`, []string{`-D'ABC=123'`}},
	}
	for _, c := range cases {
		res := splitFlags(c.in)
		if len(res) != len(c.out) {
			t.Errorf("splitFlags(%q) length mismatch: %v vs %v", c.in, res, c.out)
			continue
		}
		for i := range res {
			if res[i] != c.out[i] {
				t.Errorf("splitFlags(%q) mismatch at %d: %s vs %s", c.in, i, res[i], c.out[i])
			}
		}
	}
}

func TestExpandVars(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"$(CC)", ""},
		{"${CC}", ""},
		{"$CC", ""},
		{"$(CC) -O2", " -O2"},
		{"foo", "foo"},
		{"$()", "$()"},
	}
	for _, c := range cases {
		res := expandVars(c.in)
		if res != c.out {
			t.Errorf("expandVars(%q) = %q, want %q", c.in, res, c.out)
		}
	}
}

func TestTrimBytes(t *testing.T) {
	cases := []struct {
		in  []byte
		out []byte
	}{
		{[]byte("  foo  "), []byte("foo")},
		{[]byte("\tbar\n"), []byte("bar")},
		{[]byte("nochange"), []byte("nochange")},
		{[]byte(""), []byte("")},
	}
	for _, c := range cases {
		res := trimBytes(c.in)
		if string(res) != string(c.out) {
			t.Errorf("trimBytes(%q) = %q, want %q", c.in, res, c.out)
		}
	}
}

func TestParseDeps(t *testing.T) {
	cases := []struct {
		in  []byte
		out []string
	}{
		{[]byte("foo.c bar.c"), []string{"foo.c", "bar.c"}},
		{[]byte("main.o: main.c"), []string{"main.o:", "main.c"}},
		{[]byte(""), nil},
		{[]byte("foo.c $(VAR)"), []string{"foo.c"}},
	}
	for _, c := range cases {
		res := parseDeps(c.in)
		if len(res) != len(c.out) {
			t.Errorf("parseDeps(%q) length mismatch: %v vs %v", c.in, res, c.out)
			continue
		}
		for i := range res {
			if res[i] != c.out[i] {
				t.Errorf("parseDeps(%q) mismatch at %d: %s vs %s", c.in, i, res[i], c.out[i])
			}
		}
	}
}

func TestToConfigEmpty(t *testing.T) {
	rc := &ReverseConfig{
		Source:  []string{},
		Libs:    []string{},
		Include: []string{},
		Flags:   []string{},
		Defines: map[string]string{},
	}
	cfg := rc.toConfig()
	if cfg.Output != "" {
		t.Errorf("expected empty output, got %s", cfg.Output)
	}
	if cfg.Mode != "auto" {
		t.Errorf("expected auto mode, got %s", cfg.Mode)
	}
}

func TestParseVariable(t *testing.T) {
	rc := &ReverseConfig{
		Source:  make([]string, 0),
		Libs:    make([]string, 0),
		Include: make([]string, 0),
		Flags:   make([]string, 0),
		Defines: make(map[string]string),
	}
	rc.parseVariable([]byte("CFLAGS"), []byte("-Wall -DDEBUG"))
	if len(rc.Flags) == 0 {
		t.Error("expected flags")
	}
	if _, ok := rc.Defines["DEBUG"]; !ok {
		t.Error("expected define DEBUG")
	}
	rc.parseVariable([]byte("TARGET"), []byte("prog"))
	if rc.Target != "prog" {
		t.Errorf("expected target prog, got %s", rc.Target)
	}
}
