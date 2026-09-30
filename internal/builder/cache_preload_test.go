/*
 *   Copyright (c) 2026 forgezero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version of the License.
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
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestPreloadCachePopulatesL1(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, ".fz_cache")
	if err := os.MkdirAll(filepath.Join(cacheDir, "actions"), 0o755); err != nil {
		t.Fatal(err)
	}
	var data [32]byte
	for i := range data {
		data[i] = byte(i)
	}
	hash := hex.EncodeToString(data[:])
	path := filepath.Join(cacheDir, "actions", hash+".dat")
	if err := os.WriteFile(path, []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := PreloadCache(context.Background(), cacheDir); err != nil {
		t.Fatal(err)
	}
	preloadWait.Wait()

	idx := l1Key(data)
	entry, ok := l1Load(idx)
	if !ok {
		t.Fatal("expected l1Load to find entry")
	}
	if entry.hash != data {
		t.Fatalf("expected hash %x, got %x", data, entry.hash)
	}
}

func TestL1LoadSnapshotConcurrentStore(t *testing.T) {
	const key = uint64(0x7f12000000004321)
	var initial [32]byte
	initial[8] = 1
	l1Store(key, initial, 1, 2)

	var group sync.WaitGroup
	group.Add(5)
	for reader := 0; reader < 4; reader++ {
		go func() {
			defer group.Done()
			for iteration := 0; iteration < 5000; iteration++ {
				entry, ok := l1Load(key)
				if !ok {
					t.Error("cache entry disappeared")
					return
				}
				value := uint32(entry.hash[8])
				if entry.size != value || entry.offset != uint64(value)+2 {
					t.Errorf("inconsistent cache snapshot: size=%d offset=%d hash-byte=%d", entry.size, entry.offset, value)
					return
				}
			}
		}()
	}
	go func() {
		defer group.Done()
		for iteration := 0; iteration < 5000; iteration++ {
			value := byte(iteration%255 + 1)
			var digest [32]byte
			digest[8] = value
			l1Store(key, digest, uint32(value), uint64(value)+1)
		}
	}()
	group.Wait()
}

func TestPreloadCacheNoCacheDirectory(t *testing.T) {
	if err := PreloadCache(context.Background(), ""); err != nil {
		t.Fatalf("expected no error for empty cache dir, got %v", err)
	}
}

func TestPreloadCacheDoesNotBlockBuild(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, ".fz_cache")
	if err := os.MkdirAll(filepath.Join(cacheDir, "actions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "actions", "0000000000000000000000000000000000000000000000000000000000000000.dat"), []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := PreloadCache(ctx, cacheDir); err != nil {
		t.Fatal(err)
	}

	select {
	case <-context.Background().Done():
		t.Fatal("unexpected build context done")
	default:
	}
}
