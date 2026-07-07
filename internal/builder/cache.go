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
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/forgezero-cli/ForgeZero/internal/config"
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
	mu      sync.RWMutex
	entries map[string]*cachedObject
}

func newObjectCache() *objectCache {
	return &objectCache{entries: make(map[string]*cachedObject)}
}

func (c *objectCache) get(key string) (*cachedObject, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	return entry, ok
}

func (c *objectCache) delete(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

func (c *objectCache) set(key string, object, syms []byte) {
	c.mu.Lock()
	c.entries[key] = &cachedObject{object: append([]byte(nil), object...), syms: append([]byte(nil), syms...)}
	c.mu.Unlock()
}

var ramObjectStore = newObjectCache()

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

func restoreRAMCache(src, obj string, debug bool, mode string) (bool, error) {
	h, err := utils.HashFile(src)
	if err != nil {
		return false, err
	}
	key := buildCacheKey(h, debug, mode)
	entry, ok := ramObjectStore.get(key)
	if !ok {
		return false, nil
	}
	if len(entry.object) == 0 {
		ramObjectStore.delete(key)
		return false, nil
	}
	if err := utils.EnsureDir(obj); err != nil {
		return false, err
	}
	if err := os.WriteFile(obj, entry.object, 0o644); err != nil {
		return false, err
	}
	if len(entry.syms) > 0 {
		_ = os.WriteFile(obj+".syms", entry.syms, 0o644)
	}
	if debug {
		os.Stdout.WriteString("RAM cache restored " + src + " -> " + obj + "\n")
	}
	return true, nil
}

func storeRAMCache(src, obj string, debug bool, mode string) error {
	object, err := os.ReadFile(obj)
	if err != nil {
		return err
	}
	if len(object) == 0 {
		return errors.New("refusing to cache empty object: " + obj)
	}
	syms, err := os.ReadFile(obj + ".syms")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	h, err := utils.HashFile(src)
	if err != nil {
		return err
	}
	key := buildCacheKey(h, debug, mode)
	ramObjectStore.set(key, object, syms)
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
		return "", errors.New("cached file is empty")
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
		return errors.New("refusing to cache empty object: " + obj)
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
		return errors.New("refusing to cache empty object: " + obj)
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