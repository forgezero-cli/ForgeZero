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

	fzvfs "github.com/forgezero-cli/ForgeZero/internal/fs"
)

func TestRemovePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gone.txt")
	if err := os.WriteFile(path, []byte("x"), FilePerm); err != nil {
		t.Fatal(err)
	}
	if err := RemovePath(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file should be gone")
	}
}

func TestOpenVerifiedRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "r.txt")
	if err := os.WriteFile(path, []byte("data"), FilePerm); err != nil {
		t.Fatal(err)
	}
	f, err := OpenVerifiedRead(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestLookExecutableExported(t *testing.T) {
	_, err := LookExecutable("go")
	if err != nil {
		t.Skip("go not in path")
	}
}

func TestRemovePathMockFail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x")
	m := fzvfs.NewMock(fzvfs.Default)
	resolved, _ := ResolveSecurePath(path)
	m.SetFail("Remove", resolved, fzvfs.ErrPermission)
	SetFileSystem(m)
	defer ResetFileSystem()
	if err := RemovePath(path); err == nil {
		t.Fatal("expected error")
	}
}
