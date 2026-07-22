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

package fo

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

func TestPoolSubmitsTasks(t *testing.T) {
	pool := NewPool(2)
	defer pool.Stop()

	var count int32
	for i := 0; i < 8; i++ {
		pool.Submit(Task{Fn: func(arg unsafe.Pointer) error {
			atomic.AddInt32(&count, 1)
			return nil
		}})
	}

	pool.Stop()
	if got := atomic.LoadInt32(&count); got != 8 {
		t.Fatalf("expected 8 tasks, got %d", got)
	}
}

func TestPoolSubmitBatch(t *testing.T) {
	pool := NewPool(2)
	defer pool.Stop()

	var count int32
	tasks := make([]Task, 16)
	for i := range tasks {
		tasks[i] = Task{Fn: func(arg unsafe.Pointer) error {
			atomic.AddInt32(&count, 1)
			return nil
		}}
	}
	if !pool.SubmitBatch(tasks) {
		t.Fatalf("expected batch submission to succeed")
	}

	pool.Stop()
	if got := atomic.LoadInt32(&count); got != 16 {
		t.Fatalf("expected 16 task executions, got %d", got)
	}
}

func TestPoolExecutesAllSubmittedTasksUnderLoad(t *testing.T) {
	pool := NewPool(8)
	defer pool.Stop()

	const total = 4096
	var count int32
	for i := 0; i < total; i++ {
		if !pool.Submit(Task{Fn: func(arg unsafe.Pointer) error {
			atomic.AddInt32(&count, 1)
			return nil
		}}) {
			t.Fatalf("expected submit %d to succeed", i)
		}
	}

	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&count) < total && time.Now().Before(deadline) {
		runtime.Gosched()
	}

	if got := atomic.LoadInt32(&count); got != total {
		t.Fatalf("expected %d task executions, got %d", total, got)
	}
}
