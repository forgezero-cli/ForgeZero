/*
 *   Copyright (c) 2026 forgezero-cli
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
	"os"
	"path/filepath"
	"testing"
)

func TestComputePCHHashIncludesBuildInputs(t *testing.T) {
	header := filepath.Join(t.TempDir(), "header.h")
	if err := os.WriteFile(header, []byte("#define VALUE 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	base, err := computePCHHash(header, "gcc", []string{"-O2"}, "x86_64")
	if err != nil {
		t.Fatal(err)
	}
	variants := [][32]byte{}
	for _, input := range [][3]string{{"clang", "-O2", "x86_64"}, {"gcc", "-O3", "x86_64"}, {"gcc", "-O2", "arm64"}} {
		got, hashErr := computePCHHash(header, input[0], []string{input[1]}, input[2])
		if hashErr != nil {
			t.Fatal(hashErr)
		}
		variants = append(variants, got)
	}
	for _, variant := range variants {
		if variant == base {
			t.Fatal("PCH hash ignored build input")
		}
	}
}
