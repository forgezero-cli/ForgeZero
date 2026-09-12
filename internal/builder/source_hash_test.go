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

package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func TestRefreshSourceHashesMatchesFileDigests(t *testing.T) {
	dir := t.TempDir()
	files := map[string][]byte{
		"a.c":        []byte("int a(void) { return 1; }\n"),
		"b.c":        []byte("int b(void) { return 2; }\n"),
		"nested/c.h": []byte("#define C_VALUE 3\n"),
	}
	for name, data := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := refreshSourceHashes([]string{dir}); err != nil {
		t.Fatal(err)
	}
	for name := range files {
		path := filepath.Join(dir, name)
		want, err := utils.HashBoomBoomMappedFile(path, utils.BoomBoomContext{})
		if err != nil {
			t.Fatal(err)
		}
		got, ok := sourceHashes[path]
		if !ok {
			t.Fatalf("missing hash for %s", path)
		}
		if got.hash != want {
			t.Fatalf("hash mismatch for %s", path)
		}
	}
}

func TestRefreshSourceHashesUsesMetadataCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cached.c")
	data := []byte("int cached(void) { return 1; }\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	var sentinel [32]byte
	sentinel[0] = 0xa5
	contextDigest, err := utils.BoomBoomContextDigest(utils.BoomBoomContext{})
	if err != nil {
		t.Fatal(err)
	}
	cache := map[string]hashCacheEntry{
		path: {hash: sentinel, context: contextDigest, size: info.Size(), modTime: info.ModTime().UnixNano()},
	}
	if err := refreshSourceHashesWithCache([]string{dir}, cache); err != nil {
		t.Fatal(err)
	}
	if got := sourceHashes[path].hash; got != sentinel {
		t.Fatalf("cached hash = %x, want %x", got, sentinel)
	}
}
