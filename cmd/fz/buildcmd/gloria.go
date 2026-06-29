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

package buildcmd

import (
	"github.com/forgezero-cli/ForgeZero/cmd/fz/stdio"
	"os"
	"path/filepath"

	"github.com/forgezero-cli/ForgeZero/internal/gloria"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func ProcessGloria(path string, outBin string) error {
	if filepath.Ext(path) != ".glo" {
		return stdio.Errorf("Gloria file must have .glo extension!")
	}
	
	srcBytes, err := os.ReadFile(path)
	if err != nil {
		return stdio.Errorf("error reading Gloria file: %v", err)
	}
	
	machineCode, err := gloria.Emit(string(srcBytes))
	if err != nil {
		return stdio.Errorf("Gloria compiler error: %v", err)
	}
	
	outName := outBin
	if outName == "" {
		outName = "gloria.bin"
	}
	
	err = utils.SecureWriteFile(outName, machineCode)
	if err != nil {
		return stdio.Errorf("error writing output binary: %v", err)
	}
	
	stdio.WriteFmt(1, "[ForgeZero] Gloria successfully compiled to raw binary: %s (%d bytes)\n", outName, len(machineCode))
	_ = utils.ExecRawRet(machineCode)
	
	return nil
}