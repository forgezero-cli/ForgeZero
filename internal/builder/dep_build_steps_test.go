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

package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/config"
)

func TestDepBuilderRunCustomStepsElseBranch(t *testing.T) {
	dir := t.TempDir()
	outA := filepath.Join(dir, "a.out")
	outB := filepath.Join(dir, "b.out")
	cfg := &config.Config{
		DepBuild: config.DepBuildConfig{
			Steps: []config.BuildStep{
				{If: "${FZ_DEP_NAME} == mylib", Command: "printf 'A' > a.out"},
				{Else: true, Command: "printf 'B' > b.out"},
			},
		},
	}

	db := NewDepBuilder(context.Background(), dir, "other", cfg, nil, false)
	if err := db.runCustomSteps(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outA); err == nil {
		t.Fatal("expected a.out not to exist")
	}
	if _, err := os.Stat(outB); err != nil {
		t.Fatal(err)
	}
}

func TestDepBuilderRunCustomStepsCacheRestore(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "cached.out")
	cfg := &config.Config{
		DepBuild: config.DepBuildConfig{
			Steps: []config.BuildStep{{
				Command: "printf 'cached' > cached.out",
				Inputs:  []string{"input.txt"},
				Outputs: []string{"cached.out"},
			}},
		},
	}

	if err := os.WriteFile(filepath.Join(dir, "input.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	db := NewDepBuilder(context.Background(), dir, "mylib", cfg, nil, false)
	if err := db.runCustomSteps(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outFile); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(outFile); err != nil {
		t.Fatal(err)
	}

	db2 := NewDepBuilder(context.Background(), dir, "mylib", cfg, nil, false)
	if err := db2.runCustomSteps(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outFile); err != nil {
		t.Fatal(err)
	}
}
