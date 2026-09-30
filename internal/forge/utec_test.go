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

func TestUTECLayout(t *testing.T) {
	if size := unsafe.Sizeof(UTEC{}); size != 64 {
		t.Fatalf("UTEC size = %d, want 64", size)
	}
	if got := []uintptr{
		unsafe.Offsetof(UTEC{}.Kind),
		unsafe.Offsetof(UTEC{}.Flags),
		unsafe.Offsetof(UTEC{}.ID),
		unsafe.Offsetof(UTEC{}.SrcPtr),
		unsafe.Offsetof(UTEC{}.DstPtr),
		unsafe.Offsetof(UTEC{}.Len),
		unsafe.Offsetof(UTEC{}.Aux1),
		unsafe.Offsetof(UTEC{}.Aux2),
		unsafe.Offsetof(UTEC{}.RetVal),
		unsafe.Offsetof(UTEC{}.Aux3),
	}; got[0] != 0 || got[1] != 2 || got[2] != 4 || got[3] != 8 || got[4] != 16 || got[5] != 24 || got[6] != 32 || got[7] != 40 || got[8] != 48 || got[9] != 56 {
		t.Fatalf("UTEC offsets = %v", got)
	}
	if got := utecSizeAsm(); got != unsafe.Sizeof(UTEC{}) {
		t.Fatalf("assembly UTEC size = %d, Go size = %d", got, unsafe.Sizeof(UTEC{}))
	}
}
