/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package builder

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestTopoSortEmptyGraph(t *testing.T) {
	order, err := topoSort(nil)
	if err != nil {
		t.Fatal(err)
	}
	if order != nil {
		t.Fatalf("expected nil order for empty graph, got %v", order)
	}
}

func TestTopoSortInvalidDependency(t *testing.T) {
	_, err := topoSort([][]int{{1}})
	if err == nil || err != errInvalidDependency {
		t.Fatalf("expected invalid dependency error, got %v", err)
	}
}

func TestTopoSortCycle(t *testing.T) {
	_, err := topoSort([][]int{{1}, {0}})
	if err == nil || err != errDependencyCycle {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestTopoSortLinearGraph(t *testing.T) {
	order, err := topoSort([][]int{{}, {0}, {1}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []int{0, 1, 2}) {
		t.Fatalf("expected [0 1 2], got %v", order)
	}
}

func TestTopoSortBranchedGraph(t *testing.T) {
	order, err := topoSort([][]int{{}, {0}, {0}})
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 || order[0] != 0 {
		t.Fatalf("expected root first, got %v", order)
	}
	if !(order[1] == 1 && order[2] == 2 || order[1] == 2 && order[2] == 1) {
		t.Fatalf("unexpected order for branched graph: %v", order)
	}
}

func TestBuildDependencyGraphWithDepFile(t *testing.T) {
	dir := t.TempDir()
	srcA := filepath.Join(dir, "a.c")
	srcB := filepath.Join(dir, "b.c")
	if err := os.WriteFile(srcA, []byte("int a() { return 0; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcB, []byte("int b() { return 1; }"), 0o644); err != nil {
		t.Fatal(err)
	}

depFile := filepath.Join(dir, "a.d")
	if err := os.WriteFile(depFile, []byte("a.o: a.c b.c\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	pairs := []pair{{src: srcA, obj: filepath.Join(dir, "a.o")}, {src: srcB, obj: filepath.Join(dir, "b.o")}}
	graph, err := buildDependencyGraph(pairs, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph) != 2 {
		t.Fatalf("expected graph length 2, got %d", len(graph))
	}
	if len(graph[0]) != 1 || graph[0][0] != 1 {
		t.Fatalf("expected a.c to depend on b.c, got %v", graph[0])
	}
	if len(graph[1]) != 0 {
		t.Fatalf("expected b.c to have no deps, got %v", graph[1])
	}
}
