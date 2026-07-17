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

func TestDepBuilderRunCustomStepsParallelGroup(t *testing.T) {
	dir := t.TempDir()
	countFile := filepath.Join(dir, "count.out")
	cfg := &config.Config{
		DepBuild: config.DepBuildConfig{
			Steps: []config.BuildStep{
				{Group: "build", Parallel: true, Command: "printf '1' >> count.out", Outputs: []string{"count.out"}, Inputs: []string{"input.txt"}},
				{Group: "build", Parallel: true, Command: "printf '2' >> count.out", Outputs: []string{"count.out"}, Inputs: []string{"input.txt"}},
				{Stage: 1, Group: "final", Command: "printf '3' >> count.out", Outputs: []string{"count.out"}, Inputs: []string{"input.txt"}},
			},
		},
	}

	if err := os.WriteFile(filepath.Join(dir, "input.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	db := NewDepBuilder(context.Background(), dir, "mylib", cfg, nil, false)
	if err := db.runCustomSteps(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 3 {
		t.Fatalf("expected 3 bytes in count.out, got %d", len(data))
	}
}

func TestDepBuilderRunCustomStepsGroupOrdering(t *testing.T) {
	dir := t.TempDir()
	orderFile := filepath.Join(dir, "order.out")
	cfg := &config.Config{
		DepBuild: config.DepBuildConfig{
			Steps: []config.BuildStep{
				{Group: "fetch", Stage: 0, Command: "printf 'fetch\n' >> order.out", Outputs: []string{"order.out"}, Inputs: []string{"fetch.txt"}},
				{Group: "build", Stage: 1, Command: "printf 'build\n' >> order.out", Outputs: []string{"order.out"}, Inputs: []string{"build.txt"}},
				{Group: "publish", Stage: 2, Command: "printf 'publish\n' >> order.out", Outputs: []string{"order.out"}, Inputs: []string{"publish.txt"}},
			},
		},
	}

	for _, name := range []string{"fetch.txt", "build.txt", "publish.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	db := NewDepBuilder(context.Background(), dir, "mylib", cfg, nil, false)
	if err := db.runCustomSteps(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(orderFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fetch\nbuild\npublish\n" {
		t.Fatalf("unexpected order: %q", string(data))
	}
}

func TestDepBuilderRunCustomStepsMixedSequentialAndParallelGroups(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		DepBuild: config.DepBuildConfig{
			Steps: []config.BuildStep{
				{Group: "parallel", Stage: 0, Parallel: true, Command: "printf 'x' > a1.done", Inputs: []string{"seed.txt"}, Outputs: []string{"a1.done"}},
				{Group: "parallel", Stage: 0, Parallel: true, Command: "printf 'x' > a2.done", Inputs: []string{"seed.txt"}, Outputs: []string{"a2.done"}},
				{Group: "seq", Stage: 0, Command: "sh -c 'while [ ! -f a1.done ] || [ ! -f a2.done ]; do sleep 0.01; done; printf \"S\n\" > seq.out'", Inputs: []string{"seed.txt"}, Outputs: []string{"seq.out"}},
				{Group: "final", Stage: 1, Command: "sh -c 'while [ ! -f seq.out ]; do sleep 0.01; done; printf \"F\n\" > final.out'", Inputs: []string{"seed.txt"}, Outputs: []string{"final.out"}},
			},
		},
	}

	db := NewDepBuilder(context.Background(), dir, "mylib", cfg, nil, false)
	if err := db.runCustomSteps(); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"a1.done", "a2.done", "seq.out", "final.out"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}
}

func TestDepBuilderRunCustomStepsStepSetTemplate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
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
			Steps: []config.BuildStep{{
				StepSet: "write_file",
				With:    map[string]string{"NAME": "alpha", "CONTENT": "alpha"},
			}},
		},
	}

	db := NewDepBuilder(context.Background(), dir, "mylib", cfg, nil, false)
	if err := db.runCustomSteps(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "alpha.out")); err != nil {
		t.Fatal(err)
	}
}

func TestDepBuilderRunCustomStepsTryCatchFinally(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		DepBuild: config.DepBuildConfig{
			Steps: []config.BuildStep{
				{Group: "flow", Stage: 0, Parallel: true, Try: true, Command: "printf 'try' > try.out", Inputs: []string{"seed.txt"}, Outputs: []string{"try.out"}},
				{Group: "flow", Stage: 0, Parallel: true, Command: "false", Inputs: []string{"seed.txt"}, Outputs: []string{"fail.out"}},
				{Group: "flow", Stage: 0, Parallel: true, Catch: true, Command: "printf 'catch' > catch.out", Inputs: []string{"seed.txt"}, Outputs: []string{"catch.out"}},
				{Group: "flow", Stage: 0, Parallel: true, Finally: true, Command: "printf 'finally' > finally.out", Inputs: []string{"seed.txt"}, Outputs: []string{"finally.out"}},
			},
		},
	}

	db := NewDepBuilder(context.Background(), dir, "mylib", cfg, nil, false)
	if err := db.runCustomSteps(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"try.out", "catch.out", "finally.out"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}
}

func TestDepBuilderParallelGroupCapacity(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		DepBuild: config.DepBuildConfig{
			Steps: []config.BuildStep{
				{Group: "parallel", Parallel: true, Command: "printf 'A' > a.out", Inputs: []string{"in.txt"}, Outputs: []string{"a.out"}},
				{Group: "parallel", Parallel: true, Command: "printf 'B' > b.out", Inputs: []string{"in.txt"}, Outputs: []string{"b.out"}},
			},
		},
	}
	if err := os.WriteFile(filepath.Join(dir, "in.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	db := NewDepBuilder(context.Background(), dir, "mylib", cfg, nil, false)
	if err := db.runCustomSteps(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.out", "b.out"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}
}
