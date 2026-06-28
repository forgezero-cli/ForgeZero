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
	"github.com/forgezero-cli/ForgeZero/internal/assembler"
	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/linker"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func SetupZig(cfg *config.Config, toolchain string) {
	autoEnableZig := true
	
	if cfg != nil && (cfg.Toolchain == "gcc" || cfg.Toolchain == "clang") {
		autoEnableZig = false
	}
	if toolchain == "gcc" || toolchain == "clang" {
		autoEnableZig = false
	}
	
	if cfg != nil && cfg.Toolchain == "zig" || toolchain == "zig" {
		assembler.ZigEnabled = true
		linker.ZigEnabled = true
	} else if autoEnableZig && utils.CheckTool("zig") == nil {
		assembler.ZigEnabled = true
		linker.ZigEnabled = true
	}
}