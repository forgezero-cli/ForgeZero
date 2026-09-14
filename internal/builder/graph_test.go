/*
 *   Copyright (c) 2026 forgezero-cli
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

func TestTopoSortWideGraph(t *testing.T) {
	const width = 128
	graph := make([][]int, width+1)
	for node := 1; node <= width; node++ {
		graph[node] = []int{0}
	}
	order, err := topoSort(graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != width+1 || order[0] != 0 {
		t.Fatalf("unexpected order length or root: %v", order[:min(len(order), 3)])
	}
	for index := 1; index <= width; index++ {
		if order[index] != index {
			t.Fatalf("order[%d] = %d, want %d", index, order[index], index)
		}
	}
}

func TestTopoSortSteadyStateDoesNotAllocate(t *testing.T) {
	graph := [][]int{{}, {0}, {0}, {1, 2}}
	allocs := testing.AllocsPerRun(1000, func() {
		order, err := topoSort(graph)
		if err != nil || len(order) != len(graph) {
			t.Fatalf("topoSort() = (%v, %v)", order, err)
		}
	})
	if allocs > 4 {
		t.Fatalf("topoSort allocations = %g, want at most 4", allocs)
	}
}

func TestRunDAGBuildWithoutPool(t *testing.T) {
	pairs := []pair{{src: "a"}, {src: "b"}, {src: "c"}}
	graph := [][]int{{}, {0}, {1}}
	var built []string
	err := runDAGBuild(nil, pairs, graph, func(item pair) error {
		built = append(built, item.src)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(built, []string{"a", "b", "c"}) {
		t.Fatalf("built order = %v, want [a b c]", built)
	}
}

func TestRunDAGBuildRejectsIncompleteGraph(t *testing.T) {
	pairs := []pair{{src: "a"}, {src: "b"}}
	if err := runDAGBuild(nil, pairs, [][]int{{}}, func(pair) error { return nil }); err != errInvalidDependency {
		t.Fatalf("incomplete graph error = %v", err)
	}
	if err := runDAGBuild(nil, pairs, [][]int{{1}, {0}}, func(pair) error { return nil }); err != errDependencyCycle {
		t.Fatalf("cyclic graph error = %v", err)
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
