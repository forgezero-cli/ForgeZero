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

package assembler

import (
	"strings"
)

func IsBinFormat() bool {
	return OutputFormat == "bin"
}

func IsBareMetalTarget() bool {
	t := Target
	if strings.Contains(t, "baremetal") || strings.Contains(t, "bare-metal") {
		return true
	}
	if strings.Contains(t, "cortex-") {
		return true
	}
	if strings.Contains(t, "none-") {
		return true
	}
	if strings.Contains(t, "unknown-elf") {
		return true
	}
	if strings.Contains(t, "-elf") && !strings.Contains(t, "linux") {
		return true
	}
	return false
}

func SkipLinker() bool {
	return IsBinFormat()
}

func formatFlagForTarget() string {
	if IsBinFormat() {
		return "-fbin"
	}
	switch {
	case isWasmTarget():
		return ""
	case strings.Contains(Target, "x86_64"):
		return "-felf64"
	case strings.Contains(Target, "i386") || strings.Contains(Target, "i686"):
		return "-felf32"
	case strings.Contains(Target, "arm"):
		return "-march=armv7-a"
	default:
		return "-felf64"
	}
}
