//go:build windows

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

func TestWindowsRenameAtomic(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.tmp")
	newPath := filepath.Join(dir, "new.dat")
	if err := os.WriteFile(old, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := Windows{}
	if err := w.Rename(old, newPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "payload" {
		t.Fatalf("got %q", data)
	}
}

func TestWindowsOpenVerified(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := Windows{}
	f, err := w.OpenVerified(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestWindowsChmodNoPanic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.txt")
	if err := os.WriteFile(path, []byte("z"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (Windows{}).Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestWindowsCleanPathBackslash(t *testing.T) {
	got := CleanPath(`C:\a\b\c`)
	if got == "" {
		t.Fatal("empty")
	}
}

func TestDefaultIsWindows(t *testing.T) {
	if _, ok := Default.(Windows); !ok {
		t.Fatalf("Default type %T", Default)
	}
}
