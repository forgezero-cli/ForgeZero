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

func TestMockInjectedErrors(t *testing.T) {
	dir := t.TempDir()
	m := NewMock(Default)
	m.SetFailOp("MkdirAll", ErrDiskFull)
	if err := m.MkdirAll(filepath.Join(dir, "x"), 0o700); err != ErrDiskFull {
		t.Fatalf("got %v", err)
	}
}

func TestOpenVerifiedSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "t")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "l")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("no symlink")
	}
	_, err := Default.OpenVerified(link)
	if err != ErrSymlink {
		t.Fatalf("got %v", err)
	}
}
