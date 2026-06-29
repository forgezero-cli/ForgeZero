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
	slots []atomic.Pointer[taskSlot]
	cap   uint64
	head  atomic.Uint64
	tail  atomic.Uint64
}

func newRingQueue(capacity int) *ringQueue {
	if capacity < 2 {
		capacity = 2
	}
	capacity = nextPowerOfTwo(capacity)
	q := &ringQueue{
		slots: make([]atomic.Pointer[taskSlot], capacity),
		cap:   uint64(capacity),
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

func (q *ringQueue) tryEnqueue(slot *taskSlot) bool {
	for {
		tail := q.tail.Load()
		head := q.head.Load()
		if tail-head >= q.cap {
			return false
		}
		idx := tail & (q.cap - 1)
		if !q.slots[idx].CompareAndSwap(nil, slot) {
			runtime.Gosched()
			continue
		}
		if q.tail.CompareAndSwap(tail, tail+1) {
			return true
		}
		q.slots[idx].Store(nil)
	}
}

func (q *ringQueue) spinEnqueue(slot *taskSlot) {
	for !q.tryEnqueue(slot) {
		runtime.Gosched()
	}
}

func (q *ringQueue) tryDequeue() (*taskSlot, bool) {
	for {
		head := q.head.Load()
		tail := q.tail.Load()
		if head >= tail {
			return nil, false
		}
		idx := head & (q.cap - 1)
		slot := q.slots[idx].Load()
		if slot == nil {
			runtime.Gosched()
			continue
		}
		if q.head.CompareAndSwap(head, head+1) {
			q.slots[idx].Store(nil)
			return slot, true
		}
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
