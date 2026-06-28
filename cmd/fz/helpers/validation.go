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

package helpers

import (
	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func ValidateAll(flags *Flags) bool {
	if err := utils.ValidateCLIPath(flags.ConfigPath); err != nil {
		WriteFmt(2, "invalid config path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.GloriaPath); err != nil {
		WriteFmt(2, "error: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.AsmPath); err != nil {
		WriteFmt(2, "invalid asm path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.CcPath); err != nil {
		WriteFmt(2, "invalid cc path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.DirPath); err != nil {
		WriteFmt(2, "invalid dir path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.OutBin); err != nil {
		WriteFmt(2, "invalid output path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.OutObj); err != nil {
		WriteFmt(2, "invalid object output path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.LdScript); err != nil {
		WriteFmt(2, "invalid linker script path: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIPath(flags.TextAddr); err != nil {
		WriteFmt(2, "invalid text address: %v\n", err)
		return false
	}
	if flags.PluginPath != "" {
		if err := utils.ValidateCLIPath(flags.PluginPath); err != nil {
			WriteFmt(2, "invalid plugin path: %v\n", err)
			return false
		}
	}
	if err := utils.ValidateCLIArg(flags.Mode); err != nil {
		WriteFmt(2, "invalid mode: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIArg(flags.Format); err != nil {
		WriteFmt(2, "invalid format: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIArg(flags.Target); err != nil {
		WriteFmt(2, "invalid target: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIArg(flags.Toolchain); err != nil {
		WriteFmt(2, "invalid toolchain: %v\n", err)
		return false
	}
	if err := utils.ValidateCLIArg(flags.Isolation); err != nil {
		WriteFmt(2, "invalid isolation: %v\n", err)
		return false
	}
	if flags.Isolation != "none" && flags.Isolation != "standard" && flags.Isolation != "strict" {
		WriteFmt(2, "%s\n", "error: -isolation must be none, standard, or strict")
		return false
	}
	if err := utils.ValidateCLIArg(flags.BuildType); err != nil {
		WriteFmt(2, "invalid build type: %v\n", err)
		return false
	}
	if _, err := utils.ValidateFlagTokens([]byte(flags.CcFlags)); err != nil {
		WriteFmt(2, "invalid C compiler flags: %v\n", err)
		return false
	}
	if _, err := utils.ValidateFlagTokens([]byte(flags.LdFlags)); err != nil {
		WriteFmt(2, "invalid linker flags: %v\n", err)
		return false
	}
	if flags.Mode != "" && flags.Mode != "auto" && flags.Mode != "c" && flags.Mode != "raw" {
		WriteFmt(2, "%s\n", "error: -mode must be auto, c, or raw")
		return false
	}
	if flags.Toolchain != "" {
		if !config.IsValidToolchain(flags.Toolchain) {
			WriteFmt(2, "%s\n", "error: -toolchain must be one of auto, zig, fasm, nasm, gas, gcc, clang, ld")
			return false
		}
	}
	return true
}