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

package utils

func fnv1aHexUint64(h uint64) string {
	var out [16]byte
	const hextable = "0123456789abcdef"
	for i := 0; i < 8; i++ {
		b := byte(h >> ((7 - i) * 8))
		out[i*2] = hextable[b>>4]
		out[i*2+1] = hextable[b&0x0f]
	}
	return string(out[:])
}

func fnv1aHashString(s string) uint64 {
	const (
		offset uint64 = 1469598103934665603
		prime  uint64 = 1099511628211
	)
	h := offset
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	return h
}

func fnv1aHashAppendString(h uint64, s string) uint64 {
	const prime uint64 = 1099511628211
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	return h
}

func fnv1aHashAppendByte(h uint64, b byte) uint64 {
	const prime uint64 = 1099511628211
	h ^= uint64(b)
	h *= prime
	return h
}

func fnv1aHexFromString(s string) string {
	return fnv1aHexUint64(fnv1aHashString(s))
}
