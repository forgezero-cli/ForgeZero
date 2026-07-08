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

package builder

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/forgezero-cli/ForgeZero/internal/hashpool"
	"github.com/forgezero-cli/ForgeZero/internal/io_uring"
	"github.com/forgezero-cli/ForgeZero/internal/logger"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func l3DataDir(cacheDir string) string {
	return filepath.Join(cacheDir, "actions")
}

func l2DataPath(cacheDir string) string {
	return filepath.Join(cacheDir, "actions.l2")
}

func archivePath(hashHex string, cacheDir string) string {
	return filepath.Join(l3DataDir(cacheDir), hashHex+".dat")
}

const (
	l1EntryCount       = 1 << 16
	l1EntryMask        = l1EntryCount - 1
	l2HeaderMagic      = 0xA1C3CAC0
	l2HeaderVersion    = 1
	l2HeaderSize       = 64
	l2OutputHeaderSize = 16
	l1OffsetBias       = 1
)

type l1Entry struct {
	key    uint64
	hash   [32]byte
	size   uint32
	offset uint64
	flags  uint32
	_      [8]byte
}

type actionCacheJob struct {
	cacheDir string
	inputs   []string
	action   string
	outputs  []string
	env      []string
}

var (
	l2Data       []byte
	l2File       *os.File
	l2Mutex      sync.RWMutex
	jobQueue     chan actionCacheJob
	initOnce     sync.Once
	preloadStart sync.Map
	preloadWait  sync.WaitGroup
)

func actionCacheInit() {
	initOnce.Do(func() {
		jobQueue = make(chan actionCacheJob, 1024)
		workers := runtime.GOMAXPROCS(0)
		if workers <= 0 {
			workers = 1
		}
		for i := 0; i < workers; i++ {
			go actionCacheWorker()
		}
	})
}

func actionCacheWorker() {
	for job := range jobQueue {
		_ = actionCacheStoreSync(job.inputs, job.action, job.outputs, job.env, job.cacheDir)
	}
}

func actionCacheRestore(ctx context.Context, inputs []string, action string, outputs []string, cacheDir string) (bool, error) {
	if len(outputs) == 0 {
		return false, nil
	}
	env := cacheEnv(ctx)
	digest, err := actionCacheKey(inputs, action, env)
	if err != nil {
		return false, err
	}
	idx := l1Key(digest)
	if entry, ok := l1Load(idx); ok {
		if entry.hash == digest {
			if entry.offset != 0 {
				if err := restoreFromL2(cacheDir, entry.offset-l1OffsetBias); err == nil {
					logger.Debug("action cache restored from L2\n")
					return true, nil
				}
			}
			if err := restoreFromL3(cacheDir, digest); err == nil {
				logger.Debug("action cache restored from L3\n")
				return true, nil
			}
		}
	}
	go PreloadCache(context.Background(), cacheDir)
	return false, nil
}

func actionCacheStore(ctx context.Context, inputs []string, action string, outputs []string, cacheDir string) error {
	if len(outputs) == 0 {
		return nil
	}
	actionCacheInit()
	jobQueue <- actionCacheJob{cacheDir: cacheDir, inputs: inputs, action: action, outputs: outputs, env: cacheEnv(ctx)}
	return nil
}

func cacheEnv(ctx context.Context) []string {
	if cfg := utils.ConfigFromContext(ctx); cfg != nil {
		return utils.SafeEnv(cfg)
	}
	env := os.Environ()
	sort.Strings(env)
	return env
}

func l1Index(key uint64, probe int) int {
	return int((key + uint64(probe)) & l1EntryMask)
}

var l1Entries [l1EntryCount]l1Entry

func l1Key(digest [32]byte) uint64 {
	return binary.LittleEndian.Uint64(digest[0:8])
}

func l1Load(key uint64) (*l1Entry, bool) {
	expected := key
	for probe := 0; probe < 16; probe++ {
		idx := l1Index(key, probe)
		entry := &l1Entries[idx]
		prefetch(unsafe.Pointer(entry))
		if probe+1 < 16 {
			next := l1Index(key, probe+1)
			prefetch(unsafe.Pointer(&l1Entries[next]))
		}
		if atomic.LoadUint64(&entry.key) != expected {
			if atomic.LoadUint64(&entry.key) == 0 {
				return nil, false
			}
			continue
		}
		if atomic.LoadUint32(&entry.flags)&1 == 0 {
			return nil, false
		}
		return entry, true
	}
	return nil, false
}

