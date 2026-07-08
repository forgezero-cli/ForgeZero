/*
 * Copyright (c) 2026 ForgeZero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even even the implied warranty of
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

type intRingQueue struct {
	slots []intSlot
	cap   uint64
	mask  uint64
	head  atomic.Uint64
	tail  atomic.Uint64
}

type intSlot struct {
	seq   atomic.Uint64
	value int
}

func newIntRingQueue(capacity int) *intRingQueue {
	if capacity < 2 {
		capacity = 2
	}
	capacity = nextPowerOfTwo(capacity)
	q := &intRingQueue{
		slots: make([]intSlot, capacity),
		cap:   uint64(capacity),
		mask:  uint64(capacity - 1),
	}
	for i := 0; i < capacity; i++ {
		q.slots[i].seq.Store(uint64(i))
	}
	return q
}

func (q *intRingQueue) tryEnqueue(value int) bool {
	for {
		tail := q.tail.Load()
		slot := &q.slots[tail&q.mask]
		seq := slot.seq.Load()
		if seq == tail {
			if q.tail.CompareAndSwap(tail, tail+1) {
				slot.value = value
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

func (q *intRingQueue) spinEnqueue(value int) {
	for !q.tryEnqueue(value) {
		runtime.Gosched()
	}
}

func (q *intRingQueue) tryDequeue() (int, bool) {
	for {
		head := q.head.Load()
		slot := &q.slots[head&q.mask]
		seq := slot.seq.Load()
		if seq == head+1 {
			if q.head.CompareAndSwap(head, head+1) {
				value := slot.value
				slot.seq.Store(head + q.cap)
				return value, true
			}
			continue
		}
		if seq <= head {
			return 0, false
		}
		runtime.Gosched()
	}
}
