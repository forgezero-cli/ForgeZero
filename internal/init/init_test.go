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

package initpkg

import (
	"os"
	"testing"
)

func TestRunCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change to test dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
	if err := Run(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(".fz.toml"); err != nil {
		t.Error(".fz.toml not created")
	}
	if _, err := os.Stat(".fzignore"); err != nil {
		t.Error(".fzignore not created")
	}
	if _, err := os.Stat("README.md"); err != nil {
		t.Error("README.md not created")
	}
	if _, err := os.Stat("configure.fz"); err != nil {
		t.Error("configure.fz not created")
	}
}

func TestRunFailsIfFilesExist(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change to test directory: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	f, err := os.Create(".fz.toml")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Run(); err == nil {
		t.Error("expected error because .fz.toml exists")
	}
}
