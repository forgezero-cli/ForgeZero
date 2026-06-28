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
	"os/exec"
	"strings"
	"testing"
)

func TestCcCxxGasForAllTargets(t *testing.T) {
	old := Target
	defer func() { Target = old }()
	cases := []struct {
		target  string
		wantCC  string
		wantCXX string
		wantGas string
		wantFmt string
	}{
		{"x86_64-linux-gnu", "gcc", "g++", "as", "-felf64"},
		{"i386-linux-gnu", "gcc", "g++", "as", "-felf32"},
		{"arm-linux-gnueabihf", "arm-linux-gnueabihf-gcc", "arm-linux-gnueabihf-g++", "arm-linux-gnueabihf-as", "-march=armv7-a"},
		{"riscv64-unknown-elf", "riscv64-unknown-elf-gcc", "riscv64-unknown-elf-g++", "riscv64-unknown-elf-as", "-felf64"},
		{"wasm32-unknown-unknown", "clang", "clang++", "clang", ""},
	}
	for _, tc := range cases {
		Target = tc.target
		if tc.target == "wasm32-unknown-unknown" {
			if _, err := exec.LookPath("emcc"); err == nil {
				tc.wantCC = "emcc"
			}
			if _, err := exec.LookPath("em++"); err == nil {
				tc.wantCXX = "em++"
			}
		}

		if tc.target == "riscv64-unknown-elf" {
			gotCC := ccForTarget()
			if strings.Contains(gotCC, "zig") {
				tc.wantCC = gotCC
			}

			gotCXX := cxxForTarget()
			if strings.Contains(gotCXX, "zig") {
				tc.wantCXX = gotCXX
			} else {
				tc.wantCXX = "riscv64-unknown-elf-g++"
			}
		}
		if got := ccForTarget(); got != tc.wantCC {
			t.Fatalf("%s cc: %s", tc.target, got)
		}
		if got := cxxForTarget(); got != tc.wantCXX {
			t.Fatalf("%s cxx: %s", tc.target, got)
		}
		if got := gasCmdForTarget(); got != tc.wantGas {
			t.Fatalf("%s gas: %s", tc.target, got)
		}
		if got := formatFlagForTarget(); got != tc.wantFmt {
			t.Fatalf("%s fmt: %s", tc.target, got)
		}
	}
}
