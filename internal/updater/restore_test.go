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

package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallVersionEmpty(t *testing.T) {
	if err := InstallVersion("   "); err == nil {
		t.Fatal("expected error for empty version")
	}
}

func TestRestoreBackupNoBackup(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "fz")
	if err := os.WriteFile(exe, []byte("current"), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := executablePathFunc
	executablePathFunc = func() (string, error) { return exe, nil }
	defer func() { executablePathFunc = orig }()

	if err := RestoreBackup(); err == nil {
		t.Fatal("expected error when no backup present")
	}
}

func TestRestoreBackupSwaps(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "fz")
	if err := os.WriteFile(exe, []byte("current"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe+".old", []byte("previous"), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := executablePathFunc
	executablePathFunc = func() (string, error) { return exe, nil }
	defer func() { executablePathFunc = orig }()

	if err := RestoreBackup(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "previous" {
		t.Errorf("expected restored 'previous', got %q", string(got))
	}
	old, err := os.ReadFile(exe + ".old")
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != "current" {
		t.Errorf("expected rotated backup 'current', got %q", string(old))
	}
}