func l1Store(key uint64, hash [32]byte, size uint32, offset uint64) {
	if offset != 0 {
		offset += l1OffsetBias
	}
	idx := l1Index(key, 0)
	for probe := 0; probe < 16; probe++ {
		entry := &l1Entries[idx]
		current := atomic.LoadUint64(&entry.key)
		if current == 0 {
			if atomic.CompareAndSwapUint64(&entry.key, 0, key) {
				atomic.StoreUint32(&entry.flags, 0)
				entry.hash = hash
				entry.size = size
				entry.offset = offset
				atomic.StoreUint32(&entry.flags, 1)
				return
			}
		}
		if current == key {
			atomic.StoreUint32(&entry.flags, 0)
			entry.hash = hash
			entry.size = size
			entry.offset = offset
			atomic.StoreUint32(&entry.flags, 1)
			return
		}
		if probe == 15 {
			atomic.StoreUint32(&entry.flags, 0)
			atomic.StoreUint64(&entry.key, key)
			entry.hash = hash
			entry.size = size
			entry.offset = offset
			atomic.StoreUint32(&entry.flags, 1)
			return
		}
		idx = (idx + 1) & l1EntryMask
	}
}

func readFileMaybeIOUring(path string) ([]byte, error) {
	if io_uring.Enabled() {
		data, err := io_uring.ReadFile(path)
		if err == nil {
			return data, nil
		}
		logger.Debug("io_uring read failed: " + err.Error() + "\n")
	}
	return os.ReadFile(path)
}

func writeFileMaybeIOUring(path string, data []byte, perm os.FileMode) error {
	if io_uring.Enabled() {
		if err := io_uring.WriteFile(path, data, perm); err == nil {
			return nil
		} else {
			logger.Debug("io_uring write failed: " + err.Error() + "\n")
		}
	}
	return os.WriteFile(path, data, perm)
}

func actionCacheKey(inputs []string, action string, env []string) ([32]byte, error) {
	sorted := make([]string, len(inputs))
	copy(sorted, inputs)
	sort.Strings(sorted)
	hasher := hashpool.GetHasher()
	defer hashpool.PutHasher(hasher)
	for _, in := range sorted {
		digest, err := utils.HashFileDigest(in)
		if err != nil {
			return [32]byte{}, err
		}
		if _, err := hasher.Write(digest[:]); err != nil {
			return [32]byte{}, err
		}
		if _, err := hasher.Write([]byte{0}); err != nil {
			return [32]byte{}, err
		}
	}
	if _, err := hasher.Write([]byte(action)); err != nil {
		return [32]byte{}, err
	}
	if _, err := hasher.Write([]byte{0}); err != nil {
		return [32]byte{}, err
	}
	sort.Strings(env)
	for _, v := range env {
		if _, err := hasher.Write([]byte(v)); err != nil {
			return [32]byte{}, err
		}
		if _, err := hasher.Write([]byte{0}); err != nil {
			return [32]byte{}, err
		}
	}
	var out [32]byte
	digest := hasher.Digest()
	_, err := io.ReadFull(digest, out[:])
	if err != nil {
		return [32]byte{}, err
	}
	return out, nil
}

func preloadActionCache(cacheDir string) {
	dir := l3DataDir(cacheDir)
	list, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range list {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".dat") {
			continue
		}
		hashText := strings.TrimSuffix(name, ".dat")
		decoded, err := hex.DecodeString(hashText)
		if err != nil || len(decoded) != 32 {
			continue
		}
		var digest [32]byte
		copy(digest[:], decoded)
		l1Store(l1Key(digest), digest, 0, 0)
	}
}

func restoreFromL2(cacheDir string, offset uint64) error {
	if err := mapL2Data(cacheDir); err != nil {
		return err
	}
	l2Mutex.RLock()
	data := l2Data
	l2Mutex.RUnlock()
	if len(data) < int(offset)+l2HeaderSize {
		return os.ErrNotExist
	}
	record := data[offset:]
	if binary.LittleEndian.Uint32(record[0:4]) != l2HeaderMagic {
		return os.ErrNotExist
	}
	if binary.LittleEndian.Uint32(record[4:8]) != l2HeaderVersion {
		return os.ErrNotExist
	}
	recordSize := binary.LittleEndian.Uint32(record[46:50])
	if len(record) < int(l2HeaderSize+recordSize) {
		return os.ErrNotExist
	}
	return restoreOutputsFromBytes(record[:l2HeaderSize+recordSize])
}

func restoreFromL3(cacheDir string, digest [32]byte) error {
	archive := archivePath(hex.EncodeToString(digest[:]), cacheDir)
	data, err := readFileMaybeIOUring(archive)
	if err != nil {
		return err
	}
	if err := restoreOutputsFromBytes(data); err != nil {
		return err
	}
	offset, err := appendL2FromArchive(data, cacheDir)
	if err == nil {
		l1Store(l1Key(digest), digest, uint32(len(data)), offset)
	}
	return nil
}

func restoreOutputsFromBytes(data []byte) error {
	if len(data) < l2HeaderSize {
		return os.ErrInvalid
	}
	count := binary.LittleEndian.Uint16(data[40:42])
	pos := l2HeaderSize
	for i := 0; i < int(count); i++ {
		if len(data) < pos+l2OutputHeaderSize {
			return os.ErrInvalid
		}
		nameLen := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
		dataLen := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		pos += l2OutputHeaderSize
		if len(data) < pos+nameLen {
			return os.ErrInvalid
		}
		name := string(data[pos : pos+nameLen])
		pos += nameLen
		if len(data) < pos+dataLen {
			return os.ErrInvalid
		}
		if err := utils.EnsureDir(name); err != nil {
			return err
		}
		if err := writeFileMaybeIOUring(name, data[pos:pos+dataLen], 0o644); err != nil {
			return err
		}
		pos += dataLen
	}
	return nil
}

