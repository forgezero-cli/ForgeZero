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

package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnoreFile(t *testing.T) {
	dir := t.TempDir()
	ignorePath := filepath.Join(dir, ".fzignore")
	content := `
# comment
*.o
temp/
`
	err := os.WriteFile(ignorePath, []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	matcher, err := LoadIgnoreFile(ignorePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(matcher.patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(matcher.patterns))
	}
	if !matcher.Match("test.o") {
		t.Error("*.o should match test.o")
	}
	if !matcher.Match("temp/file.asm") {
		t.Error("temp/ should match temp/file.asm")
	}
}

func TestMatch(t *testing.T) {
	matcher := &IgnoreMatcher{patterns: []string{"*.asm", "test_*", "build/"}}
	if !matcher.Match("main.asm") {
		t.Error("*.asm should match main.asm")
	}
	if !matcher.Match("test_something") {
		t.Error("test_* should match test_something")
	}
	if !matcher.Match("build/output.o") {
		t.Error("build/ should match build/output.o")
	}
	if matcher.Match("main.c") {
		t.Error("main.c should not match")
	}
}
