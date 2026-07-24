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

package main

import (
	"encoding/json"
	"os"
	"os/exec"
)

type IRInstruction struct {
	Op   string `json:"op"`
	Dest string `json:"dest"`
	Arg1 string `json:"arg1"`
	Arg2 string `json:"arg2"`
}

func main() {
	cmd := exec.Command("python3", "parser.py", "test.py")
	output, err := cmd.Output()
	if err != nil {
		_, _ = os.Stderr.WriteString("Error parsing python: " + err.Error() + "\n")
		return
	}

	var irInstructions []IRInstruction
	if err := json.Unmarshal(output, &irInstructions); err != nil {
		_, _ = os.Stderr.WriteString("Error decoding IR: " + err.Error() + "\n")
		return
	}

	_, _ = os.Stdout.WriteString("[ForgeZero] Received IR from Python frontend:\n")
	for _, inst := range irInstructions {
		_, _ = os.Stdout.WriteString("Instruction: Op=" + inst.Op + ", Dest=" + inst.Dest + ", Arg1=" + inst.Arg1 + ", Arg2=" + inst.Arg2 + "\n")
	}
}
