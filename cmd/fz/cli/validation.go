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
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func ValidateAll(flags *Flags) bool {
	if err := utils.ValidateCLIPath(flags.ConfigPath); err != nil {
		stdio.WriteFmt(2, "invalid config path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.GloriaPath); err != nil {
		stdio.WriteFmt(2, "error: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.AsmPath); err != nil {
		stdio.WriteFmt(2, "invalid asm path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.CcPath); err != nil {
		stdio.WriteFmt(2, "invalid cc path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.DirPath); err != nil {
		stdio.WriteFmt(2, "invalid dir path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.OutBin); err != nil {
		stdio.WriteFmt(2, "invalid output path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.OutObj); err != nil {
		stdio.WriteFmt(2, "invalid object output path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.LdScript); err != nil {
		stdio.WriteFmt(2, "invalid linker script path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.TextAddr); err != nil {
		stdio.WriteFmt(2, "invalid text address: %v\n", err)
		return false
	}
	if flags.PluginPath != "" {
		if err := utils.ValidateCLIPath(flags.PluginPath); err != nil {
			stdio.WriteFmt(2, "invalid plugin path: %v\n", err)
			return false
		}
	}
	if err := utils.ValidateCLIArg(flags.Mode); err != nil {
		stdio.WriteFmt(2, "invalid mode: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIArg(flags.Format); err != nil {
		stdio.WriteFmt(2, "invalid format: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIArg(flags.Target); err != nil {
		stdio.WriteFmt(2, "invalid target: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIArg(flags.Toolchain); err != nil {
		stdio.WriteFmt(2, "invalid toolchain: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIArg(flags.Isolation); err != nil {
		stdio.WriteFmt(2, "invalid isolation: %v\n", err)
		return false
	}
	if flags.Isolation != "none" && flags.Isolation != "standard" && flags.Isolation != "strict" {
		stdio.WriteFmt(2, "%s\n", "error: -isolation must be none, standard, or strict")
		return false
	}
	if err := utils.ValidateCLIArg(flags.BuildType); err != nil {
		stdio.WriteFmt(2, "invalid build type: %v\n", err)
		return false
	}
	if _, err := utils.ValidateFlagTokens([]byte(flags.CcFlags)); err != nil {
		stdio.WriteFmt(2, "invalid C compiler flags: %v\n", err)
		return false
	}
	if _, err := utils.ValidateFlagTokens([]byte(flags.LdFlags)); err != nil {
		stdio.WriteFmt(2, "invalid linker flags: %v\n", err)
		return false
	}
	if flags.Mode != "" && flags.Mode != "auto" && flags.Mode != "c" && flags.Mode != "raw" {
		stdio.WriteFmt(2, "%s\n", "error: -mode must be auto, c, or raw")
		return false
	}
	if flags.Toolchain != "" {
		if !config.IsValidToolchain(flags.Toolchain) {
			stdio.WriteFmt(2, "%s\n", "error: -toolchain must be one of auto, zig, fasm, nasm, gas, gcc, clang, ld")
			return false
		}
	}
	return true
}

func ValidateSourceFlags(flags *Flags, cfg *config.Config) (string, error) {
	srcProvided := 0
	var srcPath string

	if flags.AsmPath != "" {
		srcProvided++
		srcPath = flags.AsmPath
	}
	if flags.CcPath != "" {
		srcProvided++
		srcPath = flags.CcPath
	}
	if flags.GloriaPath != "" {
		srcProvided++
		srcPath = flags.GloriaPath
	}
	if flags.DirPath != "" {
		srcProvided++
	}

	if srcProvided == 0 {
		if cfg != nil {
			if cfg.SourceFile != "" || cfg.SourceDir != "" || len(cfg.SourceDirs) > 0 || len(cfg.SourceFiles) > 0 {
				return "", nil
			}
		}
		return "", stdio.Errorf("missing source: use -asm, -cc, -dir, or config")
	}

	if srcProvided > 1 {
		return "", stdio.Errorf("specify only one of -asm, -cc, -gloria or -dir")
	}

	return srcPath, nil
}