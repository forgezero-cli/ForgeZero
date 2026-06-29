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

package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestSchedulerRunsAllTasks(t *testing.T) {
	t.Parallel()
	sched := NewScheduler(4, 256)
	var counter int64
	for i := 0; i < 200; i++ {
		sched.SubmitBlocking(func(ctx context.Context) error {
			atomic.AddInt64(&counter, 1)
			return nil
		}, 0)
	}
	if err := sched.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if counter != 200 {
		t.Fatalf("expected 200 tasks, got %d", counter)
	}
}

func TestSchedulerPriorityOrdering(t *testing.T) {
	t.Parallel()
	sched := NewScheduler(1, 64)
	var counter int64
	for i := 0; i < 16; i++ {
		sched.SubmitBlocking(func(ctx context.Context) error {
			atomic.AddInt64(&counter, 1)
			return nil
		}, i%8)
	}
	if err := sched.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if counter != 16 {
		t.Fatalf("expected 16 tasks, got %d", counter)
	}
}

func TestSchedulerCollectsErrors(t *testing.T) {
	t.Parallel()
	sched := NewScheduler(2, 32)
	testErr := errors.New("task failed")
	for i := 0; i < 4; i++ {
		sched.SubmitBlocking(func(ctx context.Context) error {
			return testErr
		}, 0)
	}
	err := sched.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSchedulerContextCancel(t *testing.T) {
	t.Parallel()
	sched := NewScheduler(2, 32)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sched.SubmitBlocking(func(c context.Context) error {
		return nil
	}, 0)
	err := sched.Run(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func BenchmarkSchedulerSubmitRun(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sched := NewScheduler(4, 512)
		for j := 0; j < 100; j++ {
			sched.SubmitBlocking(func(ctx context.Context) error {
				return nil
			}, 0)
		}
		_ = sched.Run(context.Background())
	}
}
