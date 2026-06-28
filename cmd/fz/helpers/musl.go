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
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/assembler"
	"github.com/forgezero-cli/ForgeZero/internal/musl"
)

func FindLibGccPath() string {
	execRoot := os.Getenv("FZ_EXEC_ROOT")

	if execRoot != "" {
		localPath := filepath.Join(execRoot, "riscv64-linux-musl-cross", "lib", "gcc", "riscv64-linux-musl")

		if files, err := os.ReadDir(localPath); err == nil && len(files) > 0 {
			for _, f := range files {
				if f.IsDir() {
					candidate := filepath.Join(localPath, f.Name(), "libgcc.a")
					if _, err := os.Stat(candidate); err == nil {
						return candidate
					}
				}
			}
		}
	}

	if gccBin, err := exec.LookPath("riscv64-linux-musl-gcc"); err == nil {
		cmd := exec.Command(gccBin, "-print-libgcc-file-name")
		if out, err := cmd.Output(); err == nil {
			candidate := strings.TrimSpace(string(out))
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}

	return ""
}

func SetupMusl(opt string) MuslContext {
	ctx := MuslContext{Use: false}
	if opt == "" {
		return ctx
	}
	ctx.Use = true
	if opt == "true" {
		ctx.Arch = "x86_64"
	} else {
		ctx.Arch = opt
	}
	if ctx.Arch == "riscv64" {
		ctx.Target = "riscv64-linux-musl"
	} else {
		ctx.Target = "x86_64-linux-musl"
	}
	assembler.Target = ctx.Target
	return ctx
}

func BuildWithMusl(objFiles []string, binary string, arch string, verbose bool, jsonOutput bool) error {
	muslDir := musl.NewToolchain(arch)
	tmpPath, err := muslDir.Prepare()
	if err != nil {
		return Errorf("musl extract failed: %w", err)
	}
	defer muslDir.Close()

	argsCount := len(objFiles) + 10
	args := make([]string, argsCount)
	musl.GetLinkerArgsZeroAlloc(args, tmpPath, objFiles, binary)

	if strings.Contains(assembler.Target, "riscv") {
		if lgcc := FindLibGccPath(); lgcc != "" {
			if verbose {
				WriteFmt(1, "Using detected libgcc: %s\n", lgcc)
			}
			args = append(args, lgcc)
		}
	}

	if verbose && !jsonOutput {
		WriteFmt(1, "Linking %d object files with static Musl -> %s\n", len(objFiles), binary)
	}

	cmd := exec.Command("ld.lld", args...)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return Errorf("musl link failed: %v\n%s", err, errBuf.String())
	}

	if !jsonOutput {
		WriteFmt(1, "build(static musl [%s]): %s\n", arch, binary)
	}
	return nil
}