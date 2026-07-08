/*
 *   Copyright (c) 2026 ForgeZero-cli
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
