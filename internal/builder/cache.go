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
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	fzerr "github.com/forgezero-cli/ForgeZero/internal/errors"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

type cacheMode string

const (
	cacheDisk cacheMode = "disk"
	cacheRAM  cacheMode = "ram"
	cacheOff  cacheMode = "off"
)

type cachedObject struct {
	object []byte
	syms   []byte
}

type objectCache struct {
	entries sync.Map
}

func newObjectCache() *objectCache { return &objectCache{} }

func (c *objectCache) get(key string) (*cachedObject, bool) {
	v, ok := c.entries.Load(key)
	if !ok {
		return nil, false
	}
	return v.(*cachedObject), true
}

func (c *objectCache) delete(key string) {
	if v, ok := c.entries.Load(key); ok {
		if ent, ok2 := v.(*cachedObject); ok2 && ent != nil {
			c.entries.Delete(key)
			size := int64(len(ent.object) + len(ent.syms))
			if size > 0 {
				atomic.AddInt64(&ramCacheUsedBytes, -size)
			}
			data := ent.object
			if len(data) > 0 {
				_ = munmapFile(data)
				ent.object = nil
				runtime.KeepAlive(data)
			}
			return
		}
	}
	c.entries.Delete(key)
}

func (c *objectCache) set(key string, object, syms []byte) {
	if existing, ok := c.entries.Load(key); ok {
		if ent, ok2 := existing.(*cachedObject); ok2 && ent != nil {
			c.delete(key)
		}
	}
	c.entries.Store(key, &cachedObject{object: object, syms: append([]byte(nil), syms...)})
}

var ramObjectStore = newObjectCache()
var ramCacheHits *utils.NumaCounters
var ramCacheMisses *utils.NumaCounters
var ramCacheCapacityBytes int64
var ramCacheUsedBytes int64

type cacheTask struct {
	src      string
	obj      string
	cacheDir string
	debug    bool
	verbose  bool
	mode     string
}

var cacheWriteCh chan cacheTask

func init() {
	cacheWriteCh = make(chan cacheTask, 1024)
	workers := runtime.GOMAXPROCS(0)
	if workers <= 0 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		go func() {
			for t := range cacheWriteCh {
				_ = storeCache(t.src, t.obj, t.cacheDir, t.debug, t.verbose, t.mode)
			}
		}()
	}
	ramCacheHits = utils.NewNumaCounters()
	ramCacheMisses = utils.NewNumaCounters()
}

func AsyncStoreCache(src, obj, cacheDir string, debug, verbose bool, mode string) error {
	t := cacheTask{src: src, obj: obj, cacheDir: cacheDir, debug: debug, verbose: verbose, mode: mode}
	cacheWriteCh <- t
	return nil
}

type shadowTask struct {
	src   string
	obj   string
	debug bool
	mode  string
}

var shadowWriteCh = make(chan shadowTask, 256)

func init() {
	go func() {
		for t := range shadowWriteCh {
			_ = storeShadowCache(t.src, t.obj, t.debug, t.mode)
		}
	}()
}

func AsyncStoreShadowCache(src, obj string, debug bool, mode string) error {
	t := shadowTask{src: src, obj: obj, debug: debug, mode: mode}
	shadowWriteCh <- t
	return nil
}

type pathBuffer struct {
	buf   [2048]byte
	n     int
	extra []byte
}

func (p *pathBuffer) appendString(s string) {
	if p.extra != nil {
		p.extra = append(p.extra, s...)
		return
	}
	if len(s)+p.n <= len(p.buf) {
		copy(p.buf[p.n:], s)
		p.n += len(s)
		return
	}
	p.extra = append(p.extra, p.buf[:p.n]...)
	p.extra = append(p.extra, s...)
}

func (p *pathBuffer) appendByte(b byte) {
	if p.extra != nil {
		p.extra = append(p.extra, b)
		return
	}
	if p.n < len(p.buf) {
		p.buf[p.n] = b
		p.n++
		return
	}
	p.extra = append(p.extra, p.buf[:p.n]...)
	p.extra = append(p.extra, b)
}

