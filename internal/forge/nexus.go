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

package forge

import (
	"unsafe"
)

const UnsupportedResult = ^uint64(0)

func DispatchBatch(batch []UTEC) {
	if len(batch) == 0 {
		return
	}
	dispatchBatchPlatform(batch)
}

func dispatchBatchGeneric(batch []UTEC) {
	abort := false
	for index := range batch {
		task := &batch[index]
		if abort {
			task.RetVal = ^uint64(124)
			continue
		}
		switch task.Kind {
		case uint16(KindHashByteBuffer):
			if task.Len > uint64(^uint(0)>>1) || task.SrcPtr == nil && task.Len != 0 {
				task.RetVal = UnsupportedResult
				continue
			}
			task.RetVal = hashForgeScalar(unsafe.Slice((*byte)(task.SrcPtr), int(task.Len)), task.Aux1)
		case uint16(KindMemMove):
			if task.Len > uint64(^uint(0)>>1) || (task.SrcPtr == nil || task.DstPtr == nil) && task.Len != 0 {
				task.RetVal = UnsupportedResult
				continue
			}
			task.RetVal = uint64(copy(unsafe.Slice((*byte)(task.DstPtr), int(task.Len)), unsafe.Slice((*byte)(task.SrcPtr), int(task.Len))))
		case uint16(KindVectorCompare):
			if task.Len > uint64(^uint(0)>>1) || (task.SrcPtr == nil || task.DstPtr == nil) && task.Len != 0 {
				task.RetVal = UnsupportedResult
				continue
			}
			left := unsafe.Slice((*byte)(task.SrcPtr), int(task.Len))
			right := unsafe.Slice((*byte)(task.DstPtr), int(task.Len))
			equal := true
			for offset := range left {
				if left[offset] != right[offset] {
					equal = false
					break
				}
			}
			if equal {
				task.RetVal = 1
			} else {
				task.RetVal = 0
			}
		case uint16(KindDriverIO):
			if task.Len != 64 || task.SrcPtr == nil || task.DstPtr == nil {
				task.RetVal = UnsupportedResult
				continue
			}
			target := (*DriverIOTarget)(task.DstPtr)
			if target.SQE == nil || target.ArraySlot == nil {
				task.RetVal = UnsupportedResult
				continue
			}
			copy(unsafe.Slice((*byte)(target.SQE), 64), unsafe.Slice((*byte)(task.SrcPtr), 64))
			*(*uint32)(target.ArraySlot) = task.ID
			task.RetVal = 64
		case uint16(KindSyscallProxy):
			task.RetVal = UnsupportedResult
			if task.Flags&FlagAbortBatchOnError != 0 {
				abort = true
			}
		default:
			task.RetVal = UnsupportedResult
		}
	}
}

func HashBytes(data []byte, seed uint64) uint64 {
	tasks := [1]UTEC{{Kind: uint16(KindHashByteBuffer), SrcPtr: unsafe.Pointer(unsafe.SliceData(data)), Len: uint64(len(data)), Aux1: seed}}
	DispatchBatch(tasks[:])
	return tasks[0].RetVal
}

func CompareBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	tasks := [1]UTEC{{Kind: uint16(KindVectorCompare), SrcPtr: unsafe.Pointer(unsafe.SliceData(left)), DstPtr: unsafe.Pointer(unsafe.SliceData(right)), Len: uint64(len(left))}}
	DispatchBatch(tasks[:])
	return tasks[0].RetVal == 1
}

func MoveBytes(dst, src []byte, flags uint16) int {
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	tasks := [1]UTEC{{Kind: uint16(KindMemMove), Flags: flags, SrcPtr: unsafe.Pointer(unsafe.SliceData(src)), DstPtr: unsafe.Pointer(unsafe.SliceData(dst)), Len: uint64(n)}}
	DispatchBatch(tasks[:])
	if tasks[0].RetVal == UnsupportedResult {
		return 0
	}
	return int(tasks[0].RetVal)
}
