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
	"runtime"
	"sync/atomic"
)

type ringQueue struct {
	slots []slot
	cap   uint64
	mask  uint64
	head  atomic.Uint64
	tail  atomic.Uint64
}

type slot struct {
	seq  atomic.Uint64
	task Task
}

func newRingQueue(capacity int) *ringQueue {
	if capacity < 2 {
		capacity = 2
	}
	capacity = nextPowerOfTwo(capacity)
	q := &ringQueue{
		slots: make([]slot, capacity),
		cap:   uint64(capacity),
		mask:  uint64(capacity - 1),
	}
	for i := 0; i < capacity; i++ {
		q.slots[i].seq.Store(uint64(i))
	}
	return q
}

func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

func (q *ringQueue) tryEnqueue(task Task) bool {
	for {
		tail := q.tail.Load()
		slot := &q.slots[tail&q.mask]
		seq := slot.seq.Load()
		if seq == tail {
			if q.tail.CompareAndSwap(tail, tail+1) {
				slot.task = task
				slot.seq.Store(tail + 1)
				return true
			}
			continue
		}
		if seq < tail {
			return false
		}
		runtime.Gosched()
	}
}

func (q *ringQueue) spinEnqueue(task Task) {
	for !q.tryEnqueue(task) {
		runtime.Gosched()
	}
}

func (q *ringQueue) tryDequeue() (Task, bool) {
	for {
		head := q.head.Load()
		slot := &q.slots[head&q.mask]
		seq := slot.seq.Load()
		if seq == head+1 {
			if q.head.CompareAndSwap(head, head+1) {
				task := slot.task
				slot.task = Task{}
				slot.seq.Store(head + q.cap)
				return task, true
			}
			continue
		}
		if seq <= head {
			return Task{}, false
		}
		runtime.Gosched()
	}
}

func (q *ringQueue) len() uint64 {
	tail := q.tail.Load()
	head := q.head.Load()
	if tail >= head {
		return tail - head
	}
	return 0
}
