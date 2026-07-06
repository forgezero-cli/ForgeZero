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

package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDepFilePath(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.c")
	if err := os.WriteFile(src, []byte("int main() {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	dp := filepath.Join(dir, "main.d")
	depData := "main.o: main.c \\\n header.h header2.h \\\n    subdir/header3.h\n"
	if err := os.WriteFile(dp, []byte(depData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "subdir", "header3.h"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := ParseDepFilePath(dp)
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 4 {
		t.Fatalf("expected 4 deps, got %d", len(d))
	}
	if filepath.Base(d[0]) != "main.c" || filepath.Base(d[1]) != "header.h" || filepath.Base(d[2]) != "header2.h" || filepath.Base(d[3]) != "header3.h" {
		t.Fatalf("unexpected deps: %v", d)
	}
}
