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

package builder

import (
	"runtime"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func DetectHostTarget(cfg *config.Config) string {
	if cfg != nil && cfg.Target != "" {
		return cfg.Target
	}
	host := runtime.GOOS
	arch := runtime.GOARCH
	switch host {
	case "darwin":
		if arch == "arm64" {
			return "aarch64-apple-darwin"
		}
		return "x86_64-apple-darwin"
	case "windows":
		if arch == "arm64" {
			return "aarch64-windows-gnu"
		}
		return "x86_64-windows-gnu"
	default:
		if arch == "arm64" {
			return "aarch64-linux-gnu"
		}
		return "x86_64-linux-gnu"
	}
}

func DetectHostToolchain(cfg *config.Config) string {
	if cfg != nil && cfg.Toolchain != "" && cfg.Toolchain != "auto" {
		return cfg.Toolchain
	}
	if _, err := utils.LookExecutable("zig"); err == nil {
		return "zig"
	}
	if _, err := utils.LookExecutable("clang"); err == nil {
		return "clang"
	}
	if _, err := utils.LookExecutable("gcc"); err == nil {
		return "gcc"
	}
	return "auto"
}

func ApplyHostDetection(cfg *config.Config) {
	if cfg == nil {
		return
	}
	if cfg.Target == "" {
		cfg.Target = DetectHostTarget(cfg)
	}
	if cfg.Toolchain == "" || cfg.Toolchain == "auto" {
		cfg.Toolchain = DetectHostToolchain(cfg)
	}
}
