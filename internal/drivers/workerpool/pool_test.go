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

package workerpool

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestWorkerPoolExecutesTasks(t *testing.T) {
	t.Parallel()
	pool := NewWorkerPool(4)
	var counter int64
	for i := 0; i < 100; i++ {
		pool.Submit(func(ctx context.Context) error {
			atomic.AddInt64(&counter, 1)
			return nil
		})
	}
	pool.Stop()
	if counter != 100 {
		t.Fatalf("expected 100 tasks, got %d", counter)
	}
}

func TestWorkerPoolConcurrentSubmit(t *testing.T) {
	t.Parallel()
	pool := NewWorkerPool(8)
	var counter int64
	var submitWg sync.WaitGroup
	for i := 0; i < 8; i++ {
		submitWg.Add(1)
		go func() {
			defer submitWg.Done()
			for j := 0; j < 50; j++ {
				pool.Submit(func(ctx context.Context) error {
					atomic.AddInt64(&counter, 1)
					return nil
				})
			}
		}()
	}
	submitWg.Wait()
	pool.Stop()
	if counter != 400 {
		t.Fatalf("expected 400 tasks, got %d", counter)
	}
}

func BenchmarkWorkerPoolSubmit(b *testing.B) {
	pool := NewWorkerPool(4)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(func(ctx context.Context) error {
			return nil
		})
	}
	pool.Stop()
}
