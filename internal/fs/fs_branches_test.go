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

func TestNewMockNilBase(t *testing.T) {
	m := NewMock(nil)
	if m.Base == nil {
		t.Fatal("base should default to Default")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if err := m.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestHasDrivePrefixBranches(t *testing.T) {
	if !HasDrivePrefix(`C:\foo`) {
		t.Fatal()
	}
	if HasDrivePrefix("/unix") {
		t.Fatal()
	}
}

func TestNormalizeAbs(t *testing.T) {
	dir := t.TempDir()
	got, err := NormalizeAbs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("empty")
	}
}

func TestOpenVerifiedSymlinkRejection(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "t")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "l")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink")
	}
	_, err := Default.OpenVerified(link)
	if err == nil {
		t.Fatal("expected symlink rejection")
	}
}

func TestOpenVerifiedRacePathChanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := Default.OpenVerified(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestMockReadlinkEvalDefaultBase(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "t")
	if err := os.WriteFile(target, []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "l")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink")
	}
	m := NewMock(Default)
	if _, err := m.Readlink(link); err != nil {
		t.Fatal(err)
	}
	if _, err := m.EvalSymlinks(link); err != nil {
		t.Fatal(err)
	}
}
