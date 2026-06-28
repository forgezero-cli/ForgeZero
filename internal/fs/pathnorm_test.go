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
	"path/filepath"
	"testing"
)

func TestCleanPathDrive(t *testing.T) {
	got := CleanPath(`C:\project\src\main.go`)
	if got == "" {
		t.Fatal("empty path")
	}
	if !HasDrivePrefix(got) {
		t.Fatalf("expected drive prefix in %q", got)
	}
	if filepath.Base(filepath.FromSlash(`C:/project/src/main.go`)) != "main.go" {
		t.Fatal("sanity check failed")
	}
}

func TestCleanPathUNC(t *testing.T) {
	unc := `\\server\share\dir\file.txt`
	got := CleanPath(unc)
	if !IsUNC(got) {
		t.Fatalf("expected UNC, got %q", got)
	}
}

func TestHasDrivePrefix(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{`D:\build`, true},
		{`/unix`, false},
		{`C:relative`, true},
	}
	for _, tc := range cases {
		if HasDrivePrefix(tc.path) != tc.want {
			t.Fatalf("%q: got %v want %v", tc.path, !tc.want, tc.want)
		}
	}
}

func TestNormalizeAbsRelative(t *testing.T) {
	dir := t.TempDir()
	abs, err := NormalizeAbs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if abs == "" {
		t.Fatal("empty abs")
	}
}
