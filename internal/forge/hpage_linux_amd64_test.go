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

package forge

import (
	"testing"
	"unsafe"
)

func TestMmapHugepagesFallbackAndUnmap(t *testing.T) {
	const size = 8192
	if MmapHugepages(0) != nil {
		t.Fatal("zero-size mapping returned a pointer")
	}
	pointer := MmapHugepages(size)
	if pointer == nil {
		t.Fatal("anonymous mmap fallback failed")
	}
	data := unsafe.Slice((*byte)(pointer), size)
	data[0] = 0x5a
	data[size-1] = 0xa5
	if data[0] != 0x5a || data[size-1] != 0xa5 {
		t.Fatal("mapped memory did not retain writes")
	}
	MunmapHugepages(pointer, size)
}
