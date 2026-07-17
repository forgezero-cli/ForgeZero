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

package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/config"
)

func writeCSource(t *testing.T, dir, name, content string) string {
	src := filepath.Join(dir, name+".c")
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, name+".o")
	if out, err := exec.Command("gcc", "-c", src, "-o", obj).CombinedOutput(); err != nil {
		t.Skipf("gcc -c failed: %v\n%s", err, out)
	}
	return obj
}

func TestConfigSpecifiesBuild(t *testing.T) {
	cases := []struct {
		name string
		cfg  *config.Config
		want bool
	}{
		{"nil config", nil, false},
		{"config only", &config.Config{ConfigOnly: true}, true},
		{"parse makefile", &config.Config{ParseMakefile: true}, true},
		{"source file", &config.Config{SourceFile: "main.c"}, true},
		{"source dir", &config.Config{SourceDir: "src"}, true},
		{"source files", &config.Config{SourceFiles: []string{"a.c"}}, true},
		{"build rules", &config.Config{BuildRules: []config.BuildRule{{Name: "x"}}}, true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := configSpecifiesBuild(tt.cfg)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestSkipPlatformSource(t *testing.T) {
	cases := []struct {
		path     string
		targetOS string
		want     bool
	}{
		{"src/ngx_linux_file.c", "linux", false},
		{"src/ngx_linux_file.c", "darwin", true},
		{"src/ngx_darwin_file.c", "darwin", false},
		{"src/os/win32/foo.c", "windows", false},
		{"src/os/win32/foo.c", "linux", true},
		{"src/os/unix/foo.c", "windows", true},
		{"src/os/unix/foo.c", "linux", false},
	}

	for _, tt := range cases {
		if got := skipPlatformSource(tt.path, tt.targetOS); got != tt.want {
			t.Fatalf("skipPlatformSource(%q, %q) = %v, want %v", tt.path, tt.targetOS, got, tt.want)
		}
	}
}

func TestFilterPlatformSpecificSourcesPlatformPaths(t *testing.T) {
	srcFiles := []string{"src/ngx_linux_main.c", "src/ngx_darwin_main.c", "src/common.c"}
	got := filterPlatformSpecificSources(srcFiles, "linux")
	if len(got) != 2 || !strings.Contains(strings.Join(got, ","), "ngx_linux_main.c") || !strings.Contains(strings.Join(got, ","), "common.c") {
		t.Fatalf("unexpected filtered sources: %v", got)
	}
}

func TestFindIncludedSourceFiles(t *testing.T) {
	dir := t.TempDir()
	mainSrc := filepath.Join(dir, "main.c")
	included := filepath.Join(dir, "sub.c")
	content := `#include "sub.c"
// #include "ignored.c"
#include "missing.txt"
`
	if err := os.WriteFile(mainSrc, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(included, []byte("int sub() { return 0; }"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := findIncludedSourceFiles([]string{mainSrc})
	if len(result) != 1 {
		t.Fatalf("expected 1 included source, got %v", result)
	}
	if _, ok := result[filepath.Clean(included)]; !ok {
		t.Fatalf("expected included source %s, got %v", included, result)
	}
}

func TestDiscoverSourceIncludeDirs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(src, "one"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "two", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "one", "header.h"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "two", "nested", "header.hpp"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	result := discoverSourceIncludeDirs(dir)
	if len(result) != 2 {
		t.Fatalf("expected 2 include dirs, got %v", result)
	}
	if !containsPath(result, filepath.Join(src, "one")) || !containsPath(result, filepath.Join(src, "two", "nested")) {
		t.Fatalf("unexpected include dirs %v", result)
	}
}

func TestDiscoverDependencyIncludeDirs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(src)
	if err := os.WriteFile(filepath.Join(parent, "common.h"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "deps", "lib", "include"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "deps", "lib", "include", "lib.h"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	result := discoverDependencyIncludeDirs(src)
	if !containsPath(result, parent) || !containsPath(result, filepath.Join(parent, "deps", "lib", "include")) {
		t.Fatalf("unexpected dependency include dirs %v", result)
	}
}

func TestBuildDirParallelStress(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	const count = 24
	var decls strings.Builder
	var calls strings.Builder
	for i := 0; i < count; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.c", i)), []byte(fmt.Sprintf("int f%d(void) { return %d; }\n", i, i)), 0o644); err != nil {
			t.Fatal(err)
		}
		decls.WriteString(fmt.Sprintf("int f%d(void);\n", i))
		calls.WriteString(fmt.Sprintf("    sum += f%d();\n", i))
	}
	mainSrc := decls.String() + "\nint main(void) {\n    int sum = 0;\n" + calls.String() + "    return sum;\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	outBin := filepath.Join(t.TempDir(), "stress")
	res, err := BuildDir(context.Background(), []string{dir}, outBin, false, false, "c", false, true, false, false, false, nil, nil, nil, nil, nil, 8, "executable")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(res.Binary); err != nil {
		t.Fatalf("binary missing: %v", err)
	}
	if len(res.ObjectFiles) != count+1 {
		t.Fatalf("expected %d object files, got %d", count+1, len(res.ObjectFiles))
	}
}

func TestBuildDirSkipsIncludedSources(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	helper := filepath.Join(dir, "helper.c")
	if err := os.WriteFile(helper, []byte("int helper(void) { return 1; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainSrc := `#include "helper.c"
int main(void) {
    return helper();
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	outBin := filepath.Join(t.TempDir(), "included")
	res, err := BuildDir(context.Background(), []string{dir}, outBin, false, false, "c", false, true, false, false, false, nil, nil, nil, nil, nil, 2, "executable")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ObjectFiles) != 1 {
		t.Fatalf("expected 1 object file, got %d: %v", len(res.ObjectFiles), res.ObjectFiles)
	}
	if _, err := os.Stat(res.Binary); err != nil {
		t.Fatalf("binary missing: %v", err)
	}
}

func containsPath(list []string, path string) bool {
	for _, item := range list {
		if filepath.Clean(item) == filepath.Clean(path) {
			return true
		}
	}
	return false
}
