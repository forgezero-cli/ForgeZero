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

package concurrency

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestMutexLockUnlock(t *testing.T) {
	t.Parallel()
	var m Mutex
	var counter int64
	done := make(chan struct{})
	go func() {
		m.Lock()
		atomic.AddInt64(&counter, 1)
		m.Unlock()
		close(done)
	}()
	m.Lock()
	atomic.AddInt64(&counter, 1)
	m.Unlock()
	<-done
	if counter != 2 {
		t.Fatalf("expected counter=2, got %d", counter)
	}
}

func TestWaitGroupWait(t *testing.T) {
	t.Parallel()
	var wg WaitGroup
	var counter int64
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&counter, 1)
		}()
	}
	wg.Wait()
	if counter != 32 {
		t.Fatalf("expected counter=32, got %d", counter)
	}
}

func TestSemaphoreAcquireRelease(t *testing.T) {
	t.Parallel()
	sem := NewSemaphore(4)
	if err := sem.Acquire(2); err != nil {
		t.Fatal(err)
	}
	if err := sem.Acquire(2); err != nil {
		t.Fatal(err)
	}
	sem.Release(4)
	if err := sem.Acquire(4); err != nil {
		t.Fatal(err)
	}
}

func TestSemaphoreAcquireContextCancel(t *testing.T) {
	t.Parallel()
	sem := NewSemaphore(0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sem.AcquireContext(ctx, 1); err == nil {
		t.Fatal("expected cancellation error")
	}
}

func BenchmarkSemaphoreAcquireRelease(b *testing.B) {
	sem := NewSemaphore(b.N + 1)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = sem.Acquire(1)
		sem.Release(1)
	}
}

func BenchmarkMutexLockUnlock(b *testing.B) {
	var m Mutex
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Lock()
		m.Unlock()
	}
}
