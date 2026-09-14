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
	"encoding/binary"
	"errors"
	"os"
	"runtime"
	"sync"
	"unsafe"

	fzio "github.com/forgezero-cli/ForgeZero/internal/io_uring"
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
	parallelMu      sync.Mutex
	parallelCond    *sync.Cond
	parallelWG      sync.WaitGroup
	parallelJobs    []boomLeafJob
	parallelNext    int
	parallelDone    int
	parallelCount   int
	parallelBatch   uint64
	parallelClosed  bool
	parallelWorkers int
	initialized     bool
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

const boomBoomMmapThreshold = 16 << 10

func NewBoomBoomHasher(context BoomBoomContext) (*BoomBoomHasher, error) {
	key, err := BoomBoomContextDigest(context)
	if err != nil {
		return nil, err
	}
	return &BoomBoomHasher{key: key}, nil
}

func BoomBoomContextDigest(context BoomBoomContext) ([32]byte, error) {
	data := make([]byte, 0, 64+len(context.Compiler)+len(context.Version)+len(context.Target))
	data = append(data, boomBoomDomain[:]...)
	data = appendBoomField(data, context.Compiler)
	data = appendBoomField(data, context.Version)
	data = appendBoomField(data, context.Target)
	var count [8]byte
	binary.LittleEndian.PutUint64(count[:], uint64(len(context.Flags)))
	data = append(data, count[:]...)
	for _, flag := range context.Flags {
		data = appendBoomField(data, flag)
	}
	var key [32]byte
	hash := HashBB64(data, binary.LittleEndian.Uint64(boomBoomKey[:8]))
	for i := range 4 {
		hash ^= hash >> 29
		hash *= bb64Mul
		binary.LittleEndian.PutUint64(key[i*8:], hash)
	}
	return key, nil
}

func appendBoomField(data []byte, value string) []byte {
	var size [8]byte
	binary.LittleEndian.PutUint64(size[:], uint64(len(value)))
	data = append(data, size[:]...)
	return append(data, unsafeStringBytes(value)...)
}

func (h *BoomBoomHasher) Hash(data []byte) ([32]byte, error) {
	return h.Update(data)
}

