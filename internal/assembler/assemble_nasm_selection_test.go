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
	"strings"
	"testing"
)

type runCmdCapture struct {
	nasmCalled bool
}

func TestAsm_DefaultUsesInternalAssembler_NoNasm(t *testing.T) {
	oldForceInternal := ForceInternalAsm
	oldUseNasm := UseNasm
	defer func() {
		ForceInternalAsm = oldForceInternal
		UseNasm = oldUseNasm
	}()

	ForceInternalAsm = true
	UseNasm = false

	cap := &runCmdCapture{}
	oldRun := runCommand
	defer func() { runCommand = oldRun }()
	SetRunCommand(func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if strings.Contains(name, "nasm") {
			cap.nasmCalled = true
		}
		return "", nil
	})

	dir := t.TempDir()
	src := filepath.Join(dir, "t.asm")
	if err := os.WriteFile(src, []byte("section .text\nstart:\n  nop\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "t.o")

	_ = Assemble(context.Background(), src, obj, false, false, "raw")

	if cap.nasmCalled {
		t.Fatal("nasm was invoked, expected internal assembler by default")
	}
}

func TestAsm_UseNasmRoutesToNasm(t *testing.T) {
	oldForceInternal := ForceInternalAsm
	oldUseNasm := UseNasm
	defer func() {
		ForceInternalAsm = oldForceInternal
		UseNasm = oldUseNasm
	}()

	ForceInternalAsm = false
	UseNasm = true

	cap := &runCmdCapture{}
	oldRun := runCommand
	defer func() { runCommand = oldRun }()
	SetRunCommand(func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if strings.Contains(name, "nasm") {
			cap.nasmCalled = true
		}
		return "", nil
	})

	dir := t.TempDir()
	src := filepath.Join(dir, "t.asm")
	if err := os.WriteFile(src, []byte("section .text\nstart:\n  nop\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "t.o")

	_ = Assemble(context.Background(), src, obj, false, false, "raw")

	if !cap.nasmCalled {
		t.Fatal("nasm was not invoked, expected NASM routing when UseNasm=true")
	}
}
