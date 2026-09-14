/*
 *   Copyright (c) 2026 forgezero-cli
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
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	ch "github.com/forgezero-cli/ForgeZero/internal/drivers/chan"
	"github.com/forgezero-cli/ForgeZero/internal/drivers/thread"
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
	task     *Task
}

type ringBuffer struct {
	cap      uint64
	mask     uint64
	head     uint64
	_padHead [56]byte
	tail     uint64
	_padTail [56]byte
	slots    []ringSlot
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
	tp      *thread.Pool
	publicQ *ch.MPSC
}

var globalPool atomic.Pointer[Pool]
var taskPool sync.Pool

func init() {
	taskPool.New = func() any { return &Task{} }
}

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
	p.tp = thread.NewPool(size)
	p.publicQ = ch.NewMPSC(1 << 12)
	for i := range workers {
		workers[i].owner = p
		atomic.AddUint64(&p.live, 1)
		idx := i
		p.tp.Submit(func() { p.worker(idx) })
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
	if p == nil || atomic.LoadUint32(&p.stop) != 0 || task.Fn == nil {
		return false
	}
	w := p.queueForSubmit()
	if w != nil {
		spin := 0
		for {
			tail := atomic.LoadUint64(&w.queue.tail)
			head := atomic.LoadUint64(&w.queue.head)
			if tail-head < w.queue.cap {
				if atomic.CompareAndSwapUint64(&w.queue.tail, tail, tail+1) {
					idx := int(tail & w.queue.mask)
					slot := &w.queue.slots[idx]
					tp := taskPool.Get().(*Task)
					*tp = task
					slot.task = tp
					atomic.StoreUint64(&slot.sequence, tail+1)
					return true
				}
			}
			spin++
			if spin < 8 {
				runtime.Gosched()
				continue
			}
			break
		}
	}
	if p.publicQ != nil {
		return p.publicQ.Enqueue(task)
	}
	return false
}

func (p *Pool) reserveBatch(w *worker, tasks []Task) int {
	if w == nil || len(tasks) == 0 {
		return 0
	}
	for spin := 0; ; spin++ {
		tail := atomic.LoadUint64(&w.queue.tail)
		head := atomic.LoadUint64(&w.queue.head)
		used := tail - head
		if used >= w.queue.cap {
			if spin < 8 {
				runtime.Gosched()
				continue
			}
			return 0
		}
		available := int(w.queue.cap - used)
		count := len(tasks)
		if count > available {
			count = available
		}
		if count == 0 {
			return 0
		}
		if atomic.CompareAndSwapUint64(&w.queue.tail, tail, tail+uint64(count)) {
			for i := 0; i < count; i++ {
				idx := int((tail + uint64(i)) & w.queue.mask)
				slot := &w.queue.slots[idx]
				tp := taskPool.Get().(*Task)
				*tp = tasks[i]
				slot.task = tp
				atomic.StoreUint64(&slot.sequence, tail+uint64(i)+1)
			}
			return count
		}
		if spin < 8 {
			runtime.Gosched()
			continue
		}
		time.Sleep(time.Microsecond)
	}
}

func (p *Pool) SubmitBatch(tasks []Task) bool {
	if p == nil || len(tasks) == 0 || atomic.LoadUint32(&p.stop) != 0 {
		return false
	}
	start := 0
	for start < len(tasks) {
		w := p.queueForSubmit()
		if w == nil {
			return false
		}
		count := p.reserveBatch(w, tasks[start:])
		if count == 0 {
			continue
		}
		start += count
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
		if p.tp != nil {
			p.tp.Stop()
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
		spun := 0
		handled := false
		for spun < 64 {
			if task, ok := w.popLocal(); ok {
				w.idleBackoff = 0
				_ = task.Run()
				handled = true
				break
			}
			if task, ok := w.steal(); ok {
				w.idleBackoff = 0
				_ = task.Run()
				handled = true
				break
			}
			runtime.Gosched()
			spun++
		}
		if handled {
			continue
		}
		pause := time.Microsecond << (w.idleBackoff & 7)
		if pause > time.Millisecond {
			pause = time.Millisecond
		}
		time.Sleep(pause)
		if w.idleBackoff < 255 {
			w.idleBackoff++
		}
	}
}

func (w *worker) popLocal() (Task, bool) {
	for spin := 0; ; spin++ {
		head := atomic.LoadUint64(&w.queue.head)
		tail := atomic.LoadUint64(&w.queue.tail)
		if head >= tail {
			return Task{}, false
		}
		idx := int(head & w.queue.mask)
		slot := &w.queue.slots[idx]
		if atomic.LoadUint64(&slot.sequence) != head+1 {
			if spin < 8 {
				runtime.Gosched()
			} else if spin < 16 {
				time.Sleep(time.Microsecond)
			} else {
				time.Sleep(10 * time.Microsecond)
			}
			continue
		}
		if atomic.CompareAndSwapUint64(&w.queue.head, head, head+1) {
			tp := slot.task
			slot.task = nil
			atomic.StoreUint64(&slot.sequence, head+w.queue.cap)
			if tp == nil {
				return Task{}, false
			}
			task := *tp
			taskPool.Put(tp)
			return task, true
		}
	}
}

func (w *worker) steal() (Task, bool) {
	if w.owner == nil || w.owner.publicQ == nil || len(w.owner.workers) <= 1 {
		return Task{}, false
	}
	if atomic.LoadUint64(&w.queue.tail) > atomic.LoadUint64(&w.queue.head) {
		return Task{}, false
	}
	for step := 0; step < len(w.owner.workers); step++ {
		victimID := (w.id + step + 1) % len(w.owner.workers)
		victim := w.owner.workers[victimID]
		if victim == nil || victim == w {
			continue
		}
		if task, ok := victim.popLocal(); ok {
			if task.Fn == nil {
				continue
			}
			return task, true
		}
	}
	if w.owner != nil && w.owner.publicQ != nil {
		var batchArr [64]Task
		batch := batchArr[:0]
		for i := 0; i < 64; i++ {
			if v, ok := w.owner.publicQ.Dequeue(); ok {
				if t, ok2 := v.(Task); ok2 {
					batch = append(batch, t)
					continue
				}
			}
			break
		}
		if len(batch) > 0 {
			if cnt := w.owner.reserveBatch(w, batch); cnt > 0 {
				if task, ok := w.popLocal(); ok {
					return task, true
				}
			}
		}
	}
	return Task{}, false
}
