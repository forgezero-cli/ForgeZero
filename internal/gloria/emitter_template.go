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

var bareMetalPrintHeaderTemplate = [32]byte{
	0x48, 0xBA, 0x00, 0x80, 0x0B, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x48, 0xB9, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x48, 0x8D, 0x35, 0x00,
	0x00, 0x00, 0x00, 0x8A, 0x06, 0xB4, 0x0A, 0x66,
}

var bareMetalPrintHeaderMask = [32]byte{
	12: 0x80,
	13: 0x80,
	14: 0x80,
	15: 0x80,
	16: 0x80,
	17: 0x80,
	18: 0x80,
	19: 0x80,
	23: 0x80,
	24: 0x80,
	25: 0x80,
	26: 0x80,
}

func appendBareMetalPrintHeader(out []byte, length uint64, displacement int32) []byte {
	start := len(out)
	if cap(out)-start < len(bareMetalPrintHeaderTemplate) {
		var reserve [32]byte
		out = append(out, reserve[:]...)
	} else {
		out = out[:start+len(bareMetalPrintHeaderTemplate)]
	}
	writeBareMetalPrintHeader(out[start:], length, displacement)
	return out
}
