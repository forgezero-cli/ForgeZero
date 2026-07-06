/*
 * Copyright (c) 2026 ForgeZero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package builder

import (
	"context"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/assembler"
	"github.com/forgezero-cli/ForgeZero/internal/drivers/scheduler"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)



func TestPCHSetPCHIncludeArgsZeroAlloc(t *testing.T) {
	assembler.ResetPCH()
	before := make([]string, 0, 0)
	assembler.SetPCHIncludeArgs(before)
	include := []string{"-include", "hdr"}
	assembler.SetPCHIncludeArgs(include)

	f := func() {
		if len(assembler.PCHIncludeArgs) != 2 {
			t.Fatalf("unexpected PCHIncludeArgs length: %d", len(assembler.PCHIncludeArgs))
		}
		if assembler.PCHIncludeArgs[0] != "-include" {
			t.Fatalf("unexpected PCHIncludeArgs[0]: %q", assembler.PCHIncludeArgs[0])
		}
		if assembler.PCHIncludeArgs[1] != "hdr" {
			t.Fatalf("unexpected PCHIncludeArgs[1]: %q", assembler.PCHIncludeArgs[1])
		}
	}

	allocs := testing.AllocsPerRun(1000, f)
	if allocs != 0 {
		t.Fatalf("expected 0 allocs/op, got %f", allocs)
	}
}

func TestPreBuildHooksExecuteBeforeCompileAndRefreshUpdatesCache(t *testing.T) {
	srcDir := t.TempDir()
	srcFile := srcDir + "/main.c"
	if err := utils.EnsureInsideRoot(srcDir, srcFile); err != nil {
		t.Fatal(err)
	}
	if err := utils.SecureMkdirAll(srcDir); err != nil {
		t.Fatal(err)
	}
	if err := utils.SecureWriteFile(srcFile, []byte("int main(){return 0;}")); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	assembler.ResetPCH()

	order := make([]int, 3)
	pos := 0


	compilerTask := func(ctx context.Context) error {
		order[pos] = 2
		pos++
		return nil
	}
	refreshTask := func(ctx context.Context) error {
		order[pos] = 1
		pos++
		return refreshSourceHashes([]string{srcDir})
	}
	genTask := func(ctx context.Context) error {
		order[pos] = 0
		pos++
		return nil
	}

	dag := scheduler.NewDAGScheduler(2, 8)
	genDep, err := dag.Submit(genTask, nil)
	if err != nil {
		t.Fatal(err)
	}
	refreshDep, err := dag.Submit(refreshTask, []int{genDep})
	if err != nil {
		t.Fatal(err)
	}
	_, err = dag.Submit(func(taskCtx context.Context) error {
		_ = refreshDep
		return compilerTask(taskCtx)
	}, []int{refreshDep})
	if err != nil {
		t.Fatal(err)
	}

	if err := dag.Run(ctx); err != nil {
		t.Fatal(err)
	}

	if order[0] != 0 || order[1] != 1 || order[2] != 2 {
		t.Fatalf("unexpected order %v", order)
	}
}

