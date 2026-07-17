/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version of the License.
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

func TestDepBuilderRunCustomStepsIntegrationExample(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		DepBuild: config.DepBuildConfig{
			StepSets: []config.StepSet{{
				Name: "write_file",
				BuildStep: config.BuildStep{
					Command: "printf '${CONTENT}' > ${NAME}.out",
					Inputs:  []string{"seed.txt"},
					Outputs: []string{"${NAME}.out"},
				},
			}},
			Steps: []config.BuildStep{
				{Group: "setup", Stage: 0, Parallel: true, StepSet: "write_file", With: map[string]string{"NAME": "alpha", "CONTENT": "alpha"}},
				{Group: "setup", Stage: 0, Parallel: true, StepSet: "write_file", With: map[string]string{"NAME": "beta", "CONTENT": "beta"}},
				{Group: "flow", Stage: 1, Try: true, Command: "printf 'try' > try.out", Inputs: []string{"seed.txt"}, Outputs: []string{"try.out"}},
				{Group: "flow", Stage: 1, Command: "false", Inputs: []string{"seed.txt"}, Outputs: []string{"fail.out"}},
				{Group: "flow", Stage: 1, Catch: true, Command: "printf 'catch' > catch.out", Inputs: []string{"seed.txt"}, Outputs: []string{"catch.out"}},
				{Group: "flow", Stage: 1, Finally: true, Command: "printf 'finally' > finally.out", Inputs: []string{"seed.txt"}, Outputs: []string{"finally.out"}},
				{Group: "checks", Stage: 2, Command: "printf 'done' > summary.out", Inputs: []string{"try.out"}, Outputs: []string{"summary.out"}},
			},
		},
	}

	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	db := NewDepBuilder(context.Background(), dir, "mylib", cfg, nil, false)
	if err := db.runCustomSteps(); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"alpha.out", "beta.out", "try.out", "catch.out", "finally.out", "summary.out"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}
}
