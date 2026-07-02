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

package utils

import (
	"runtime"
	"testing"
)

func TestShellCommand(t *testing.T) {
	name, args := ShellCommand("echo hi")
	if name == "" {
		t.Fatal("expected non-empty shell name")
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[1] != "echo hi" {
		t.Errorf("expected command preserved, got %q", args[1])
	}
	if runtime.GOOS == "windows" {
		if args[0] != "/C" {
			t.Errorf("expected /C on windows, got %q", args[0])
		}
	} else {
		if args[0] != "-c" {
			t.Errorf("expected -c on unix, got %q", args[0])
		}
	}
}
