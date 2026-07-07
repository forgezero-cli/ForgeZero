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

package lfqueue

import (
	"sync/atomic"
	"unsafe"
)

type node struct {
	val  unsafe.Pointer
	next atomic.Pointer[node]
}

type Queue struct{
	head atomic.Pointer[node]
	tail atomic.Pointer[node]
}

func New() *Queue {
	stub := &node{}
	q := &Queue{}
	q.head.Store(stub)
	q.tail.Store(stub)
	return q
}

func (q *Queue) Enqueue(v any) {
	n := &node{val: unsafe.Pointer(&v)}
	for {
		t := q.tail.Load()
		next := t.next.Load()
		if t == q.tail.Load() {
			if next == nil {
				if t.next.CompareAndSwap(nil, n) {
					q.tail.CompareAndSwap(t, n)
					return
				}
			} else {
				q.tail.CompareAndSwap(t, next)
			}
		}
	}
}

func (q *Queue) Dequeue() (any, bool) {
	for {
		h := q.head.Load()
		t := q.tail.Load()
		next := h.next.Load()
		if h == q.head.Load() {
			if h == t {
				if next == nil {
					return nil, false
				}
				q.tail.CompareAndSwap(t, next)
			} else {
				valPtr := next.val
				if q.head.CompareAndSwap(h, next) {
					if valPtr == nil {
						return nil, false
					}
					v := *(*any)(valPtr)
					return v, true
				}
			}
		}
	}
}
