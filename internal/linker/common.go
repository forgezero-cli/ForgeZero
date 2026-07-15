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

package linker

import (
	"errors"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/assembler"
)

var validFormats = []string{"elf32", "elf64", "bin"}

func ApplyGccLdFlags(args []string, ldScript, textAddr string) []string {
	if ldScript != "" {
		args = append(args, "-Wl,-T,"+ldScript)
	}
	if textAddr != "" {
		args = append(args, "-Wl,-Ttext="+textAddr)
	}
	if !strings.Contains(assembler.Target, "wasm") && !strings.Contains(assembler.Target, "wasm32") {
		args = append(args, "-Wl,--build-id=none")
	}
	return args
}

func ApplyLdFlags(args []string, ldScript, textAddr string) []string {
	if ldScript != "" {
		args = append(args, "-T", ldScript)
	}
	if textAddr != "" {
		args = append(args, "-Ttext", textAddr)
	}
	if !strings.Contains(assembler.Target, "wasm") && !strings.Contains(assembler.Target, "wasm32") {
		args = append(args, "--build-id=none")
	}
	return args
}

func SetOutputFormat(format string) error {
	for _, f := range validFormats {
		if f == format {
			assembler.OutputFormat = format
			return nil
		}
	}
	return errors.New("invalid output format: " + format + " (supported: elf32, elf64, bin)")
}
