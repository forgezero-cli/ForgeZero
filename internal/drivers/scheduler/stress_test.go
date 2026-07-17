/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 */

package scheduler

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
)

func TestSchedulerStressNoLeak(t *testing.T) {
	const tasks = 1000
	s := NewScheduler(4, 1024)
	var counter int64
	for i := 0; i < tasks; i++ {
		s.SubmitBlocking(AcquireTask(func(arg uintptr, extra uintptr) error {
			atomic.AddInt64(&counter, 1)
			return nil
		}, 0, 0), i%numPriorities)
	}
	var mStart runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&mStart)
	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}
	runtime.GC()
	var mEnd runtime.MemStats
	runtime.ReadMemStats(&mEnd)
	if counter != tasks {
		t.Fatalf("expected %d tasks, got %d", tasks, counter)
	}
	// allow small growth but detect large leaks
	if mEnd.HeapAlloc > mStart.HeapAlloc+10*1024*1024 {
		t.Fatalf("suspected memory leak: start=%d end=%d", mStart.HeapAlloc, mEnd.HeapAlloc)
	}
}

func TestDAGSchedulerStressDependencies(t *testing.T) {
	const nodes = 128
	sched := NewDAGScheduler(8, nodes)
	order := make([]int, 0, nodes)
	var added atomic.Int64
	for i := 0; i < nodes; i++ {
		deps := make([]int, 0, 2)
		if i > 0 {
			deps = append(deps, i-1)
		}
		if i > 1 {
			deps = append(deps, i-2)
		}
		idx, err := sched.Submit(AcquireTask(func(arg uintptr, extra uintptr) error {
			order = append(order, int(arg))
			added.Add(1)
			return nil
		}, uintptr(i), 0), deps)
		if err != nil {
			t.Fatalf("submit failed for node %d: %v", i, err)
		}
		if idx != i {
			t.Fatalf("expected node index %d, got %d", i, idx)
		}
	}
	if err := sched.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if added.Load() != nodes {
		t.Fatalf("expected %d tasks run, got %d", nodes, added.Load())
	}
	for i, id := range order {
		if id != i {
			t.Fatalf("unexpected order at %d: got %d", i, id)
		}
	}
}
