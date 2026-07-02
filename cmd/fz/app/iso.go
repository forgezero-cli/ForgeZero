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
	"github.com/forgezero-cli/ForgeZero/cmd/fz/cli"
	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/iso/core"
)   

func ISORequested(flags *cli.Flags, cfg *config.Config) bool {
	if flags != nil && flags.ISO.Enabled {
		return true
	}
	return cfg != nil && cfg.ISO.Enabled
}

func BuildISOOptions(flags *cli.Flags, cfg *config.Config) core.Options {
	var ic config.ISOConfig
	if cfg != nil {
		ic = cfg.ISO
	}
	opts := core.Options{
		SourceDir:     ic.SourceDir,
		OutputPath:    ic.Output,
		VolumeID:      ic.VolumeID,
		BootImage:     ic.BootImage,
		BootCatalog:   ic.BootCatalog,
		BootLoadSize:  ic.BootLoadSize,
		NoEmulBoot:    ic.NoEmulBoot,
		BootInfoTable: ic.BootInfoTable,
		Joliet:        ic.Joliet,
		RockRidge:     ic.RockRidge,
		Hybrid:        ic.Hybrid,
		CustomArgs:    ic.CustomArgs,
	}
	if flags != nil {
		if flags.ISO.Dir != "" {
			opts.SourceDir = flags.ISO.Dir
		}
		if flags.IsoOut != "" {
			opts.OutputPath = flags.IsoOut
		}
		if flags.IsoHybrid {
			opts.Hybrid = true
		}
	}
	if opts.SourceDir == "" && cfg != nil {
		if cfg.SourceDir != "" {
			opts.SourceDir = cfg.SourceDir
		} else if flags != nil && flags.DirPath != "" {
			opts.SourceDir = flags.DirPath
		}
	}
	if opts.SourceDir == "" && flags != nil && flags.DirPath != "" {
		opts.SourceDir = flags.DirPath
	}
	if opts.SourceDir == "" {
		opts.SourceDir = "."
	}
	if opts.OutputPath == "" && cfg != nil && cfg.Output != "" {
		opts.OutputPath = cfg.Output + ".iso"
	}
	return opts
}

func HandleISO(flags *cli.Flags, cfg *config.Config) error {
	if !ISORequested(flags, cfg) {
		return nil
	}
	return core.Build(BuildISOOptions(flags, cfg))
}
