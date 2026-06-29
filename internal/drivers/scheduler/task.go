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
	"context"
	"sync"
)

type Task func(ctx context.Context) error

type taskSlot struct {
	task Task
}

var taskSlotPool = sync.Pool{
	New: func() interface{} {
		return &taskSlot{}
	},
}

func acquireTaskSlot() *taskSlot {
	return taskSlotPool.Get().(*taskSlot)
}

func releaseTaskSlot(s *taskSlot) {
	s.task = nil
	taskSlotPool.Put(s)
}
