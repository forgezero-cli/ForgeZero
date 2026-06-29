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

const numPriorities = 8

type priorityQueues struct {
	levels [numPriorities]*ringQueue
}

func newPriorityQueues(queueSize int) *priorityQueues {
	if queueSize <= 0 {
		queueSize = 8 
	}
	
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
	slot := acquireTaskSlot()
	slot.task = task
	level := clampPriority(priority)
	if pq.levels[level].tryEnqueue(slot) {
		return true
	}
	releaseTaskSlot(slot)
	return false
}

func (pq *priorityQueues) spinEnqueue(task Task, priority int) {
	slot := acquireTaskSlot()
	slot.task = task
	level := clampPriority(priority)
	pq.levels[level].spinEnqueue(slot)
}

func (pq *priorityQueues) dequeue() (Task, bool) {
	for i := numPriorities - 1; i >= 0; i-- {
		slot, ok := pq.levels[i].tryDequeue()
		if ok && slot != nil {
			task := slot.task
			releaseTaskSlot(slot)
			return task, true
		}
	}
	return nil, false
}

func (pq *priorityQueues) pending() uint64 {
	var total uint64
	for i := 0; i < numPriorities; i++ {
		total += pq.levels[i].len()
	}
	return total
}