func (p *pathBuffer) appendBytes(b []byte) {
	if p.extra != nil {
		p.extra = append(p.extra, b...)
		return
	}
	if len(b)+p.n <= len(p.buf) {
		copy(p.buf[p.n:], b)
		p.n += len(b)
		return
	}
	p.extra = append(p.extra, p.buf[:p.n]...)
	p.extra = append(p.extra, b...)
}

func (p *pathBuffer) String() string {
	if p.extra != nil {
		return string(p.extra)
	}
	return string(p.buf[:p.n])
}

func joinPath(base, name string) string {
	var pb pathBuffer
	pb.appendString(base)
	if len(base) > 0 && base[len(base)-1] != byte(os.PathSeparator) {
		pb.appendByte(byte(os.PathSeparator))
	}
	pb.appendString(name)
	return pb.String()
}

func buildCacheKey(hash string, debug bool, mode string) string {
	var pb pathBuffer
	pb.appendString(hash)
	pb.appendByte('_')
	if debug {
		pb.appendByte('1')
	} else {
		pb.appendByte('0')
	}
	pb.appendByte('_')
	pb.appendString(mode)
	return pb.String()
}

func cacheEntryPath(dir, key string) string {
	var pb pathBuffer
	pb.appendString(dir)
	if len(dir) > 0 && dir[len(dir)-1] != byte(os.PathSeparator) {
		pb.appendByte(byte(os.PathSeparator))
	}
	pb.appendString(key)
	return pb.String()
}

func determineCacheMode(cfg *config.Config, noCache bool) cacheMode {
	if noCache {
		return cacheOff
	}
	if cfg == nil {
		return cacheDisk
	}
	if cfg.NoCache {
		return cacheOff
	}
	switch cfg.CacheMode {
	case config.CacheModeRAM:
		return cacheRAM
	case config.CacheModeOff:
		return cacheOff
	default:
		return cacheDisk
	}
}

func SetRAMCacheCapacityMB(mb int) {
	if mb <= 0 {
		atomic.StoreInt64(&ramCacheCapacityBytes, 0)
		return
	}
	atomic.StoreInt64(&ramCacheCapacityBytes, int64(mb)*1024*1024)
}

func RAMCacheCapacityBytes() int64 {
	return atomic.LoadInt64(&ramCacheCapacityBytes)
}

func canStoreRAMCache(size int64) bool {
	max := atomic.LoadInt64(&ramCacheCapacityBytes)
	if max <= 0 {
		return true
	}
	if size > max {
		return false
	}
	if atomic.AddInt64(&ramCacheUsedBytes, size) > max {
		atomic.AddInt64(&ramCacheUsedBytes, -size)
		return false
	}
	return true
}

func restoreRAMCache(src, obj string, debug bool, mode string) (bool, error) {
	h, err := utils.HashFile(src)
	if err != nil {
		return false, err
	}
	key := buildCacheKey(h, debug, mode)
	entry, ok := ramObjectStore.get(key)
	if !ok {
		if ramCacheMisses != nil {
			ramCacheMisses.Inc()
		}
		return false, nil
	}
	if len(entry.object) == 0 {
		ramObjectStore.delete(key)
		if ramCacheMisses != nil {
			ramCacheMisses.Inc()
		}
		return false, nil
	}
	if err := utils.EnsureDir(obj); err != nil {
		return false, err
	}
	if err := writeFileMaybeIOUring(obj, entry.object, 0o644); err != nil {
		return false, err
	}
	if len(entry.syms) > 0 {
		_ = writeFileMaybeIOUring(obj+".syms", entry.syms, 0o644)
	}
	if debug {
		os.Stdout.WriteString("RAM cache restored " + src + " -> " + obj + "\n")
	}
	if ramCacheHits != nil {
		ramCacheHits.Inc()
	}
	return true, nil
}

