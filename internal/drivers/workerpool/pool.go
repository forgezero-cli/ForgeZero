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
)

type WorkerPool struct {
	tasks    chan Task
	size     int
	wg       sync.WaitGroup
	stopped  atomic.Bool
	stopOnce sync.Once
}

func NewWorkerPool(size int) *WorkerPool {
	if size <= 0 {
		size = 1
	}
	capacity := size * 64
	if capacity < 64 {
		capacity = 64
	}
	p := &WorkerPool{
		tasks: make(chan Task, capacity),
		size:  size,
	}
	p.wg.Add(size)
	for i := 0; i < size; i++ {
		go p.workerLoop()
	}
	return p
}

func (p *WorkerPool) Submit(task Task) {
	if task == nil || p.stopped.Load() {
		return
	}
	p.tasks <- task
}

func (p *WorkerPool) Stop() {
	p.stopOnce.Do(func() {
		p.stopped.Store(true)
		close(p.tasks)
		p.wg.Wait()
	})
}

func (p *WorkerPool) workerLoop() {
	defer p.wg.Done()
	ctx := context.Background()
	for task := range p.tasks {
		if task != nil {
			_ = task(ctx)
		}
	}
}
