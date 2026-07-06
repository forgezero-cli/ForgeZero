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

type dagNode struct {
	task       Task
	deps       atomic.Int32
	dependents []int
}

type DAGScheduler struct {
	pool      *Scheduler
	nodes     []dagNode
	ready     chan int
	running   atomic.Bool
	errOnce   sync.Once
	err       error
	pending   atomic.Int64
	closeOnce sync.Once
}

func NewDAGScheduler(workerPoolSize int, nodeCapacity int) *DAGScheduler {
	if nodeCapacity < 1 {
		nodeCapacity = 1
	}
	return &DAGScheduler{
		pool:  NewScheduler(workerPoolSize, nodeCapacity*2),
		nodes: make([]dagNode, 0, nodeCapacity),
		ready: make(chan int, nodeCapacity),
	}
}

func (d *DAGScheduler) Submit(task Task, deps []int) (int, error) {
	if task == nil {
		return -1, errors.New("task required")
	}
	if d.running.Load() {
		return -1, errors.New("scheduler already running")
	}
	idx := len(d.nodes)
	node := dagNode{task: task}
	node.deps.Store(int32(len(deps)))
	d.nodes = append(d.nodes, node)
	for _, dep := range deps {
		if dep < 0 || dep >= idx {
			return -1, errors.New("invalid dependency index")
		}
		d.nodes[dep].dependents = append(d.nodes[dep].dependents, idx)
	}
	return idx, nil
}

func (d *DAGScheduler) enqueueReady(index int) {
	defer func() {
		recover()
	}()
	d.ready <- index
}

func (d *DAGScheduler) closeReady() {
	d.closeOnce.Do(func() {
		close(d.ready)
	})
}

func (d *DAGScheduler) setError(err error, cancel context.CancelFunc) {
	d.errOnce.Do(func() {
		d.err = err
		if cancel != nil {
			cancel()
		}
		d.closeReady()
	})
}

func (d *DAGScheduler) worker(ctx context.Context, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		case idx, ok := <-d.ready:
			if !ok {
				return
			}
			if d.err != nil {
				return
			}
			node := &d.nodes[idx]
			if err := node.task(ctx); err != nil {
				d.setError(err, cancel)
				return
			}
			for _, next := range node.dependents {
				if d.nodes[next].deps.Add(-1) == 0 {
					d.enqueueReady(next)
				}
			}
			if d.pending.Add(-1) == 0 {
				d.closeReady()
			}
		}
	}
}

func (d *DAGScheduler) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if !d.running.CompareAndSwap(false, true) {
		return errors.New("scheduler already running")
	}
	defer d.running.Store(false)
	if len(d.nodes) == 0 {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	d.closeOnce = sync.Once{}
	d.pending.Store(int64(len(d.nodes)))
	for i := range d.nodes {
		if d.nodes[i].deps.Load() == 0 {
			d.enqueueReady(i)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < d.pool.poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.worker(ctx, cancel)
		}()
	}
	wg.Wait()
	if d.err != nil {
		return d.err
	}
	return nil
}
