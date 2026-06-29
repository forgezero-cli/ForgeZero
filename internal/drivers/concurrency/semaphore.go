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

package concurrency

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
)

var errCancelled = errors.New("semaphore acquire cancelled")

type Semaphore struct {
	available int64
}

func NewSemaphore(initial int) *Semaphore {
	return &Semaphore{available: int64(initial)}
}

func (s *Semaphore) Acquire(n int) error {
	if n <= 0 {
		return nil
	}
	need := int64(n)
	for {
		cur := atomic.LoadInt64(&s.available)
		if cur >= need {
			if atomic.CompareAndSwapInt64(&s.available, cur, cur-need) {
				return nil
			}
			continue
		}
		runtime.Gosched()
	}
}

func (s *Semaphore) AcquireContext(ctx context.Context, n int) error {
	if n <= 0 {
		return nil
	}
	if ctx == nil {
		return s.Acquire(n)
	}
	need := int64(n)
	for {
		select {
		case <-ctx.Done():
			return errCancelled
		default:
		}
		cur := atomic.LoadInt64(&s.available)
		if cur >= need {
			if atomic.CompareAndSwapInt64(&s.available, cur, cur-need) {
				return nil
			}
			continue
		}
		runtime.Gosched()
	}
}

func (s *Semaphore) Release(n int) {
	if n > 0 {
		atomic.AddInt64(&s.available, int64(n))
	}
}
