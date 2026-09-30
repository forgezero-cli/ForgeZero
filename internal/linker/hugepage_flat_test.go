//go:build linux && amd64

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

package linker

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFlatBinaryFileLargeMapping(t *testing.T) {
	const size = 2 << 20
	data := make([]byte, size)
	for index := range data {
		data[index] = byte(index*29 + 7)
	}
	layout, err := NewNakedMemoryLayout(
		Region{Name: "FLASH", Origin: 0x08000000, Length: 4 << 20, Permissions: PermRead | PermExec},
		Region{},
		data,
		nil,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "firmware.bin")
	if err := writeFlatBinaryFile(path, layout, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("hugepage-backed flat binary differs from emitted section")
	}
}