func writeActionArchive(path string, outputs []string, digest [32]byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "action_*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	count := len(outputs)
	recordSize := 0
	for _, out := range outputs {
		info, err := os.Stat(out)
		if err != nil {
			tmp.Close()
			return err
		}
		recordSize += l2OutputHeaderSize + len(out) + int(info.Size())
	}
	var header [l2HeaderSize]byte
	binary.LittleEndian.PutUint32(header[0:4], l2HeaderMagic)
	binary.LittleEndian.PutUint32(header[4:8], l2HeaderVersion)
	copy(header[8:40], digest[:])
	binary.LittleEndian.PutUint16(header[40:42], uint16(count))
	binary.LittleEndian.PutUint32(header[46:50], uint32(recordSize))
	if _, err := tmp.Write(header[:]); err != nil {
		tmp.Close()
		return err
	}
	for _, out := range outputs {
		info, err := os.Stat(out)
		if err != nil {
			tmp.Close()
			return err
		}
		var entryHeader [l2OutputHeaderSize]byte
		binary.LittleEndian.PutUint16(entryHeader[0:2], uint16(len(out)))
		binary.LittleEndian.PutUint32(entryHeader[4:8], uint32(info.Size()))
		if _, err := tmp.Write(entryHeader[:]); err != nil {
			tmp.Close()
			return err
		}
		if _, err := tmp.Write([]byte(out)); err != nil {
			tmp.Close()
			return err
		}
		if err := copyFileContents(tmp, out); err != nil {
			tmp.Close()
			return err
		}
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func copyFileContents(dst *os.File, path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	buf := make([]byte, 32768)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, err2 := dst.Write(buf[:n]); err2 != nil {
				return err2
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func appendL2FromArchive(data []byte, cacheDir string) (uint64, error) {
	if err := os.MkdirAll(filepath.Dir(l2DataPath(cacheDir)), 0o755); err != nil {
		return 0, err
	}
	l2Mutex.Lock()
	defer l2Mutex.Unlock()
	dst, err := os.OpenFile(l2DataPath(cacheDir), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return 0, err
	}
	defer dst.Close()
	offset, err := dst.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	if _, err := dst.Write(data); err != nil {
		return 0, err
	}
	if err := dst.Sync(); err != nil {
		return 0, err
	}
	if err := reloadL2Data(cacheDir); err != nil {
		return 0, err
	}
	return uint64(offset), nil
}

func mapL2Data(cacheDir string) error {
	l2Mutex.RLock()
	if l2Data != nil {
		l2Mutex.RUnlock()
		return nil
	}
	l2Mutex.RUnlock()
	l2Mutex.Lock()
	defer l2Mutex.Unlock()
	if l2Data != nil {
		return nil
	}
	path := l2DataPath(cacheDir)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	if info.Size() == 0 {
		file.Close()
		return nil
	}
	data, err := mmapFile(int(file.Fd()), int(info.Size()))
	if err != nil {
		file.Close()
		return err
	}
	prefetchMappedFile(data)
	l2File = file
	l2Data = data
	return nil
}

func reloadL2Data(cacheDir string) error {
	if l2Data != nil {
		_ = munmapFile(l2Data)
		l2Data = nil
	}
	if l2File != nil {
		_ = l2File.Close()
		l2File = nil
	}
	path := l2DataPath(cacheDir)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	if info.Size() == 0 {
		file.Close()
		return nil
	}
	data, err := mmapFile(int(file.Fd()), int(info.Size()))
	if err != nil {
		file.Close()
		return err
	}
	prefetchMappedFile(data)
	l2File = file
	l2Data = data
	return nil
}

func actionCacheStoreSync(inputs []string, action string, outputs []string, env []string, cacheDir string) error {
	if err := ensureActionDirs(cacheDir); err != nil {
		return err
	}
	digest, err := actionCacheKey(inputs, action, env)
	if err != nil {
		return err
	}
	archive := archivePath(hex.EncodeToString(digest[:]), cacheDir)
	if err := writeActionArchive(archive, outputs, digest); err != nil {
		return err
	}
	offset, err := appendL2FromArchiveFromDisk(archive, cacheDir)
	if err != nil {
		return err
	}
	l1Store(l1Key(digest), digest, 0, offset)
	logger.Debug("action cache stored\n")
	return nil
}

func appendL2FromArchiveFromDisk(path string, cacheDir string) (uint64, error) {
	data, err := readFileMaybeIOUring(path)
	if err != nil {
		return 0, err
	}
	return appendL2FromArchive(data, cacheDir)
}

func ensureActionDirs(cacheDir string) error {
	if err := os.MkdirAll(l3DataDir(cacheDir), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l2DataPath(cacheDir)), 0o755); err != nil {
		return err
	}
	return nil
}