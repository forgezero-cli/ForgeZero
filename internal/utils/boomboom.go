/*
 *   BOOMBOOM HASH by ForgeZero CLI / AlexVoste
 *
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
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	fzio "github.com/forgezero-cli/ForgeZero/internal/io_uring"
	"github.com/zeebo/blake3"
)

type BoomBoomContext struct {
	Compiler string
	Version  string
	Target   string
	Flags    []string
}

type BoomBoomChunk struct {
	Start  int
	End    int
	Kind   uint8
	Digest [32]byte
}

type BoomBoomHasher struct {
	key             [32]byte
	hasher          *blake3.Hasher
	chunks          []BoomBoomChunk
	previous        []byte
	tree            [][32]byte
	treeBase        int
	treeLeaves      int
	scan            []BoomBoomChunk
	root            [32]byte
	changed         int
	changedIndexes  []int
	chunkHash       [32]byte
	parallelOnce    sync.Once
	parallelJobs    []boomLeafJob
	parallelNext    atomic.Int64
	parallelDone    atomic.Int64
	parallelCount   atomic.Int64
	parallelBatch   atomic.Int64
	parallelStop    atomic.Bool
	parallelWorkers int
}

type boomLeafJob struct {
	data   []byte
	chunks []BoomBoomChunk
	index  int
}

var (
	boomBoomDomain = [16]byte{'B', 'o', 'o', 'm', 'B', 'o', 'o', 'm', 1}
	boomBoomKey    = [32]byte{0x9d, 0x74, 0x31, 0x6f, 0xd5, 0x23, 0x1b, 0xe4, 0xa1, 0x8f, 0x03, 0x71, 0x42, 0x5d, 0x6b, 0x9a, 0x3c, 0xf4, 0x75, 0x28, 0x0d, 0x62, 0x8a, 0x19, 0xbf, 0x4e, 0x50, 0x33, 0x13, 0x21, 0x97, 0x6c}
)

func NewBoomBoomHasher(context BoomBoomContext) (*BoomBoomHasher, error) {
	key, err := BoomBoomContextDigest(context)
	if err != nil {
		return nil, err
	}
	h, err := blake3.NewKeyed(key[:])
	if err != nil {
		return nil, err
	}
	return &BoomBoomHasher{key: key, hasher: h}, nil
}

func BoomBoomContextDigest(context BoomBoomContext) ([32]byte, error) {
	base, err := blake3.NewKeyed(boomBoomKey[:])
	if err != nil {
		return [32]byte{}, err
	}
	_, _ = base.Write(boomBoomDomain[:])
	writeBoomField(base, context.Compiler)
	writeBoomField(base, context.Version)
	writeBoomField(base, context.Target)
	var count [8]byte
	binary.LittleEndian.PutUint64(count[:], uint64(len(context.Flags)))
	_, _ = base.Write(count[:])
	for _, flag := range context.Flags {
		writeBoomField(base, flag)
	}
	digest := base.Digest()
	var key [32]byte
	_, err = digest.Read(key[:])
	if err != nil {
		return [32]byte{}, err
	}
	return key, nil
}

func writeBoomField(h *blake3.Hasher, value string) {
	var size [8]byte
	binary.LittleEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = h.Write(size[:])
	_, _ = h.Write(unsafeStringBytes(value))
}

func (h *BoomBoomHasher) Hash(data []byte) ([32]byte, error) {
	return h.Update(data)
}

func (h *BoomBoomHasher) Update(data []byte) ([32]byte, error) {
	if h == nil || h.hasher == nil {
		return [32]byte{}, errors.New("nil BoomBoomHasher")
	}
	h.scan = scanBoomBoom(data, h.scan[:0])
	h.changed = 0
	h.changedIndexes = h.changedIndexes[:0]
	if len(h.chunks) != len(h.scan) {
		h.chunks = growBoomChunks(h.chunks, len(h.scan))
	}
	for i := range h.scan {
		candidate := &h.scan[i]
		cached := &h.chunks[i]
		if cached.Start >= 0 && cached.End <= len(h.previous) && cached.Start <= cached.End && cached.Start == candidate.Start && cached.End-cached.Start == candidate.End-candidate.Start && cached.Kind == candidate.Kind && bytesEqual(h.previous[cached.Start:cached.End], data[candidate.Start:candidate.End]) {
			candidate.Digest = cached.Digest
			continue
		}
		h.changedIndexes = append(h.changedIndexes, i)
	}
	h.changed = len(h.changedIndexes)
	if h.changed > 4 {
		h.hashChangedLeavesParallel(data)
	} else {
		for _, index := range h.changedIndexes {
			candidate := &h.scan[index]
			if err := h.hashLeaf(data[candidate.Start:candidate.End], candidate.Kind, index); err != nil {
				return [32]byte{}, err
			}
		}
	}
	copy(h.chunks, h.scan)
	if cap(h.previous) < len(data) {
		h.previous = make([]byte, len(data))
	} else {
		h.previous = h.previous[:len(data)]
	}
	copy(h.previous, data)
	h.root = h.hashTree(h.chunks)
	return h.root, nil
}

func (h *BoomBoomHasher) ChangedChunks() int {
	if h == nil {
		return 0
	}
	return h.changed
}

func (h *BoomBoomHasher) Chunks() []BoomBoomChunk {
	if h == nil {
		return nil
	}
	return h.chunks
}

func (h *BoomBoomHasher) Close() {
	if h == nil || len(h.parallelJobs) == 0 {
		return
	}
	h.parallelStop.Store(true)
}

func (h *BoomBoomHasher) hashLeaf(data []byte, kind uint8, index int) error {
	return hashBoomLeaf(h.hasher, data, kind, index, &h.scan[index].Digest)
}

func hashBoomLeaf(hasher *blake3.Hasher, data []byte, kind uint8, index int, output *[32]byte) error {
	hasher.Reset()
	var header [32]byte
	copy(header[:16], boomBoomDomain[:])
	header[16] = 1
	header[17] = kind
	binary.LittleEndian.PutUint64(header[24:], uint64(index))
	_, _ = hasher.Write(header[:])
	_, _ = hasher.Write(data)
	digest := hasher.Digest()
	_, err := digest.Read(output[:])
	return err
}

func (h *BoomBoomHasher) hashChangedLeavesParallel(data []byte) {
	if len(h.changedIndexes) == 0 {
		return
	}
	h.parallelOnce.Do(h.startParallelWorkers)
	if cap(h.parallelJobs) < len(h.changedIndexes) {
		h.parallelJobs = make([]boomLeafJob, len(h.changedIndexes))
	} else {
		h.parallelJobs = h.parallelJobs[:len(h.changedIndexes)]
	}
	for i, index := range h.changedIndexes {
		h.parallelJobs[i] = boomLeafJob{data: data, chunks: h.scan, index: index}
	}
	h.parallelNext.Store(0)
	h.parallelDone.Store(0)
	h.parallelCount.Store(int64(len(h.changedIndexes)))
	h.parallelBatch.Add(1)
	for h.parallelDone.Load() != int64(len(h.changedIndexes)) {
		runtime.Gosched()
	}
}

func (h *BoomBoomHasher) startParallelWorkers() {
	workers := runtime.GOMAXPROCS(0)
	if workers < 2 {
		workers = 2
	}
	if workers > 16 {
		workers = 16
	}
	h.parallelJobs = make([]boomLeafJob, 0, len(h.scan))
	h.parallelWorkers = workers
	for i := 0; i < workers; i++ {
		go func() {
			leafHasher, _ := blake3.NewKeyed(h.key[:])
			batch := int64(0)
			for !h.parallelStop.Load() {
				current := h.parallelBatch.Load()
				if current == batch {
					runtime.Gosched()
					continue
				}
				batch = current
				for {
					index := h.parallelNext.Add(1) - 1
					if index >= h.parallelCount.Load() {
						break
					}
					job := &h.parallelJobs[index]
					candidate := &job.chunks[job.index]
					_ = hashBoomLeaf(leafHasher, job.data[candidate.Start:candidate.End], candidate.Kind, job.index, &candidate.Digest)
					h.parallelDone.Add(1)
				}
			}
		}()
	}
}

func (h *BoomBoomHasher) hashTree(chunks []BoomBoomChunk) [32]byte {
	if len(chunks) == 0 {
		h.hasher.Reset()
		_, _ = h.hasher.Write(boomBoomDomain[:])
		digest := h.hasher.Digest()
		var empty [32]byte
		_, _ = digest.Read(empty[:])
		return empty
	}
	base := 1
	for base < len(chunks) {
		base <<= 1
	}
	if h.treeBase != base || h.treeLeaves != len(chunks) {
		h.treeBase = base
		h.treeLeaves = len(chunks)
		h.tree = make([][32]byte, base*2)
		for i := range chunks {
			h.tree[base+i] = chunks[i].Digest
		}
		for i := len(chunks); i < base; i++ {
			h.tree[base+i] = chunks[len(chunks)-1].Digest
		}
		for i := base - 1; i > 0; i-- {
			h.tree[i] = h.hashPair(h.tree[i<<1], h.tree[i<<1|1])
		}
		return h.tree[1]
	}
	if len(chunks) < base && h.tree[base+len(chunks)] != chunks[len(chunks)-1].Digest {
		h.tree[base+len(chunks)] = chunks[len(chunks)-1].Digest
		position := (base + len(chunks)) >> 1
		for ; position > 0; position >>= 1 {
			h.tree[position] = h.hashPair(h.tree[position<<1], h.tree[position<<1|1])
		}
	}
	for i := range chunks {
		position := base + i
		if h.tree[position] == chunks[i].Digest {
			continue
		}
		h.tree[position] = chunks[i].Digest
		for position >>= 1; position > 0; position >>= 1 {
			h.tree[position] = h.hashPair(h.tree[position<<1], h.tree[position<<1|1])
		}
	}
	return h.tree[1]
}

func (h *BoomBoomHasher) hashPair(left, right [32]byte) [32]byte {
	h.hasher.Reset()
	var header [32]byte
	copy(header[:16], boomBoomDomain[:])
	header[16] = 2
	_, _ = h.hasher.Write(header[:])
	_, _ = h.hasher.Write(left[:])
	_, _ = h.hasher.Write(right[:])
	digest := h.hasher.Digest()
	var out [32]byte
	_, _ = digest.Read(out[:])
	return out
}

func HashBoomBoomFile(path string, context BoomBoomContext) ([32]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return [32]byte{}, err
	}
	h, err := NewBoomBoomHasher(context)
	if err != nil {
		return [32]byte{}, err
	}
	return h.Hash(data)
}

func HashBoomBoomMappedFile(path string, context BoomBoomContext) ([32]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return [32]byte{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return [32]byte{}, err
	}
	if info.Size() == 0 {
		h, err := NewBoomBoomHasher(context)
		if err != nil {
			return [32]byte{}, err
		}
		return h.Hash(nil)
	}
	data, err := mmapFile(getFileDescriptor(file), info.Size())
	if err != nil {
		if fzio.Enabled() {
			data, readErr := fzio.ReadFile(path)
			if readErr == nil {
				h, hashErr := NewBoomBoomHasher(context)
				if hashErr != nil {
					return [32]byte{}, hashErr
				}
				return h.Hash(data)
			}
		}
		return HashBoomBoomFile(path, context)
	}
	defer unmapFile(data)
	madviseNormal(data)
	h, err := NewBoomBoomHasher(context)
	if err != nil {
		return [32]byte{}, err
	}
	return h.Hash(data)
}

func scanBoomBoom(data []byte, chunks []BoomBoomChunk) []BoomBoomChunk {
	start := 0
	depth := 0
	lineStart := true
	inString := byte(0)
	inLineComment := false
	inBlockComment := false
	for i := 0; i < len(data); i++ {
		if offset := bytes.IndexAny(data[i:], "{};\n\"'#/"); offset > 0 {
			if lineStart && len(bytes.TrimSpace(data[i:i+offset])) > 0 {
				lineStart = false
			}
			i += offset
		}
		current := data[i]
		if inLineComment {
			if current == '\n' {
				inLineComment = false
				lineStart = true
			}
			continue
		}
		if inBlockComment {
			if current == '*' && i+1 < len(data) && data[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if inString != 0 {
			if current == '\\' {
				i++
				continue
			}
			if current == inString {
				inString = 0
			}
			continue
		}
		if current == '/' && i+1 < len(data) && data[i+1] == '/' {
			inLineComment = true
			i++
			continue
		}
		if current == '/' && i+1 < len(data) && data[i+1] == '*' {
			inBlockComment = true
			i++
			continue
		}
		if current == '"' || current == '\'' {
			inString = current
			lineStart = false
			continue
		}
		if lineStart && (current == '#' || current == '.') {
			end := bytesIndexByte(data[i:], '\n')
			if end < 0 {
				end = len(data) - i
			} else {
				end++
			}
			if i > start {
				chunks = append(chunks, BoomBoomChunk{Start: start, End: i, Kind: boomKindCode})
			}
			chunks = append(chunks, BoomBoomChunk{Start: i, End: i + end, Kind: boomKindDirective})
			i += end - 1
			start = i + 1
			lineStart = true
			continue
		}
		if current != ' ' && current != '\t' && current != '\r' && current != '\n' {
			lineStart = false
		}
		switch current {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
			if depth == 0 {
				chunks = append(chunks, BoomBoomChunk{Start: start, End: i + 1, Kind: boomKindBlock})
				start = i + 1
			}
		case ';':
			if depth == 0 {
				chunks = append(chunks, BoomBoomChunk{Start: start, End: i + 1, Kind: boomKindStatement})
				start = i + 1
			}
		case '\n':
			lineStart = true
		}
	}
	if start < len(data) {
		chunks = append(chunks, BoomBoomChunk{Start: start, End: len(data), Kind: boomKindCode})
	}
	for i := range chunks {
		chunks[i].Start = trimBoomStart(data, chunks[i].Start, chunks[i].End)
		chunks[i].End = trimBoomEnd(data, chunks[i].Start, chunks[i].End)
	}
	return compactBoomChunks(data, chunks)
}

const (
	boomKindCode uint8 = iota
	boomKindDirective
	boomKindBlock
	boomKindStatement
)

func trimBoomStart(data []byte, start, end int) int {
	for start < end && isBoomSpace(data[start]) {
		start++
	}
	return start
}

func trimBoomEnd(data []byte, start, end int) int {
	for end > start && isBoomSpace(data[end-1]) {
		end--
	}
	return end
}

func compactBoomChunks(data []byte, chunks []BoomBoomChunk) []BoomBoomChunk {
	out := chunks[:0]
	for _, chunk := range chunks {
		if chunk.Start < chunk.End {
			out = append(out, chunk)
		}
	}
	return out
}

func growBoomChunks(chunks []BoomBoomChunk, size int) []BoomBoomChunk {
	if cap(chunks) >= size {
		return chunks[:size]
	}
	capacity := cap(chunks) * 2
	if capacity < size {
		capacity = size
	}
	return make([]BoomBoomChunk, size, capacity)
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func bytesIndexByte(data []byte, value byte) int {
	for i, current := range data {
		if current == value {
			return i
		}
	}
	return -1
}

func isBoomSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func unsafeStringBytes(value string) []byte {
	return unsafe.Slice(unsafe.StringData(value), len(value))
}
