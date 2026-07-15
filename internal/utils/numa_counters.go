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

package utils

import (
	"os"
	"runtime"
	"sync/atomic"
	"unsafe"
)

type NumaCounters struct {
	shards []atomic.Int64
	n      int
}

func newNumaShards() int {
	dirs, err := os.ReadDir("/sys/devices/system/node/")
	if err != nil || len(dirs) == 0 {
		p := runtime.GOMAXPROCS(0)
		if p <= 0 {
			p = 1
		}
		return p
	}
	count := 0
	for _, d := range dirs {
		if d.IsDir() && len(d.Name()) > 4 && d.Name()[:4] == "node" {
			count++
		}
	}
	if count == 0 {
		p := runtime.GOMAXPROCS(0)
		if p <= 0 {
			p = 1
		}
		return p
	}
	return count
}

func NewNumaCounters() *NumaCounters {
	n := newNumaShards()
	c := &NumaCounters{n: n}
	c.shards = make([]atomic.Int64, n)
	return c
}

func (c *NumaCounters) shardIndex() int {
	var x int
	p := uintptr(unsafe.Pointer(&x))
	return int(p % uintptr(c.n))
}

func (c *NumaCounters) Add(delta int64) {
	idx := c.shardIndex()
	c.shards[idx].Add(delta)
}

func (c *NumaCounters) Inc() { c.Add(1) }

func (c *NumaCounters) Load() int64 {
	var sum int64
	for i := 0; i < c.n; i++ {
		sum += c.shards[i].Load()
	}
	return sum
}
