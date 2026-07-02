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

func TestISOConfigParse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fz.yaml")
	data := "" +
		"output: app\n" +
		"iso:\n" +
		"  enabled: true\n" +
		"  source_dir: root\n" +
		"  output: live.iso\n" +
		"  volume_id: FZ\n" +
		"  boot_image: boot/grub.img\n" +
		"  joliet: true\n" +
		"  rock_ridge: true\n" +
		"  hybrid: true\n" +
		"  custom_args:\n" +
		"    - -p\n" +
		"    - pub\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.ISO.Enabled || cfg.ISO.SourceDir != "root" || cfg.ISO.Output != "live.iso" {
		t.Errorf("unexpected iso config: %+v", cfg.ISO)
	}
	if cfg.ISO.VolumeID != "FZ" || !cfg.ISO.Joliet || !cfg.ISO.Hybrid {
		t.Errorf("unexpected iso flags: %+v", cfg.ISO)
	}
	if len(cfg.ISO.CustomArgs) != 2 {
		t.Errorf("expected 2 custom args, got %v", cfg.ISO.CustomArgs)
	}
}

func TestISOConfigMerge(t *testing.T) {
	base := &Config{}
	base.ISO.SourceDir = "a"
	base.ISO.Joliet = true

	over := &Config{}
	over.ISO.Enabled = true
	over.ISO.SourceDir = "b"
	over.ISO.Hybrid = true

	base.Merge(over)
	if !base.ISO.Enabled {
		t.Error("expected enabled after merge")
	}
	if base.ISO.SourceDir != "b" {
		t.Errorf("expected source_dir b, got %q", base.ISO.SourceDir)
	}
	if !base.ISO.Joliet {
		t.Error("expected joliet preserved")
	}
	if !base.ISO.Hybrid {
		t.Error("expected hybrid from override")
	}
}

func TestISOConfigExpand(t *testing.T) {
	cfg := &Config{
		Variables: map[string]string{"NAME": "distro"},
	}
	cfg.ISO.Output = "${NAME}.iso"
	cfg.ISO.VolumeID = "${NAME}"
	cfg.expand()
	if cfg.ISO.Output != "distro.iso" {
		t.Errorf("expected distro.iso, got %q", cfg.ISO.Output)
	}
	if cfg.ISO.VolumeID != "distro" {
		t.Errorf("expected distro, got %q", cfg.ISO.VolumeID)
	}
}
