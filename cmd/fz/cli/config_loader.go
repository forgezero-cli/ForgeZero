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

package cli

import (
	"github.com/forgezero-cli/ForgeZero/cmd/fz/stdio"
	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/profiles"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func LoadConfig(flags *Flags) (*config.Config, string, error) {
	var cfg *config.Config
	var err error

	if flags.ConfigFZPPath != "" {
		cfg, err = config.LoadFZP(flags.ConfigFZPPath)
		if err != nil {
			return nil, "", err
		}
	} else if flags.ConfigPath != "" {
		cfg, err = config.Load(flags.ConfigPath)
		if err != nil {
			return nil, "", err
		}
	} else {
		cfg, err = config.LoadMerged("")
		if err != nil {
			return nil, "", err
		}
	}
	if len(flags.SetOverrides) > 0 {
		if err := cfg.ApplySetOverrides(flags.SetOverrides); err != nil {
			return nil, "", err
		}
	}
	if flags.VerifySignatures {
		if err := cfg.Validate(); err != nil {
			return nil, "", err
		}
	}

	srcPath, err := ValidateSourceFlags(flags, cfg)
	if err != nil {
		return nil, "", err
	}
	flags.SourcePath = srcPath

	return cfg, srcPath, nil
}

func ApplyConfigToFlags(cfg *config.Config, flags *Flags) {
	if cfg == nil {
		return
	}

	for k, v := range cfg.ToolChecksums {
		utils.ToolChecksums.Store(k, v)
	}

	cfg.MergeFromFlags(flags.SourcePath, flags.DirPath, flags.OutBin, flags.OutObj,
		flags.Debug, flags.Verbose, flags.KeepObj, flags.NoCache,
		flags.Mode, flags.Toolchain, flags.Isolation)

	cfg.Profile = flags.ProfileFlag
	p := profiles.ParseUserProfile(cfg.Profile)
	flags.Jobs = p.EffectiveJobs(flags.Jobs)

	if cfg.OptimizationLevel == 0 {
		switch p.Name {
		case "performance":
			cfg.OptimizationLevel = 4
		case "power-saver":
			cfg.OptimizationLevel = 1
		default:
			cfg.OptimizationLevel = 2
		}
	}

	utils.SetToolchainPolicy(cfg.Toolchain)

	if flags.Verbose && !flags.JSONOutput {
		stdio.WriteFmt(1, "Profile: %s\n", flags.ProfileFlag)
		stdio.WriteFmt(1, "Loaded config from %s\n", func() string {
			if flags.ConfigPath != "" {
				return flags.ConfigPath
			}
			return config.DefaultConfigPath()
		}())
	}
}
