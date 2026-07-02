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
	"sync"
	"sync/atomic"
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
	errMu     sync.Mutex
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

	errCh := make(chan error, total)
	var wg sync.WaitGroup

	for i := 0; i < s.poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				task, ok := s.queues.dequeue()
				if !ok {
					return
				}
				err := task(ctx)
				if err != nil {
					errCh <- err
				}
				if s.pending.Add(-1) == 0 {
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	var errs []error
	for {
		select {
		case err, ok := <-errCh:
			if !ok {
				if len(errs) == 0 {
					return nil
				}
				if len(errs) == 1 {
					return errs[0]
				}
				return errAggregated
			}
			errs = append(errs, err)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}