func storeRAMCache(src, obj string, debug bool, mode string) error {
	f, err := os.Open(obj)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fzerr.NewMsg(fzerr.CodeCacheEmpty, "refusing to cache empty object: "+obj)
	}
	size := int64(info.Size())
	syms, err := readFileMaybeIOUring(obj + ".syms")
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	size += int64(len(syms))
	if !canStoreRAMCache(size) {
		if debug {
			os.Stdout.WriteString("RAM cache skipped " + src + " (limit reached)\n")
		}
		return nil
	}
	fd := int(f.Fd())
	data, err := mmapFile(fd, int(info.Size()))
	if err != nil {
		if _, err2 := readFileMaybeIOUring(obj); err2 != nil {
			atomic.AddInt64(&ramCacheUsedBytes, -size)
			return err
		}
		object, err2 := readFileMaybeIOUring(obj)
		if err2 != nil {
			atomic.AddInt64(&ramCacheUsedBytes, -size)
			return err
		}
		syms, err2 = readFileMaybeIOUring(obj + ".syms")
		if err2 != nil && !os.IsNotExist(err2) {
			atomic.AddInt64(&ramCacheUsedBytes, -size)
			return err2
		}
		h, err2 := utils.HashFile(src)
		if err2 != nil {
			atomic.AddInt64(&ramCacheUsedBytes, -size)
			return err2
		}
		key := buildCacheKey(h, debug, mode)
		ramObjectStore.set(key, object, syms)
		if debug {
			os.Stdout.WriteString("RAM cache stored " + src + "\n")
		}
		return nil
	}
	syms, err = readFileMaybeIOUring(obj + ".syms")
	if err != nil && !os.IsNotExist(err) {
		_ = munmapFile(data)
		atomic.AddInt64(&ramCacheUsedBytes, -size)
		return err
	}
	h, err := utils.HashFile(src)
	if err != nil {
		_ = munmapFile(data)
		atomic.AddInt64(&ramCacheUsedBytes, -size)
		return err
	}
	key := buildCacheKey(h, debug, mode)
	ramObjectStore.set(key, data, syms)
	if debug {
		os.Stdout.WriteString("RAM cache stored " + src + "\n")
	}
	return nil
}

func checkCache(src, cacheDir string, debug, verbose bool, mode string) (string, error) {
	h, err := utils.HashFile(src)
	if err != nil {
		return "", err
	}
	key := buildCacheKey(h, debug, mode)
	cacheObj := cacheEntryPath(cacheDir, key+".o")
	info, err := os.Stat(cacheObj)
	if err != nil {
		return "", err
	}
	if info.Size() == 0 {
		os.Remove(cacheObj)
		return "", fzerr.New(fzerr.CodeCacheEmpty)
	}
	return cacheObj, nil
}

func restoreShadowCache(src, obj string, debug bool, mode string) (bool, error) {
	flags := []string{"debug=" + strconv.FormatBool(debug), "mode=" + mode}
	key, err := utils.ShadowCacheKey(src, flags)
	if err != nil {
		return false, err
	}
	shadowObj := utils.ShadowCachePath(key)
	info, err := os.Stat(shadowObj)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.Size() == 0 {
		os.Remove(shadowObj)
		return false, nil
	}
	if err := utils.EnsureDir(obj); err != nil {
		return false, err
	}
	if err := utils.LinkOrClone(shadowObj, obj); err != nil {
		return false, err
	}
	if err := os.Chmod(obj, utils.FilePerm); err != nil {
		return false, err
	}
	if debug {
		os.Stdout.WriteString("Shadow cache restored " + shadowObj + " -> " + obj + "\n")
	}
	return true, nil
}

func storeCache(src, obj, cacheDir string, debug, verbose bool, mode string) error {
	info, err := os.Stat(obj)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fzerr.NewMsg(fzerr.CodeCacheEmpty, "refusing to cache empty object: "+obj)
	}
	h, err := utils.HashFile(src)
	if err != nil {
		return err
	}
	key := buildCacheKey(h, debug, mode)
	cacheObj := cacheEntryPath(cacheDir, key+".o")
	return utils.CopyFile(obj, cacheObj)
}

func storeShadowCache(src, obj string, debug bool, mode string) error {
	info, err := os.Stat(obj)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fzerr.NewMsg(fzerr.CodeCacheEmpty, "refusing to cache empty object: "+obj)
	}
	flags := []string{"debug=" + strconv.FormatBool(debug), "mode=" + mode}
	key, err := utils.ShadowCacheKey(src, flags)
	if err != nil {
		return err
	}
	shadowObj := utils.ShadowCachePath(key)
	if err := os.MkdirAll(filepath.Dir(shadowObj), 0o755); err != nil {
		return err
	}
	if err := utils.LinkOrClone(obj, shadowObj); err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	return nil
}
