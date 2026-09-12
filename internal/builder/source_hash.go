/*
 * Copyright (c) 2026 forgezero-cli
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

package builder

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

var sourceHashes = make(map[string]hashCacheEntry)

func refreshSourceHashes(dirs []string) error {
	return refreshSourceHashesWithCacheAndContext(dirs, nil, utils.BoomBoomContext{})
}

func refreshSourceHashesWithCache(dirs []string, cache map[string]hashCacheEntry) error {
	return refreshSourceHashesWithCacheAndContext(dirs, cache, utils.BoomBoomContext{})
}

func refreshSourceHashesWithConfig(dirs []string, cache map[string]hashCacheEntry, cfg *config.Config) error {
	return refreshSourceHashesWithCacheAndContext(dirs, cache, boomBoomBuildContext(cfg))
}

func boomBoomBuildContext(cfg *config.Config) utils.BoomBoomContext {
	if cfg == nil {
		return utils.BoomBoomContext{}
	}
	flags := make([]string, 0, len(cfg.Flags.Cc)+len(cfg.Flags.Asm)+len(cfg.Flags.Ld)+len(cfg.InstructionSets)+3)
	flags = append(flags, cfg.Flags.Cc...)
	flags = append(flags, cfg.Flags.Asm...)
	flags = append(flags, cfg.Flags.Ld...)
	flags = append(flags, cfg.InstructionSets...)
	if cfg.Sysroot != "" {
		flags = append(flags, "--sysroot="+cfg.Sysroot)
	}
	if cfg.CPUTarget != "" {
		flags = append(flags, "--cpu-target="+cfg.CPUTarget)
	}
	return utils.BoomBoomContext{
		Compiler: cfg.Toolchain + "|" + cfg.Compiler.Path,
		Version:  cfg.Profile,
		Target:   cfg.Target,
		Flags:    flags,
	}
}

func refreshSourceHashesWithCacheAndContext(dirs []string, cache map[string]hashCacheEntry, context utils.BoomBoomContext) error {
	contextDigest, err := utils.BoomBoomContextDigest(context)
	if err != nil {
		return err
	}
	paths := make([]string, 0)
	metadata := make([]hashCacheEntry, 0)
	for _, root := range dirs {
		if root == "" {
			continue
		}
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				fi, serr := os.Stat(path)
				if serr != nil {
					return serr
				}
				if fi.IsDir() {
					return nil
				}
				metadata = append(metadata, hashCacheEntry{size: fi.Size(), modTime: fi.ModTime().UnixNano()})
			} else {
				fi, serr := d.Info()
				if serr != nil {
					return serr
				}
				metadata = append(metadata, hashCacheEntry{size: fi.Size(), modTime: fi.ModTime().UnixNano()})
			}
			paths = append(paths, path)
			return nil
		}); err != nil {
			return err
		}
	}
	result := make(map[string]hashCacheEntry, len(paths))
	if len(paths) == 0 {
		sourceHashes = result
		return nil
	}
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > len(paths) {
		workers = len(paths)
	}
	type hashJob struct {
		path  string
		index int
	}
	hashes := make([]hashCacheEntry, len(paths))
	pending := 0
	for index, path := range paths {
		if entry, ok := cache[path]; ok && entry.context == contextDigest && entry.modTime == metadata[index].modTime && entry.size == metadata[index].size && entry.modTime != 0 {
			hashes[index] = entry
			continue
		}
		pending++
	}
	if pending == 0 {
		for index, path := range paths {
			result[path] = hashes[index]
		}
		sourceHashes = result
		return nil
	}
	jobs := make(chan hashJob, workers)
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex
	var stopped atomic.Bool
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if stopped.Load() {
					continue
				}
				sum, hashErr := utils.HashBoomBoomMappedFile(job.path, context)
				if hashErr == nil {
					hashes[job.index] = hashCacheEntry{
						hash:    sum,
						context: contextDigest,
						size:    metadata[job.index].size,
						modTime: metadata[job.index].modTime,
					}
				} else {
					errMu.Lock()
					if firstErr == nil {
						firstErr = hashErr
					}
					errMu.Unlock()
					stopped.Store(true)
				}
			}
		}()
	}
	for index, path := range paths {
		if hashes[index].modTime == 0 {
			jobs <- hashJob{path: path, index: index}
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	for index, path := range paths {
		result[path] = hashes[index]
	}
	sourceHashes = result
	return nil
}
