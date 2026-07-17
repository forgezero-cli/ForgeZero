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
	"sync/atomic"
)

const numPriorities = 8

type priorityQueues struct {
	levels [numPriorities]*ringQueue
	mask   atomic.Uint64
	counts [numPriorities]atomic.Int64
}

func newPriorityQueues(queueSize int) *priorityQueues {
	pq := &priorityQueues{}
	for i := 0; i < numPriorities; i++ {
		pq.levels[i] = newRingQueue(queueSize)
	}
	return pq
}

func clampPriority(priority int) int {
	if priority < 0 {
		return 0
	}
	if priority >= numPriorities {
		return numPriorities - 1
	}
	return priority
}

func (pq *priorityQueues) enqueue(task Task, priority int) bool {
	level := clampPriority(priority)
	if !pq.levels[level].tryEnqueue(task) {
		return false
	}
	pq.counts[level].Add(1)
	bit := uint64(1) << uint(level)
	for {
		old := pq.mask.Load()
		if old&bit != 0 {
			break
		}
		if pq.mask.CompareAndSwap(old, old|bit) {
			break
		}
	}
	return true
}

func (pq *priorityQueues) spinEnqueue(task Task, priority int) {
	level := clampPriority(priority)
	pq.levels[level].spinEnqueue(task)
	pq.counts[level].Add(1)
	bit := uint64(1) << uint(level)
	for {
		old := pq.mask.Load()
		if old&bit != 0 {
			break
		}
		if pq.mask.CompareAndSwap(old, old|bit) {
			break
		}
	}
}

func (pq *priorityQueues) dequeue() (Task, bool) {
	m := pq.mask.Load()
	if m == 0 {
		return Task{}, false
	}
	for i := numPriorities - 1; i >= 0; i-- {
		bit := uint64(1) << uint(i)
		if m&bit == 0 {
			continue
		}
		task, ok := pq.levels[i].tryDequeue()
		if !ok {
			m = pq.mask.Load()
			continue
		}
		newCount := pq.counts[i].Add(-1)
		if newCount == 0 {
			for {
				old := pq.mask.Load()
				if old&bit == 0 {
					break
				}
				if pq.mask.CompareAndSwap(old, old&^bit) {
					break
				}
			}
		}
		return task, true
	}
	return Task{}, false
}

func (pq *priorityQueues) pending() uint64 {
	var total uint64
	for i := 0; i < numPriorities; i++ {
		total += uint64(pq.counts[i].Load())
	}
	return total
}
