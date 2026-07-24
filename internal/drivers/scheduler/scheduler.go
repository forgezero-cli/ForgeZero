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
)

var (
	errQueueFull  = errors.New("scheduler queue full")
	errAggregated = errors.New("scheduler task failures")
)

type runContextHolder struct {
	ctx context.Context
}

var globalRunContext atomic.Pointer[runContextHolder]

type Scheduler struct {
	poolSize    int
	queueSize   int
	queues      *priorityQueues
	workers     []*priorityQueues
	nextSubmit  atomic.Uint64
	pending     atomic.Int64
	pendingMu   sync.Mutex
	pendingCond *sync.Cond
	running     atomic.Bool
	errMu       sync.Mutex
	errs        []error
	ctxHolder   runContextHolder
}

func NewScheduler(workerPoolSize int, queueSize int) *Scheduler {
	if workerPoolSize <= 0 {
		workerPoolSize = runtime.GOMAXPROCS(0)
		if workerPoolSize <= 0 {
			workerPoolSize = 1
		}
	}
	if queueSize <= 0 {
		queueSize = workerPoolSize * 64
	}
	s := &Scheduler{
		poolSize:  workerPoolSize,
		queueSize: queueSize,
		queues:    newPriorityQueues(queueSize),
		workers:   make([]*priorityQueues, workerPoolSize),
	}
	s.pendingCond = sync.NewCond(&s.pendingMu)

	for i := 0; i < workerPoolSize; i++ {
		s.workers[i] = newPriorityQueues(queueSize)
	}
	for i := 0; i < workerPoolSize; i++ {
		go s.workerLoop(i)
	}
	return s
}

func (s *Scheduler) Submit(task Task, priority int) error {
	if task.Fn == nil {
		return nil
	}
	if s.running.Load() {
		return errQueueFull
	}
	if s.poolSize <= 0 {
		return errQueueFull
	}
	idx := int((s.nextSubmit.Add(1) - 1) % uint64(s.poolSize))
	if !s.workers[idx].enqueue(task, priority) {
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

func (s *Scheduler) dequeueForWorker(idx int) (Task, bool) {
	if task, ok := s.workers[idx].dequeue(); ok {
		return task, true
	}
	n := s.poolSize
	for i := 1; i < n; i++ {
		j := (idx + i) % n
		if task, ok := s.workers[j].dequeue(); ok {
			return task, true
		}
	}
	return s.queues.dequeue()
}

func CurrentContext() context.Context {
	if ptr := globalRunContext.Load(); ptr != nil {
		return ptr.ctx
	}
	return context.Background()
}

func (s *Scheduler) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if !s.running.CompareAndSwap(false, true) {
		return errQueueFull
	}

	s.errMu.Lock()
	s.errs = s.errs[:0]
	s.errMu.Unlock()

	s.ctxHolder.ctx = ctx
	globalRunContext.Store(&s.ctxHolder)

	if s.pending.Load() == 0 {
		s.running.Store(false)
		globalRunContext.Store(nil)
		return nil
	}

	s.pendingMu.Lock()
	for s.pending.Load() > 0 {
		if err := ctx.Err(); err != nil {
			s.pendingMu.Unlock()
			s.running.Store(false)
			globalRunContext.Store(nil)
			return err
		}
		s.pendingCond.Wait()
	}
	s.pendingMu.Unlock()

	s.running.Store(false)
	globalRunContext.Store(nil)
	if len(s.errs) == 0 {
		return nil
	}
	if len(s.errs) == 1 {
		return s.errs[0]
	}
	return errAggregated
}

func (s *Scheduler) workerLoop(workerIdx int) {
	for {
		if !s.running.Load() {
			runtime.Gosched()
			continue
		}
		task, ok := s.dequeueForWorker(workerIdx)
		if !ok {
			runtime.Gosched()
			continue
		}
		if err := task.Fn(task.Arg, task.Extra); err != nil {
			s.recordError(err)
		}
		ReleaseTask(task)
		if s.pending.Add(-1) == 0 {
			s.pendingMu.Lock()
			s.pendingCond.Broadcast()
			s.pendingMu.Unlock()
		}
	}
}
