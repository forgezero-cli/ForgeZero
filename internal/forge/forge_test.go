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
	"testing"
	"unsafe"
)

func TestDispatchBatchMixed(t *testing.T) {
	var hashData [513]byte
	for index := range hashData {
		hashData[index] = byte(index*31 + 7)
	}
	var moveSource, moveDestination [192]byte
	for index := range moveSource {
		moveSource[index] = byte(index*19 + 3)
	}
	var compareLeft, compareRight [65]byte
	for index := range compareLeft {
		compareLeft[index] = byte(index*11 + 5)
		compareRight[index] = compareLeft[index]
	}
	batch := [4]UTEC{
		{Kind: uint16(KindHashByteBuffer), SrcPtr: unsafe.Pointer(&hashData[0]), Len: uint64(len(hashData)), Aux1: 0x123456789abcdef0},
		{Kind: uint16(KindMemMove), SrcPtr: unsafe.Pointer(&moveSource[0]), DstPtr: unsafe.Pointer(&moveDestination[0]), Len: uint64(len(moveSource))},
		{Kind: uint16(KindVectorCompare), SrcPtr: unsafe.Pointer(&compareLeft[0]), DstPtr: unsafe.Pointer(&compareRight[0]), Len: uint64(len(compareLeft))},
		{Kind: 65535},
	}
	DispatchBatch(batch[:])
	if got, want := batch[0].RetVal, hashForgeScalar(hashData[:], batch[0].Aux1); got != want {
		t.Fatalf("hash result = %x, want %x", got, want)
	}
	if batch[1].RetVal != uint64(len(moveSource)) {
		t.Fatalf("move result = %d", batch[1].RetVal)
	}
	for index := range moveSource {
		if moveDestination[index] != moveSource[index] {
			t.Fatalf("move byte %d = %d, want %d", index, moveDestination[index], moveSource[index])
		}
	}
	if batch[2].RetVal != 1 {
		t.Fatalf("equal compare result = %d, want 1", batch[2].RetVal)
	}
	if batch[3].RetVal != UnsupportedResult {
		t.Fatalf("unsupported kind result = %d", batch[3].RetVal)
	}
	compareRight[len(compareRight)-1]++
	different := [1]UTEC{{Kind: uint16(KindVectorCompare), SrcPtr: unsafe.Pointer(&compareLeft[0]), DstPtr: unsafe.Pointer(&compareRight[0]), Len: uint64(len(compareLeft))}}
	DispatchBatch(different[:])
	if different[0].RetVal != 0 {
		t.Fatalf("different compare result = %d, want 0", different[0].RetVal)
	}
	var sqeSource, sqeDestination [64]byte
	for index := range sqeSource {
		sqeSource[index] = byte(index*7 + 1)
	}
	var sqArrayEntry uint32
	target := DriverIOTarget{SQE: unsafe.Pointer(&sqeDestination[0]), ArraySlot: unsafe.Pointer(&sqArrayEntry)}
	driver := [1]UTEC{{Kind: uint16(KindDriverIO), ID: 11, SrcPtr: unsafe.Pointer(&sqeSource[0]), DstPtr: unsafe.Pointer(&target), Len: 64}}
	DispatchBatch(driver[:])
	if driver[0].RetVal != 64 || sqArrayEntry != 11 || sqeSource != sqeDestination {
		t.Fatal("driver SQE fill mismatch")
	}
}

func TestDispatchBatchGeneric(t *testing.T) {
	data := []byte{1, 3, 5, 7, 9}
	left := []byte{4, 6, 8, 10, 12}
	right := []byte{4, 6, 8, 10, 12}
	batch := [3]UTEC{
		{Kind: uint16(KindHashByteBuffer), SrcPtr: unsafe.Pointer(unsafe.SliceData(data)), Len: uint64(len(data)), Aux1: 17},
		{Kind: uint16(KindVectorCompare), SrcPtr: unsafe.Pointer(unsafe.SliceData(left)), DstPtr: unsafe.Pointer(unsafe.SliceData(right)), Len: uint64(len(left))},
		{Kind: uint16(KindMemMove), SrcPtr: unsafe.Pointer(unsafe.SliceData(data)), DstPtr: unsafe.Pointer(unsafe.SliceData(left)), Len: uint64(len(data))},
	}
	dispatchBatchGeneric(batch[:])
	if batch[0].RetVal != hashForgeScalar(data, 17) || batch[1].RetVal != 1 || batch[2].RetVal != uint64(len(data)) {
		t.Fatalf("generic results = (%d, %d, %d)", batch[0].RetVal, batch[1].RetVal, batch[2].RetVal)
	}
}

func TestSyscallProxyWhitelist(t *testing.T) {
	var path [2]byte
	path[0] = '/'
	tasks := [2]UTEC{
		{Kind: uint16(KindSyscallProxy), Flags: FlagAbortBatchOnError, Aux1: 39},
		{Kind: uint16(KindVectorCompare)},
	}
	DispatchBatch(tasks[:])
	if tasks[0].RetVal != UnsupportedResult {
		t.Fatalf("unlisted syscall result = %d, want %d", tasks[0].RetVal, UnsupportedResult)
	}
	if tasks[1].RetVal != ^uint64(124) {
		t.Fatalf("task after failed syscall = %d, want ECANCELED", tasks[1].RetVal)
	}
	unsafeMount := [1]UTEC{{Kind: uint16(KindSyscallProxy), Aux1: 165, SrcPtr: unsafe.Pointer(&path[0]), Len: 0x5000}}
	DispatchBatch(unsafeMount[:])
	if unsafeMount[0].RetVal != UnsupportedResult {
		t.Fatalf("invalid mount request result = %d, want %d", unsafeMount[0].RetVal, UnsupportedResult)
	}
}

