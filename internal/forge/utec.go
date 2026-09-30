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

import "unsafe"

type TaskKind uint16

const (
	KindHashByteBuffer TaskKind = iota + 1
	KindMemMove
	KindVectorCompare
	KindDriverIO
	KindSyscallProxy
)

const (
	FlagNoCache uint16 = 1 << iota
	FlagBlockingIO
	FlagAbortBatchOnError
)

//go:notinheap
type UTEC struct {
	Kind   uint16
	Flags  uint16
	ID     uint32
	SrcPtr unsafe.Pointer
	DstPtr unsafe.Pointer
	Len    uint64
	Aux1   uint64
	Aux2   uint64
	RetVal uint64
	Aux3   uint64
}

type DriverIOTarget struct {
	SQE       unsafe.Pointer
	ArraySlot unsafe.Pointer
}

var (
	_ [64 - unsafe.Sizeof(UTEC{})]byte
	_ [unsafe.Sizeof(UTEC{}) - 64]byte
)
