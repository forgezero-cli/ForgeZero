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

import "sync"

type TaskFn func(arg uintptr, extra uintptr) error

type Task struct {
	Fn    TaskFn
	Arg   uintptr
	Extra uintptr
	pool  *Task
}

var taskPool = sync.Pool{New: func() any { return new(Task) }}

func AcquireTask(fn TaskFn, arg uintptr, extra uintptr) Task {
	t := taskPool.Get().(*Task)
	t.Fn = fn
	t.Arg = arg
	t.Extra = extra
	t.pool = t
	return *t
}

func ReleaseTask(task Task) {
	if task.pool == nil {
		return
	}
	t := task.pool
	t.Fn = nil
	t.Arg = 0
	t.Extra = 0
	t.pool = nil
	taskPool.Put(t)
}
