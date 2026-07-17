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
	"strings"
	"sync/atomic"
	"testing"
)

func makeDAGTask(id int, order *[]int, mu *atomic.Int64) Task {
	return AcquireTask(func(arg uintptr, extra uintptr) error {
		*order = append(*order, id)
		mu.Add(1)
		return nil
	}, 0, 0)
}

func TestDAGSchedulerRespectsDependencies(t *testing.T) {
	sched := NewDAGScheduler(2, 8)
	var order []int
	var mu atomic.Int64
	a, err := sched.Submit(makeDAGTask(1, &order, &mu), nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := sched.Submit(makeDAGTask(2, &order, &mu), []int{a})
	if err != nil {
		t.Fatal(err)
	}
	_, err = sched.Submit(makeDAGTask(3, &order, &mu), []int{b})
	if err != nil {
		t.Fatal(err)
	}
	if err := sched.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if mu.Load() != 3 {
		t.Fatalf("expected 3 tasks run, got %d", mu.Load())
	}
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("unexpected run order %v", order)
	}
}

func TestDAGSchedulerReturnsCycleError(t *testing.T) {
	sched := NewDAGScheduler(1, 4)
	_, err := sched.Submit(AcquireTask(func(arg uintptr, extra uintptr) error { return nil }, 0, 0), []int{1})
	if err == nil {
		t.Fatal("expected invalid dependency error")
	}
}

func TestDAGSchedulerInvalidDependencyDoesNotLeakNode(t *testing.T) {
	sched := NewDAGScheduler(1, 4)
	a, err := sched.Submit(AcquireTask(func(arg uintptr, extra uintptr) error { return nil }, 0, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sched.Submit(AcquireTask(func(arg uintptr, extra uintptr) error { return nil }, 0, 0), []int{1})
	if err == nil {
		t.Fatal("expected invalid dependency error")
	}
	b, err := sched.Submit(AcquireTask(func(arg uintptr, extra uintptr) error { return nil }, 0, 0), []int{a})
	if err != nil {
		t.Fatal(err)
	}
	if b != 1 {
		t.Fatalf("expected second task index 1, got %d", b)
	}
}

func TestDAGSchedulerTaskErrorStopsDependents(t *testing.T) {
	sched := NewDAGScheduler(2, 4)
	root, err := sched.Submit(AcquireTask(func(arg uintptr, extra uintptr) error {
		return errors.New("task failed")
	}, 0, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	var ran atomic.Int64
	_, err = sched.Submit(AcquireTask(func(arg uintptr, extra uintptr) error {
		ran.Add(1)
		return nil
	}, 0, 0), []int{root})
	if err != nil {
		t.Fatal(err)
	}
	err = sched.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "task failed") {
		t.Fatalf("expected task failure, got %v", err)
	}
	if ran.Load() != 0 {
		t.Fatalf("expected dependent task not to run, got %d", ran.Load())
	}
}
