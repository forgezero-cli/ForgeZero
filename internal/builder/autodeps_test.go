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

func TestDepBuilderRunCustomStepsExecutesCommand(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "custom.out")
	cfg := &config.Config{
		DepBuild: config.DepBuildConfig{
			Steps: []config.BuildStep{{
				Command: "printf '%s' \"${FZ_DEP_NAME}\" > custom.out",
			}},
		},
	}

	db := NewDepBuilder(context.Background(), dir, "mylib", cfg, nil, false)
	if err := db.runCustomSteps(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mylib" {
		t.Fatalf("unexpected output: %q", string(data))
	}
}

func TestDepBuilderRunCustomStepsConditionSkipsStep(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "skip.out")
	cfg := &config.Config{
		DepBuild: config.DepBuildConfig{
			Steps: []config.BuildStep{{
				If:      "${FZ_DEP_NAME} == other",
				Command: "printf 'skipped' > skip.out",
			}},
		},
	}

	db := NewDepBuilder(context.Background(), dir, "mylib", cfg, nil, false)
	if err := db.runCustomSteps(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outFile); err == nil {
		t.Fatal("expected step to be skipped")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestDepBuilderRunCustomStepsInternalRunPostBuild(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "post.out")
	cfg := &config.Config{
		DepBuild: config.DepBuildConfig{
			PostBuild: []string{"printf 'done' > post.out"},
			Steps: []config.BuildStep{{
				Run: "post_build",
			}},
		},
	}

	db := NewDepBuilder(context.Background(), dir, "mylib", cfg, nil, false)
	if err := db.runCustomSteps(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "done" {
		t.Fatalf("unexpected output: %q", string(data))
	}
}
