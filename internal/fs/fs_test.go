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

package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultMkdirWriteRead(t *testing.T) {
	dir := t.TempDir()
	u := Default
	sub := filepath.Join(dir, "nested")
	if err := u.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sub, "data.bin")
	data := []byte("forgezero")
	if err := u.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	read, err := u.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(read) != string(data) {
		t.Errorf("read = %q, want %q", read, data)
	}
}

func TestDefaultFilesystem(t *testing.T) {
	if Default == nil {
		t.Fatal("Default filesystem is nil")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "probe")
	if err := Default.WriteFile(path, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Default.Stat(path); err != nil {
		t.Fatal(err)
	}
	if err := Default.Remove(path); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultCreateTempRename(t *testing.T) {
	dir := t.TempDir()
	u := Default
	f, err := u.CreateTemp(dir, "fz_*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("temp")); err != nil {
		t.Fatal(err)
	}
	tmpName := f.Name()
	f.Close()
	final := filepath.Join(dir, "final")
	if err := u.Rename(tmpName, final); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(final); err != nil {
		t.Fatal(err)
	}
}
