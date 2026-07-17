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

type dagNode struct {
	task       Task
	deps       atomic.Int32
	dependents []int
}

type DAGScheduler struct {
	pool    *Scheduler
	nodes   []dagNode
	ready   *intRingQueue
	running atomic.Bool
	errOnce sync.Once
	err     error
	pending atomic.Int64
	cancel  context.CancelFunc
}

func NewDAGScheduler(workerPoolSize int, nodeCapacity int) *DAGScheduler {
	if nodeCapacity < 1 {
		nodeCapacity = 1
	}
	return &DAGScheduler{
		pool:  NewScheduler(workerPoolSize, nodeCapacity*2),
		nodes: make([]dagNode, 0, nodeCapacity),
		ready: newIntRingQueue(nodeCapacity),
	}
}

func (d *DAGScheduler) Submit(task Task, deps []int) (int, error) {
	if task.Fn == nil {
		return -1, errors.New("task required")
	}
	if d.running.Load() {
		return -1, errors.New("scheduler already running")
	}
	idx := len(d.nodes)
	for _, dep := range deps {
		if dep < 0 || dep >= idx {
			return -1, errors.New("invalid dependency index")
		}
	}

	node := dagNode{task: task}
	node.deps.Store(int32(len(deps)))
	d.nodes = append(d.nodes, node)
	for _, dep := range deps {
		d.nodes[dep].dependents = append(d.nodes[dep].dependents, idx)
	}
	return idx, nil
}

func (d *DAGScheduler) enqueueReady(index int) {
	d.ready.spinEnqueue(index)
}

func (d *DAGScheduler) setError(err error) {
	d.errOnce.Do(func() {
		d.err = err
		if d.cancel != nil {
			d.cancel()
		}
	})
}

func (d *DAGScheduler) worker(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		idx, ok := d.ready.tryDequeue()
		if !ok {
			if d.err != nil || d.pending.Load() == 0 {
				return
			}
			runtime.Gosched()
			continue
		}
		if d.err != nil {
			return
		}
		node := &d.nodes[idx]
		if err := node.task.Fn(node.task.Arg, node.task.Extra); err != nil {
			d.setError(err)
			return
		}
		ReleaseTask(node.task)
		for _, next := range node.dependents {
			if d.nodes[next].deps.Add(-1) == 0 {
				d.enqueueReady(next)
			}
		}
		if d.pending.Add(-1) == 0 {
			return
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
	d.cancel = cancel
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
			d.worker(ctx)
		}()
	}
	wg.Wait()
	if d.err != nil {
		return d.err
	}
	return nil
}
