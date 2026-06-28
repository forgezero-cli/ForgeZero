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
	"flag"
	"runtime"
	"strings"
)

func DefaultTargetFromGOARCH() string {
	switch runtime.GOARCH {
	case "amd64":
		if runtime.GOOS == "windows" {
			return "x86_64-windows-gnu"
		}
		return "x86_64-linux-gnu"
	case "386":
		if runtime.GOOS == "windows" {
			return "i686-windows-gnu"
		}
		return "i386-linux-gnu"
	case "arm64":
		return "aarch64-linux-gnu"
	case "arm":
		return "arm-linux-gnueabihf"
	case "riscv64":
		return "riscv64-unknown-elf"
	default:
		return "x86_64-linux-gnu"
	}
}

func ApplyDefaultTarget() {
	if muslFlag := flag.Lookup("musl"); muslFlag != nil && muslFlag.Value.String() != "" {
		return
	}
	if Target == "" || Target == "x86_64-linux-gnu" {
		Target = DefaultTargetFromGOARCH()
	}
}

func TargetFromGOARCHExplicit() string {
	return DefaultTargetFromGOARCH()
}

func NormalizeTargetTriple(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return DefaultTargetFromGOARCH()
	}
	return t
}
