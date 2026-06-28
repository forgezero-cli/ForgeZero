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

package watcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWatcherAddRecursiveWalkError(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "blocked")

	if err := os.Mkdir(sub, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chmod(sub, 0o755); err != nil {
			t.Errorf("failed to restore permissions: %v", err)
		}
	}()

	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.AddRecursive(dir); err == nil {
		t.Fatal("expected walk error")
	}
}
