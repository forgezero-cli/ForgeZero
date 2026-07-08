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

func TestLoadUsesCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fz.toml")
	content := `output = "mybin"
mode = "raw"
debug = true
[flags]
cc = ["-O2"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Output = "changed"
	cached, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cached.Output != "mybin" {
		t.Fatal("cache should return cloned config so modifications do not leak")
	}
}

func BenchmarkLoadTOMLConfigUncached(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, ".fz.toml")
	content := `output = "mybin"
mode = "raw"
debug = true
[flags]
cc = ["-O2"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		clearConfigCache()
		if _, err := Load(path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadTOMLConfigCached(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, ".fz.toml")
	content := `output = "mybin"
mode = "raw"
debug = true
[flags]
cc = ["-O2"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Load(path); err != nil {
			b.Fatal(err)
		}
	}
}
