//go:build linux

/*
 * Copyright (c) 2026 forgezero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSourceFilesLinux(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(nested, "source.c")
	if err := os.WriteFile(file, []byte("int main(void) {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "source-link.c")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(root, "nested-link")
	if err := os.Symlink(nested, linkDir); err != nil {
		t.Fatal(err)
	}
	files, err := discoverSourceFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("discovered files = %d, want 2", len(files))
	}
	if files[0].path > files[1].path {
		t.Fatalf("discovery order is unstable: %q before %q", files[0].path, files[1].path)
	}
	for _, discovered := range files {
		if discovered.inode == 0 || discovered.size <= 0 || discovered.modTime == 0 {
			t.Fatalf("invalid metadata for %s", discovered.path)
		}
	}
}

func BenchmarkDiscoverSourceFilesLinux(b *testing.B) {
	root := b.TempDir()
	for directory := 0; directory < 8; directory++ {
		dir := filepath.Join(root, "dir", string(rune('a'+directory)))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			b.Fatal(err)
		}
		for file := 0; file < 128; file++ {
			path := filepath.Join(dir, "source-"+string(rune('a'+file/26))+string(rune('a'+file%26))+".c")
			if err := os.WriteFile(path, []byte("int value(void) { return 1; }\n"), 0o644); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		files, err := discoverSourceFiles(root)
		if err != nil {
			b.Fatal(err)
		}
		if len(files) != 8*128 {
			b.Fatalf("discovered files = %d", len(files))
		}
	}
}
