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
		s.SubmitBlocking(func(ctx context.Context) error {
			atomic.AddInt64(&counter, 1)
			return nil
		}, i%numPriorities)
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
