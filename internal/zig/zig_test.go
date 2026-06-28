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

package zig

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func TestCompilerForSource(t *testing.T) {
	if CompilerForSource(".c") != "cc" {
		t.Fatal("expected cc for .c")
	}
	if CompilerForSource(".cpp") != "c++" {
		t.Fatal("expected c++ for .cpp")
	}
	if CompilerForSource(".CC") != "c++" {
		t.Fatal("expected c++ for .CC")
	}
}

func TestCompileArgsIncludesTargetAndDebug(t *testing.T) {
	oldRoot := utils.GetExecutionRoot()
	defer utils.SetExecutionRoot(oldRoot)
	utils.SetExecutionRoot("/tmp/project")
	args := CompileArgs("main.c", "main.o", true, "arm-linux-gnueabihf", ".c", "-DTEST=1")
	if !strings.Contains(strings.Join(args, " "), "-target arm-linux-gnueabihf") {
		t.Fatal("missing target flag")
	}
	if !strings.Contains(strings.Join(args, " "), "-fdebug-prefix-map=/tmp/project=.") {
		t.Fatal("missing debug prefix map")
	}
	if !strings.Contains(strings.Join(args, " "), "-DTEST=1") {
		t.Fatal("missing extra flag")
	}
}

func TestLinkArgsBuildIdSuppression(t *testing.T) {
	args := LinkArgs([]string{"a.o", "b.o"}, "out", "x86_64-linux-gnu", false, false, []string{"m"}, false, "script.ld", "0x1000")
	if !strings.Contains(strings.Join(args, " "), "-Wl,--build-id=none") {
		t.Fatal("missing build-id suppression")
	}
	if !strings.Contains(strings.Join(args, " "), "-Wl,-T,script.ld") {
		t.Fatal("missing linker script flag")
	}
}

func TestLinkArgsWasmOmitsBuildId(t *testing.T) {
	args := LinkArgs([]string{"a.o"}, "out", "wasm32-unknown-unknown", false, false, nil, false, "", "")
	if strings.Contains(strings.Join(args, " "), "--build-id=none") {
		t.Fatal("build-id should be omitted for wasm")
	}
}

func TestCompileReturnsErrorWhenZigUnavailable(t *testing.T) {
	oldFunc := utils.CheckToolFunc
	defer func() { utils.CheckToolFunc = oldFunc }()
	utils.CheckToolFunc = func(name string) error {
		return fmt.Errorf("no")
	}
	ZigRequested = true
	defer func() { ZigRequested = false }()
	if err := Compile(context.Background(), "main.c", "main.o", false, false, "x86_64-linux-gnu", ""); err == nil {
		t.Fatal("expected error when zig unavailable")
	}
}

func TestLinkReturnsErrorWhenZigUnavailable(t *testing.T) {
	oldFunc := utils.CheckToolFunc
	defer func() { utils.CheckToolFunc = oldFunc }()
	utils.CheckToolFunc = func(name string) error {
		return fmt.Errorf("no")
	}
	ZigRequested = true
	defer func() { ZigRequested = false }()
	if err := Link(context.Background(), []string{"a.o"}, "out", false, "x86_64-linux-gnu", false, false, nil, false, "", "", ""); err == nil {
		t.Fatal("expected error when zig unavailable")
	}
}

func TestCompileCommandInvoked(t *testing.T) {
	oldRun := RunCommand
	oldFunc := utils.CheckToolFunc
	defer func() { RunCommand = oldRun; utils.CheckToolFunc = oldFunc }()
	utils.CheckToolFunc = func(name string) error { return nil }
	ZigEnabled = true
	defer func() { ZigEnabled = false }()
	invoked := false
	RunCommand = func(ctx context.Context, verbose bool, args ...string) (string, error) {
		invoked = true
		if args[0] != "cc" {
			t.Fatalf("expected zig cc path, got %v", args)
		}
		return "", nil
	}
	if err := Compile(context.Background(), "main.c", "main.o", false, false, "x86_64-linux-gnu", ""); err != nil {
		t.Fatal(err)
	}
	if !invoked {
		t.Fatal("expected zig compile to run")
	}
}
