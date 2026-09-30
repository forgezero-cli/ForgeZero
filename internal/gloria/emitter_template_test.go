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

package gloria

import (
	"encoding/binary"
	"testing"
)

func TestBareMetalPrintHeaderTemplate(t *testing.T) {
	var got [32]byte
	displacement := int32(-25)
	writeBareMetalPrintHeader(got[:], 0x0807060504030201, displacement)
	want := bareMetalPrintHeaderTemplate
	binary.LittleEndian.PutUint64(want[12:20], 0x0807060504030201)
	binary.LittleEndian.PutUint32(want[23:27], uint32(displacement))
	if got != want {
		t.Fatalf("header = %x, want %x", got, want)
	}
}
