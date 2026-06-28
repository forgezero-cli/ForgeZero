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
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateArgsReject(t *testing.T) {
	if err := validateArgs([]string{"bad;arg"}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateArgsOK(t *testing.T) {
	if err := validateArgs([]string{"-O2"}); err != nil {
		t.Fatal(err)
	}
}

func TestAssembleInvalidPaths(t *testing.T) {
	err := Assemble(context.Background(), "../escape", "out.o", false, false, "auto")
	if err == nil {
		t.Fatal("expected path error")
	}
}

func TestAssembleWasmASMUnsupported(t *testing.T) {
	old := Target
	defer func() { Target = old }()
	Target = "wasm32-unknown-unknown"
	dir := t.TempDir()
	src := filepath.Join(dir, "t.asm")
	if err := os.WriteFile(src, []byte("section .text\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "t.o")
	err := Assemble(context.Background(), src, obj, false, false, "auto")
	if err == nil {
		t.Fatal("expected error for wasm target")
	}
}

func TestFormatFlagDefaultArch(t *testing.T) {
	old := Target
	defer func() { Target = old }()
	Target = "powerpc-unknown"
	if got := formatFlagForTarget(); got != "-felf64" {
		t.Fatal(got)
	}
}
