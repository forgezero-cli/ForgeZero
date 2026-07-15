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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFZP(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.fz")
	if err := os.WriteFile(cfgPath, []byte("#define OUTPUT app\n#define MODE raw\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFZP(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "app" {
		t.Fatalf("Output = %q, want app", cfg.Output)
	}
	if cfg.Mode != "raw" {
		t.Fatalf("Mode = %q, want raw", cfg.Mode)
	}
}
