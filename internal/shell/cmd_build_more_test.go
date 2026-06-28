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

package shell

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCmdBuildNoSourcePath(t *testing.T) {
	state := DefaultState()
	if err := cmdBuild(state); err == nil {
		t.Fatal("expected error")
	}
}

func TestCmdSetUnknownKey(t *testing.T) {
	state := DefaultState()
	if err := cmdSet(state, []string{"set", "unknown=x"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestCmdSetMissingValue(t *testing.T) {
	state := DefaultState()
	if err := cmdSet(state, []string{"set"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestCmdBuildDirNoSource(t *testing.T) {
	state := DefaultState()
	state.SourceType = "dir"
	state.SourcePath = filepath.Join(t.TempDir(), "empty")
	state.Out = filepath.Join(t.TempDir(), "out")
	if err := cmdBuild(state); err == nil {
		t.Fatal("expected build error")
	}
}

func TestCmdShowAllFields(t *testing.T) {
	state := DefaultState()
	state.Sanitize = true
	state.NoCache = true
	state.LdScript = "x.ld"
	state.TextAddr = "0x0"
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	cmdShow(state)
	w.Close()
	os.Stdout = old
}
