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

package utils

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestAtomicBitmapOperations(t *testing.T) {
	bitmap := NewAtomicBitmap(130)
	bitmap.SetAtomic(129)
	if !bitmap.TestAndSet(129) {
		t.Fatal("TestAndSet did not observe the set bit")
	}
	if bitmap.TestAndSet(128) {
		t.Fatal("TestAndSet observed an unset bit")
	}
	bitmap.ClearAtomic(129)
	if bitmap.TestAndSet(129) {
		t.Fatal("ClearAtomic did not clear the bit")
	}
	if bitmap.TestAndSet(-1) || bitmap.TestAndSet(130) {
		t.Fatal("out-of-range bit reported as set")
	}
}

func TestAtomicBitmapTestAndSetSingleWinner(t *testing.T) {
	bitmap := NewAtomicBitmap(64)
	var winners atomic.Int32
	var group sync.WaitGroup
	for worker := 0; worker < 64; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if !bitmap.TestAndSet(23) {
				winners.Add(1)
			}
		}()
	}
	group.Wait()
	if got := winners.Load(); got != 1 {
		t.Fatalf("TestAndSet winners = %d, want 1", got)
	}
}

func TestAtomicBitmapOperationsDoNotAllocate(t *testing.T) {
	bitmap := NewAtomicBitmap(64)
	allocs := testing.AllocsPerRun(100, func() {
		bitmap.SetAtomic(3)
		bitmap.ClearAtomic(3)
	})
	if allocs != 0 {
		t.Fatalf("bitmap operation allocations = %v, want 0", allocs)
	}
}
