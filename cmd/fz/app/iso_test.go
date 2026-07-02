/*
 * Copyright (c) 2026 ForgeZero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package app

import (
	"testing"

	"github.com/forgezero-cli/ForgeZero/cmd/fz/cli"
	"github.com/forgezero-cli/ForgeZero/internal/config"
)

func TestISORequested(t *testing.T) {
	if ISORequested(nil, nil) {
		t.Error("expected false for nil inputs")
	}
	f := &cli.Flags{}
	f.ISO.Enabled = true
	if !ISORequested(f, nil) {
		t.Error("expected true when flag enabled")
	}
	cfg := &config.Config{}
	cfg.ISO.Enabled = true
	if !ISORequested(&cli.Flags{}, cfg) {
		t.Error("expected true when config enabled")
	}
}

func TestBuildISOOptionsFlagOverridesConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.ISO.SourceDir = "cfgdir"
	cfg.ISO.Output = "cfg.iso"
	cfg.ISO.Joliet = true
	f := &cli.Flags{IsoOut: "flag.iso", IsoHybrid: true}
	f.ISO.Enabled = true
	f.ISO.Dir = "flagdir"

	opts := BuildISOOptions(f, cfg)
	if opts.SourceDir != "flagdir" {
		t.Errorf("expected flagdir, got %q", opts.SourceDir)
	}
	if opts.OutputPath != "flag.iso" {
		t.Errorf("expected flag.iso, got %q", opts.OutputPath)
	}
	if !opts.Hybrid {
		t.Error("expected hybrid from flag")
	}
	if !opts.Joliet {
		t.Error("expected joliet from config")
	}
}

func TestBuildISOOptionsDefaults(t *testing.T) {
	opts := BuildISOOptions(&cli.Flags{}, &config.Config{})
	if opts.SourceDir != "." {
		t.Errorf("expected default source dir '.', got %q", opts.SourceDir)
	}
}

func TestBuildISOOptionsSourceFromConfigOutput(t *testing.T) {
	cfg := &config.Config{Output: "app", SourceDir: "src"}
	cfg.ISO.Enabled = true
	opts := BuildISOOptions(&cli.Flags{}, cfg)
	if opts.SourceDir != "src" {
		t.Errorf("expected src, got %q", opts.SourceDir)
	}
	if opts.OutputPath != "app.iso" {
		t.Errorf("expected app.iso, got %q", opts.OutputPath)
	}
}

func TestHandleISONoop(t *testing.T) {
	if err := HandleISO(&cli.Flags{}, &config.Config{}); err != nil {
		t.Errorf("expected no-op nil, got %v", err)
	}
}
