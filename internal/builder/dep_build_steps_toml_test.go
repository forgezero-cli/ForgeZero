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

func TestDepBuilderRunCustomStepsTOMLExample(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fz.toml")
	content := `
[dep_build]
outputs = ["summary.out"]

[[dep_build.step_sets]]
name = "write_file"
command = "printf '${CONTENT}' > ${NAME}.out"
inputs = ["seed.txt"]
outputs = ["${NAME}.out"]

[[dep_build.steps]]
step_set = "write_file"
group = "setup"
stage = 0
parallel = true
with = { NAME = "alpha", CONTENT = "alpha" }

[[dep_build.steps]]
step_set = "write_file"
group = "setup"
stage = 0
parallel = true
with = { NAME = "beta", CONTENT = "beta" }

[[dep_build.steps]]
group = "flow"
stage = 1
try = true
command = "printf 'try' > try.out"
inputs = ["seed.txt"]
outputs = ["try.out"]

[[dep_build.steps]]
group = "flow"
stage = 1
command = "false"
inputs = ["seed.txt"]
outputs = ["fail.out"]

[[dep_build.steps]]
group = "flow"
stage = 1
catch = true
command = "printf 'catch' > catch.out"
inputs = ["seed.txt"]
outputs = ["catch.out"]

[[dep_build.steps]]
group = "flow"
stage = 1
finally = true
command = "printf 'finally' > finally.out"
inputs = ["seed.txt"]
outputs = ["finally.out"]

[[dep_build.steps]]
group = "checks"
stage = 2
command = "printf 'done' > summary.out"
inputs = ["try.out"]
outputs = ["summary.out"]
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
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
