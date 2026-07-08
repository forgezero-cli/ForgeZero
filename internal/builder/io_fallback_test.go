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
	"testing"
)

func TestReadFileMaybeIOUringFallback(t *testing.T) {
	t.Setenv("FORGEZERO_IO_URING", "0")
	tmp := t.TempDir()
	path := filepath.Join(tmp, "fallback.dat")
	data := []byte("forgezero-fallback-test")
	if err := writeFileMaybeIOUring(path, data, 0o644); err != nil {
		t.Fatalf("writeFileMaybeIOUring: %v", err)
	}
	read, err := readFileMaybeIOUring(path)
	if err != nil {
		t.Fatalf("readFileMaybeIOUring: %v", err)
	}
	if string(read) != string(data) {
		t.Fatalf("data mismatch: got %q want %q", read, data)
	}
}

func TestWriteFileMaybeIOUringFallback(t *testing.T) {
	t.Setenv("FORGEZERO_IO_URING", "0")
	tmp := t.TempDir()
	path := filepath.Join(tmp, "write-fallback.dat")
	payload := []byte("write-fallback")
	if err := writeFileMaybeIOUring(path, payload, 0o644); err != nil {
		t.Fatalf("writeFileMaybeIOUring: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("data mismatch: got %q want %q", got, payload)
	}
}
