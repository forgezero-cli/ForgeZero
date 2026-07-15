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

package assembler

import "testing"

func TestEncoderZeroAllocs(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		e := GetEncoder()
		e.WriteByte(0x90)
		e.Write([]byte{0x90, 0x90})
		_ = e.Bytes()
		PutEncoder(e)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations, got %f", allocs)
	}
}
