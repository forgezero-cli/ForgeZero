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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMakefileVars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Makefile")
	data := `CFLAGS = -O2 -Iinclude
CPPFLAGS += -DDEBUG
LDFLAGS = -lm
SRC = src/main.c src/util.c
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}

	vars, err := parseMakefileVars(path)
	if err != nil {
		t.Fatalf("parseMakefileVars failed: %v", err)
	}
	if vars["CFLAGS"] != "-O2 -Iinclude" {
		t.Fatalf("unexpected CFLAGS: %q", vars["CFLAGS"])
	}
	if vars["CPPFLAGS"] != "-DDEBUG" {
		t.Fatalf("unexpected CPPFLAGS: %q", vars["CPPFLAGS"])
	}
	if vars["LDFLAGS"] != "-lm" {
		t.Fatalf("unexpected LDFLAGS: %q", vars["LDFLAGS"])
	}
	if vars["SRC"] != "src/main.c src/util.c" {
		t.Fatalf("unexpected SRC: %q", vars["SRC"])
	}
}

func TestDiscoverMakefileSettingsGeneric(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Makefile")
	data := `CFLAGS=-O2 -Iinclude
CPPFLAGS=-DDEBUG
LDFLAGS=-lm
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}

	includes, cflags, ldflags := discoverMakefileSettings(dir)
	if len(includes) == 0 {
		t.Fatalf("expected discoverMakefileSettings to return include dirs")
	}
	if !strings.Contains(cflags, "-O2") || !strings.Contains(cflags, "-DDEBUG") {
		t.Fatalf("unexpected cflags: %q", cflags)
	}
	if !strings.Contains(ldflags, "-lm") {
		t.Fatalf("unexpected ldflags: %q", ldflags)
	}
}

func TestFilterPlatformSpecificSources(t *testing.T) {
	srcFiles := []string{
		filepath.Join("src", "os", "win32", "ngx_files.c"),
		filepath.Join("src", "os", "unix", "ngx_file.c"),
	}
	filtered := filterPlatformSpecificSources(srcFiles, "x86_64-linux-gnu")
	if len(filtered) != 1 || filtered[0] != srcFiles[1] {
		t.Fatalf("expected only unix source for linux target, got %v", filtered)
	}
	filtered = filterPlatformSpecificSources(srcFiles, "x86_64-windows-gnu")
	if len(filtered) != 1 || filtered[0] != srcFiles[0] {
		t.Fatalf("expected only windows source for windows target, got %v", filtered)
	}
}