func (h *BoomBoomHasher) Update(data []byte) ([32]byte, error) {
	if h == nil {
		return [32]byte{}, errors.New("nil BoomBoomHasher")
	}
	if h.initialized && len(h.previous) == len(data) && bytesEqual(h.previous, data) {
		h.changed = 0
		return h.root, nil
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
		if err := h.hashChangedLeavesParallel(data); err != nil {
			return [32]byte{}, err
		}
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
	h.initialized = true
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
	if h == nil || h.parallelCond == nil {
		return
	}
	h.parallelMu.Lock()
	if !h.parallelClosed {
		h.parallelClosed = true
		h.parallelCond.Broadcast()
	}
	h.parallelMu.Unlock()
	h.parallelWG.Wait()
}

func (h *BoomBoomHasher) hashLeaf(data []byte, kind uint8, index int) error {
	expandBB64Seed(data, kind, index, binary.LittleEndian.Uint64(h.key[:8]), &h.scan[index].Digest)
	return nil
}

func (h *BoomBoomHasher) hashChangedLeavesParallel(data []byte) error {
	if len(h.changedIndexes) == 0 {
		return nil
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
	h.parallelMu.Lock()
	if h.parallelClosed {
		h.parallelMu.Unlock()
		return errors.New("BoomBoomHasher is closed")
	}
	h.parallelNext = 0
	h.parallelDone = 0
	h.parallelCount = len(h.changedIndexes)
	h.parallelBatch++
	h.parallelCond.Broadcast()
	for h.parallelDone != h.parallelCount && !h.parallelClosed {
		h.parallelCond.Wait()
	}
	closed := h.parallelClosed
	h.parallelMu.Unlock()
	if closed {
		return errors.New("BoomBoomHasher is closed")
	}
	return nil
}

func (h *BoomBoomHasher) startParallelWorkers() {
	workers := runtime.GOMAXPROCS(0)
	if workers < 2 {
		workers = 2
	}
	if workers > 16 {
		workers = 16
	}
	h.parallelCond = sync.NewCond(&h.parallelMu)
	h.parallelWorkers = workers
	for i := 0; i < workers; i++ {
		h.parallelWG.Add(1)
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			defer h.parallelWG.Done()
			batch := uint64(0)
			for {
				h.parallelMu.Lock()
				for batch == h.parallelBatch && !h.parallelClosed {
					h.parallelCond.Wait()
				}
				if h.parallelClosed {
					h.parallelMu.Unlock()
					return
				}
				batch = h.parallelBatch
				h.parallelMu.Unlock()
				for {
					h.parallelMu.Lock()
					if h.parallelClosed || h.parallelNext >= h.parallelCount {
						h.parallelMu.Unlock()
						break
					}
					index := h.parallelNext
					h.parallelNext++
					job := h.parallelJobs[index]
					h.parallelMu.Unlock()
					candidate := &job.chunks[job.index]
					expandBB64Seed(job.data[candidate.Start:candidate.End], candidate.Kind, job.index, binary.LittleEndian.Uint64(h.key[:8]), &candidate.Digest)
					h.parallelMu.Lock()
					h.parallelDone++
					if h.parallelDone == h.parallelCount {
						h.parallelCond.Broadcast()
					}
					h.parallelMu.Unlock()
				}
			}
		}()
	}
}

func (h *BoomBoomHasher) hashTree(chunks []BoomBoomChunk) [32]byte {
	if len(chunks) == 0 {
		var empty [32]byte
		expandBB64Seed(boomBoomDomain[:], 0, 0, binary.LittleEndian.Uint64(h.key[:8]), &empty)
		return empty
	}
	base := 1
	for base < len(chunks) {
		base <<= 1
	}
	if h.treeBase != base || h.treeLeaves != len(chunks) {
		h.treeBase = base
		h.treeLeaves = len(chunks)
		treeSize := base * 2
		if cap(h.tree) < treeSize {
			capacity := cap(h.tree) * 2
			if capacity < treeSize {
				capacity = treeSize
			}
			h.tree = make([][32]byte, treeSize, capacity)
		} else {
			h.tree = h.tree[:treeSize]
		}
		for i := range chunks {
			h.tree[base+i] = chunks[i].Digest
		}
		for i := len(chunks); i < base; i++ {
			h.tree[base+i] = chunks[len(chunks)-1].Digest
		}
		for i := base - 1; i > 0; i-- {
			h.tree[i] = h.hashPair(h.tree[i<<1], h.tree[i<<1|1], uint64(i))
		}
		root := h.tree[1]
		binary.LittleEndian.PutUint64(root[:8], binary.LittleEndian.Uint64(root[:8])^boomTreeFingerprint(chunks))
		return root
	}
	if len(chunks) < base && h.tree[base+len(chunks)] != chunks[len(chunks)-1].Digest {
		h.tree[base+len(chunks)] = chunks[len(chunks)-1].Digest
		position := (base + len(chunks)) >> 1
		for ; position > 0; position >>= 1 {
			h.tree[position] = h.hashPair(h.tree[position<<1], h.tree[position<<1|1], uint64(position))
		}
	}
	for i := range chunks {
		position := base + i
		if h.tree[position] == chunks[i].Digest {
			continue
		}
		h.tree[position] = chunks[i].Digest
		for position >>= 1; position > 0; position >>= 1 {
			h.tree[position] = h.hashPair(h.tree[position<<1], h.tree[position<<1|1], uint64(position))
		}
	}
	root := h.tree[1]
	binary.LittleEndian.PutUint64(root[:8], binary.LittleEndian.Uint64(root[:8])^boomTreeFingerprint(chunks))
	return root
}

func boomTreeFingerprint(chunks []BoomBoomChunk) uint64 {
	var fingerprint uint64
	for index := range chunks {
		fingerprint ^= HashBB64(chunks[index].Digest[:], uint64(index)*0x9e3779b97f4a7c15+0xd6e8feb86659fd93)
	}
	fingerprint ^= fingerprint >> 33
	fingerprint *= bb64Mul
	return fingerprint ^ fingerprint>>29
}

func (h *BoomBoomHasher) hashPair(left, right [32]byte, position uint64) [32]byte {
	return hashBB64PairSeed(left, right, 0x4242363400000001^position*0x9e3779b97f4a7c15)
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
	defer h.Close()
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
		defer h.Close()
		return h.Hash(nil)
	}
	if info.Size() < boomBoomMmapThreshold {
		data, err := os.ReadFile(path)
		if err != nil {
			return [32]byte{}, err
		}
		h, err := NewBoomBoomHasher(context)
		if err != nil {
			return [32]byte{}, err
		}
		defer h.Close()
		return h.Hash(data)
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
				defer h.Close()
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
	defer h.Close()
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
			end := len(data) - i
			for j := i; j < len(data); j++ {
				if data[j] == '\n' {
					end = j - i + 1
					break
				}
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

func isBoomSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func unsafeStringBytes(value string) []byte {
	return unsafe.Slice(unsafe.StringData(value), len(value))
}
