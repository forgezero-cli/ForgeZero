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
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

var (
	errInvalidDependency = errors.New("invalid dependency index")
	errDependencyCycle   = errors.New("dependency cycle detected")
	topoScratchPool      = sync.Pool{New: func() any { return &topoScratch{} }}
	topoOrderPool        = sync.Pool{New: func() any { s := make([]int, 0, 128); return &s }}
)

type topoScratch struct {
	inDegree  []int
	queue     []int
	adjIndex  []int
	adjCursor []int
	adjData   []int
}

func ensureIntLen(slice []int, length int) []int {
	if cap(slice) < length {
		return make([]int, length)
	}
	return slice[:length]
}

func ensureIntCap(slice []int, capacity int) []int {
	if cap(slice) < capacity {
		return make([]int, 0, capacity)
	}
	return slice[:0]
}

func depFilePath(src string) string {
	return strings.TrimSuffix(src, filepath.Ext(src)) + ".d"
}

func compileDependencies(src string, knownSources map[string]int, rootDir string) ([]int, error) {
	if src == "" {
		return nil, nil
	}
	deps, err := utils.ParseDepFilePath(depFilePath(src))
	if err != nil {
		deps, err = utils.ScanDependenciesRoot(src, rootDir)
		if err != nil {
			return nil, err
		}
	}
	if len(deps) == 0 {
		return nil, nil
	}
	unique := make([]int, 0, len(deps))
	seen := make(map[int]struct{}, len(deps))
	for _, dep := range deps {
		key := filepath.Clean(dep)
		if idx, ok := knownSources[key]; ok && idx >= 0 {
			if idx == knownSources[filepath.Clean(src)] {
				continue
			}
			if _, ok := seen[idx]; !ok {
				seen[idx] = struct{}{}
				unique = append(unique, idx)
			}
		}
	}
	sort.Ints(unique)
	return unique, nil
}

func buildDependencyGraph(pairs []pair, rootDir string) ([][]int, error) {
	knownSources := make(map[string]int, len(pairs)*2)
	for i, p := range pairs {
		src := filepath.Clean(p.src)
		knownSources[src] = i
		if abs, err := filepath.Abs(p.src); err == nil {
			knownSources[filepath.Clean(abs)] = i
		}
	}
	graph := make([][]int, len(pairs))
	for i, p := range pairs {
		if deps, err := compileDependencies(p.src, knownSources, rootDir); err != nil {
			return nil, err
		} else {
			graph[i] = deps
		}
	}
	return graph, nil
}

func topoSort(graph [][]int) ([]int, error) {
	n := len(graph)
	if n == 0 {
		return nil, nil
	}
	scratch := topoScratchPool.Get().(*topoScratch)
	orderPtr := topoOrderPool.Get().(*[]int)
	order := (*orderPtr)[:0]
	order = ensureIntCap(order, n)
	scratch.inDegree = ensureIntLen(scratch.inDegree, n)
	scratch.adjIndex = ensureIntLen(scratch.adjIndex, n+1)
	for i := 0; i <= n; i++ {
		scratch.adjIndex[i] = 0
	}
	totalEdges := 0
	for _, deps := range graph {
		totalEdges += len(deps)
	}
	scratch.adjData = ensureIntLen(scratch.adjData, totalEdges)
	scratch.adjCursor = ensureIntLen(scratch.adjCursor, n)
	scratch.queue = ensureIntCap(scratch.queue, n)
	for i, deps := range graph {
		scratch.inDegree[i] = len(deps)
		for _, dep := range deps {
			if dep < 0 || dep >= n {
				topoScratchPool.Put(scratch)
				*orderPtr = order[:0]
				topoOrderPool.Put(orderPtr)
				return nil, errInvalidDependency
			}
			scratch.adjIndex[dep+1]++
		}
	}
	for i := 1; i <= n; i++ {
		scratch.adjIndex[i] += scratch.adjIndex[i-1]
	}
	copy(scratch.adjCursor, scratch.adjIndex[:n])
	for i, deps := range graph {
		for _, dep := range deps {
			scratch.adjData[scratch.adjCursor[dep]] = i
			scratch.adjCursor[dep]++
		}
	}
	for i, deg := range scratch.inDegree {
		if deg == 0 {
			scratch.queue = append(scratch.queue, i)
		}
	}
	if len(scratch.queue) == 0 {
		topoScratchPool.Put(scratch)
		*orderPtr = order[:0]
		topoOrderPool.Put(orderPtr)
		return nil, errDependencyCycle
	}
	for len(scratch.queue) > 0 {
		node := scratch.queue[0]
		scratch.queue = scratch.queue[1:]
		order = append(order, node)
		start := scratch.adjIndex[node]
		end := scratch.adjIndex[node+1]
		for j := start; j < end; j++ {
			next := scratch.adjData[j]
			scratch.inDegree[next]--
			if scratch.inDegree[next] == 0 {
				scratch.queue = append(scratch.queue, next)
			}
		}
	}
	if len(order) != n {
		topoScratchPool.Put(scratch)
		*orderPtr = order[:0]
		topoOrderPool.Put(orderPtr)
		return nil, errDependencyCycle
	}
	scratch.inDegree = scratch.inDegree[:0]
	scratch.adjIndex = scratch.adjIndex[:0]
	scratch.adjCursor = scratch.adjCursor[:0]
	scratch.adjData = scratch.adjData[:0]
	scratch.queue = scratch.queue[:0]
	topoScratchPool.Put(scratch)
	return order, nil
}

func remapDependencies(order []int, graph [][]int) [][]int {
	n := len(order)
	pos := make([]int, n)
	for i, src := range order {
		pos[src] = i
	}
	remapped := make([][]int, n)
	for i, src := range order {
		for _, dep := range graph[src] {
			remapped[i] = append(remapped[i], pos[dep])
		}
		if len(remapped[i]) > 1 {
			sort.Ints(remapped[i])
		}
	}
	return remapped
}
