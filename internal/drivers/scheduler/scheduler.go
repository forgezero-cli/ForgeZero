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
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/forgezero-cli/ForgeZero/internal/drivers/concurrency"
	"github.com/forgezero-cli/ForgeZero/internal/drivers/workerpool"
)

var (
	errQueueFull  = errors.New("scheduler queue full")
	errAggregated = errors.New("scheduler task failures")
)

type Scheduler struct {
	poolSize  int
	queueSize int
	queues    *priorityQueues
	pending   atomic.Int64
	running   atomic.Bool
	errMu     concurrency.Mutex
	errs      []error
}

func NewScheduler(workerPoolSize int, queueSize int) *Scheduler {
	if workerPoolSize <= 0 {
		workerPoolSize = 1
	}
	if queueSize <= 0 {
		queueSize = workerPoolSize * 64
	}
	return &Scheduler{
		poolSize:  workerPoolSize,
		queueSize: queueSize,
		queues:    newPriorityQueues(queueSize),
	}
}

func (s *Scheduler) Submit(task Task, priority int) error {
	if task == nil {
		return nil
	}
	if s.running.Load() {
		return errQueueFull
	}
	if !s.queues.enqueue(task, priority) {
		return errQueueFull
	}
	s.pending.Add(1)
	return nil
}

func (s *Scheduler) SubmitBlocking(task Task, priority int) {
	if err := s.Submit(task, priority); err == nil {
		return
	}
	s.queues.spinEnqueue(task, priority)
	s.pending.Add(1)
}

func (s *Scheduler) recordError(err error) {
	if err == nil {
		return
	}
	s.errMu.Lock()
	s.errs = append(s.errs, err)
	s.errMu.Unlock()
}

func (s *Scheduler) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if !s.running.CompareAndSwap(false, true) {
		return errQueueFull
	}
	defer s.running.Store(false)

	total := s.pending.Load()
	if total == 0 {
		return nil
	}

	pool := workerpool.NewWorkerPool(s.poolSize)
	done := make(chan struct{})
	var doneOnce sync.Once

	for i := int64(0); i < total; i++ {
		select {
		case <-ctx.Done():
			pool.Stop()
			return ctx.Err()
		default:
		}
		var t Task 
		var ok bool 
		for {
			t, ok = s.queues.dequeue()
			if ok {
				break
			}
			select {
			case <-ctx.Done():
				pool.Stop()
				return ctx.Err()
			default:
			}
			runtime.Gosched()
		}
		task := t 
		pool.Submit(func(c context.Context) error {
			err := task(c)
			if err != nil {
				s.recordError(err)
			}
			if s.pending.Add(-1) == 0 {
				doneOnce.Do(func() { close(done) })
			}
			return nil
		})
	}

	select {
	case <-done:
	case <-ctx.Done():
		pool.Stop()
		return ctx.Err()
	}

	pool.Stop()

	s.errMu.Lock()
	defer s.errMu.Unlock()
	if len(s.errs) == 0 {
		return nil
	}
	if len(s.errs) == 1 {
		return s.errs[0]
	}
	return errAggregated
}
