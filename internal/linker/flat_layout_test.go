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

package linker

import (
	"testing"
)

func TestNewNakedMemoryLayoutZeroAllocs(t *testing.T) {
	flash := Region{"FLASH", 0x08000000, 0x20000, PermRead | PermExec}
	ram := Region{"RAM", 0x20000000, 0x8000, PermRead | PermWrite}
	allocs := testing.AllocsPerRun(1000, func() {
		_, err := NewNakedMemoryLayout(flash, ram, []byte{1, 2, 3, 4}, []byte{5, 6, 7, 8}, 16)
		if err != nil {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("allocs %g, want 0", allocs)
	}
}

func TestEmitFlatBinaryLayout(t *testing.T) {
	flash := Region{"FLASH", 0x08000000, 0x20000, PermRead | PermExec}
	ram := Region{"RAM", 0x20000000, 0x8000, PermRead | PermWrite}
	layout, err := NewNakedMemoryLayout(flash, ram, []byte{1, 2, 3}, []byte{16, 17}, 8)
	if err != nil {
		t.Fatal(err)
	}
	if layout.Sections[0].Origin != 0x08000000 {
		t.Fatalf("text origin %x", layout.Sections[0].Origin)
	}
	if layout.Sections[1].Origin != 0x20000000 {
		t.Fatalf("data origin %x", layout.Sections[1].Origin)
	}
	if layout.Sections[2].Origin != 0x20000004 {
		t.Fatalf("bss origin %x", layout.Sections[2].Origin)
	}
	out, err := EmitFlatBinary(layout)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{
		1, 2, 3, 0,
		16, 17, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}
	if len(out) != len(want) {
		t.Fatalf("len %d want %d", len(out), len(want))
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("byte %d = %x want %x", i, out[i], want[i])
		}
	}
}
