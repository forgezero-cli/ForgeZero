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
	"time"
	"unsafe"
)

type Task struct {
	Fn  func(arg unsafe.Pointer) error
	Arg unsafe.Pointer
}

func (t Task) Run() error {
	if t.Fn == nil {
		return nil
	}
	return t.Fn(t.Arg)
}

type ringSlot struct {
	sequence uint64
	task     Task
}

type ringBuffer struct {
	cap   uint64
	mask  uint64
	head  uint64
	tail  uint64
	slots []ringSlot
}

type worker struct {
	queue       ringBuffer
	owner       *Pool
	id          int
	idleBackoff uint32
	_pad        [56]byte
}

type Pool struct {
	workers []*worker
	next    uint64
	stop    uint32
	live    uint64
}

var globalPool atomic.Pointer[Pool]

func NewPool(size int) *Pool {
	if size <= 0 {
		size = runtime.NumCPU()
	}
	if size > 1<<20 {
		size = 1 << 20
	}
	cap := uint64(1 << 12)
	workers := make([]*worker, size)
	for i := 0; i < size; i++ {
		workers[i] = &worker{queue: ringBuffer{cap: cap, mask: cap - 1, slots: make([]ringSlot, cap)}, id: i}
	}
	p := &Pool{workers: workers}
	for i := range workers {
		workers[i].owner = p
		atomic.AddUint64(&p.live, 1)
		go p.worker(i)
	}
	return p
}

func InitGlobalPool(size int) *Pool {
	if p := globalPool.Load(); p != nil {
		return p
	}
	p := NewPool(size)
	if globalPool.CompareAndSwap(nil, p) {
		return p
	}
	p.Stop()
	return globalPool.Load()
}

func (p *Pool) Submit(task Task) bool {
	if p == nil || task.Fn == nil || atomic.LoadUint32(&p.stop) != 0 {
		return false
	}
	w := p.queueForSubmit()
	if w == nil {
		return false
	}
	for {
		tail := atomic.LoadUint64(&w.queue.tail)
		if tail-atomic.LoadUint64(&w.queue.head) >= w.queue.cap {
			runtime.Gosched()
			continue
		}
		if atomic.CompareAndSwapUint64(&w.queue.tail, tail, tail+1) {
			idx := int(tail & w.queue.mask)
			slot := &w.queue.slots[idx]
			slot.task = task
			atomic.StoreUint64(&slot.sequence, tail+1)
			return true
		}
	}
}

func (p *Pool) SubmitBatch(tasks []Task) bool {
	if p == nil {
		return false
	}
	for i := range tasks {
		if !p.Submit(tasks[i]) {
			return false
		}
	}
	return true
}

func (p *Pool) Stop() {
	if p == nil {
		return
	}
	if p == globalPool.Load() {
		return
	}
	if atomic.CompareAndSwapUint32(&p.stop, 0, 1) {
		for atomic.LoadUint64(&p.live) != 0 {
			runtime.Gosched()
		}
	}
}

func (p *Pool) queueForSubmit() *worker {
	if p == nil || len(p.workers) == 0 {
		return nil
	}
	idx := atomic.AddUint64(&p.next, 1) - 1
	if len(p.workers) == 0 {
		return nil
	}
	return p.workers[int(idx%uint64(len(p.workers)))]
}

func (p *Pool) worker(id int) {
	defer atomic.AddUint64(&p.live, ^uint64(0))
	w := p.workers[id]
	for {
		if atomic.LoadUint32(&p.stop) != 0 && atomic.LoadUint64(&w.queue.head) >= atomic.LoadUint64(&w.queue.tail) {
			return
		}
		if task, ok := w.popLocal(); ok {
			w.idleBackoff = 0
			_ = task.Run()
			continue
		}
		if task, ok := w.steal(); ok {
			w.idleBackoff = 0
			_ = task.Run()
			continue
		}

		if w.idleBackoff == 0 {
			runtime.Gosched()
			w.idleBackoff = 1
			continue
		}

		pause := time.Microsecond
		for i := uint32(0); i < w.idleBackoff && pause < time.Millisecond; i++ {
			pause *= 2
		}
		if pause > time.Millisecond {
			pause = time.Millisecond
		}
		time.Sleep(pause)
		if w.idleBackoff < 8 {
			w.idleBackoff++
		}
	}
}

func (w *worker) popLocal() (Task, bool) {
	for {
		head := atomic.LoadUint64(&w.queue.head)
		tail := atomic.LoadUint64(&w.queue.tail)
		if head >= tail {
			return Task{}, false
		}
		idx := int(head & w.queue.mask)
		slot := &w.queue.slots[idx]
		if atomic.LoadUint64(&slot.sequence) != head+1 {
			runtime.Gosched()
			continue
		}
		if atomic.CompareAndSwapUint64(&w.queue.head, head, head+1) {
			task := slot.task
			slot.task = Task{}
			atomic.StoreUint64(&slot.sequence, head+w.queue.cap)
			return task, true
		}
	}
}

func (w *worker) steal() (Task, bool) {
	if w.owner == nil || len(w.owner.workers) <= 1 {
		return Task{}, false
	}
	for step := 0; step < len(w.owner.workers); step++ {
		victimID := (w.id + step + 1) % len(w.owner.workers)
		victim := w.owner.workers[victimID]
		if victim == nil || victim == w {
			continue
		}
		if task, ok := victim.popLocal(); ok {
			return task, true
		}
	}
	return Task{}, false
}
