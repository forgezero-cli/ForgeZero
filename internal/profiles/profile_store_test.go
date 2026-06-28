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

package profiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndReadProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".profile.config")

	if err := SaveProfile(path, "performance"); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}
	p, err := ReadSavedProfile(path)
	if err != nil {
		t.Fatalf("ReadSavedProfile failed: %v", err)
	}
	if p != "performance" {
		t.Fatalf("expected performance, got %q", p)
	}
}

func TestReadSavedProfile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".profile.config")
	if err := os.WriteFile(path, []byte("\n\t"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	p, err := ReadSavedProfile(path)
	if err != nil {
		t.Fatalf("ReadSavedProfile failed: %v", err)
	}
	if p != "" {
		t.Fatalf("expected empty profile, got %q", p)
	}
}