func TestDispatchMemMoveOverlapAndStreaming(t *testing.T) {
	var overlap [129]byte
	for index := range overlap {
		overlap[index] = byte(index)
	}
	want := overlap
	copy(want[1:], want[:128])
	task := [1]UTEC{{Kind: uint16(KindMemMove), SrcPtr: unsafe.Pointer(&overlap[0]), DstPtr: unsafe.Pointer(&overlap[1]), Len: 128}}
	DispatchBatch(task[:])
	if overlap != want || task[0].RetVal != 128 {
		for index := range overlap {
			if overlap[index] != want[index] {
				t.Fatalf("overlap byte %d = %d, want %d; copied=%d", index, overlap[index], want[index], task[0].RetVal)
			}
		}
		t.Fatalf("overlap copied=%d, want 128", task[0].RetVal)
	}

	const size = 8193
	source := make([]byte, size)
	destination := make([]byte, size)
	for index := range source {
		source[index] = byte(index*17 + 9)
	}
	stream := UTEC{Kind: uint16(KindMemMove), Flags: FlagNoCache, SrcPtr: unsafe.Pointer(unsafe.SliceData(source)), DstPtr: unsafe.Pointer(unsafe.SliceData(destination)), Len: size}
	DispatchBatch([]UTEC{stream})
	for index := range source {
		if destination[index] != source[index] {
			t.Fatalf("streaming move byte %d differs", index)
		}
	}
}

func TestDispatchBatchDoesNotAllocate(t *testing.T) {
	var left, right [64]byte
	batch := [2]UTEC{
		{Kind: uint16(KindVectorCompare), SrcPtr: unsafe.Pointer(&left[0]), DstPtr: unsafe.Pointer(&right[0]), Len: uint64(len(left))},
		{Kind: uint16(KindMemMove), SrcPtr: unsafe.Pointer(&left[0]), DstPtr: unsafe.Pointer(&right[0]), Len: uint64(len(left))},
	}
	allocs := testing.AllocsPerRun(100, func() {
		DispatchBatch(batch[:])
	})
	if allocs != 0 {
		t.Fatalf("DispatchBatch allocations = %v, want 0", allocs)
	}
}

func TestForgeBufferAPIsDoNotAllocate(t *testing.T) {
	var source, destination [128]byte
	for index := range source {
		source[index] = byte(index*13 + 1)
	}
	allocs := testing.AllocsPerRun(100, func() {
		_ = HashBytes(source[:], 19)
		_ = CompareBytes(source[:], source[:])
		if copied := MoveBytes(destination[:], source[:], 0); copied != len(source) {
			t.Fatalf("copied = %d, want %d", copied, len(source))
		}
	})
	if allocs != 0 {
		t.Fatalf("Forge buffer API allocations = %v, want 0", allocs)
	}
}

func BenchmarkForgeCoreHashBatch(b *testing.B) {
	data := make([]byte, 16*1024)
	for index := range data {
		data[index] = byte(index*23 + 1)
	}
	var batch [4]UTEC
	for index := range batch {
		batch[index] = UTEC{Kind: uint16(KindHashByteBuffer), SrcPtr: unsafe.Pointer(unsafe.SliceData(data)), Len: uint64(len(data)), Aux1: uint64(index + 1)}
	}
	b.SetBytes(int64(len(data) * len(batch)))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		DispatchBatch(batch[:])
	}
}

func BenchmarkForgeCoreHashBatchGo(b *testing.B) {
	data := make([]byte, 16*1024)
	for index := range data {
		data[index] = byte(index*23 + 1)
	}
	b.SetBytes(int64(len(data) * 4))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for seed := uint64(1); seed <= 4; seed++ {
			_ = hashForgeScalar(data, seed)
		}
	}
}

func BenchmarkForgeCoreMoveBatch(b *testing.B) {
	const size = 1 << 20
	sources := [4][]byte{make([]byte, size), make([]byte, size), make([]byte, size), make([]byte, size)}
	destinations := [4][]byte{make([]byte, size), make([]byte, size), make([]byte, size), make([]byte, size)}
	var batch [4]UTEC
	for index := range batch {
		batch[index] = UTEC{Kind: uint16(KindMemMove), SrcPtr: unsafe.Pointer(unsafe.SliceData(sources[index])), DstPtr: unsafe.Pointer(unsafe.SliceData(destinations[index])), Len: size}
	}
	b.SetBytes(int64(size * len(batch)))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		DispatchBatch(batch[:])
	}
}

func BenchmarkForgeCoreMoveStreamingBatch(b *testing.B) {
	const size = 1 << 20
	sources := [4][]byte{make([]byte, size), make([]byte, size), make([]byte, size), make([]byte, size)}
	destinations := [4][]byte{make([]byte, size), make([]byte, size), make([]byte, size), make([]byte, size)}
	var batch [4]UTEC
	for index := range batch {
		batch[index] = UTEC{Kind: uint16(KindMemMove), Flags: FlagNoCache, SrcPtr: unsafe.Pointer(unsafe.SliceData(sources[index])), DstPtr: unsafe.Pointer(unsafe.SliceData(destinations[index])), Len: size}
	}
	b.SetBytes(int64(size * len(batch)))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		DispatchBatch(batch[:])
	}
}

func BenchmarkForgeCoreMoveBatchGo(b *testing.B) {
	const size = 1 << 20
	sources := [4][]byte{make([]byte, size), make([]byte, size), make([]byte, size), make([]byte, size)}
	destinations := [4][]byte{make([]byte, size), make([]byte, size), make([]byte, size), make([]byte, size)}
	b.SetBytes(int64(size * len(sources)))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for index := range sources {
			copy(destinations[index], sources[index])
		}
	}
}
