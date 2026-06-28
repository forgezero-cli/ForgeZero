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

package verify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func TestWriteManifestSecurePerm(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "manifest.json")
	file := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(file, []byte("data"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	if err := WriteManifest(manifest, root); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != utils.FilePerm {
		t.Errorf("manifest perm = %o, want %o", info.Mode().Perm(), utils.FilePerm)
	}
}

func TestCollectFilesRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real.txt")
	if err := os.WriteFile(target, []byte("x"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink not supported")
	}
	_, err := collectFiles(root)
	if err == nil {
		t.Error("expected symlink rejection")
	}
}
