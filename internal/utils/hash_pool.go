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

import (
	"sync"

	"github.com/forgezero-cli/ForgeZero/internal/hashpool"
	"github.com/zeebo/blake3"
)

var (
	hashKey = [32]byte{0x9d, 0x74, 0x31, 0x6f, 0xd5, 0x23, 0x1b, 0xe4, 0xa1, 0x8f, 0x03, 0x71, 0x42, 0x5d, 0x6b, 0x9a, 0x3c, 0xf4, 0x75, 0x28, 0x0d, 0x62, 0x8a, 0x19, 0xbf, 0x4e, 0x50, 0x33, 0x13, 0x21, 0x97, 0x6c}
	keyedHasherPool = sync.Pool{New: func() any {
		h, _ := blake3.NewKeyed(hashKey[:])
		return h
	}}
)

func getKeyedHasher() *blake3.Hasher {
	return keyedHasherPool.Get().(*blake3.Hasher)
}

func putKeyedHasher(h *blake3.Hasher) {
	h.Reset()
	keyedHasherPool.Put(h)
}

func GetHasher() *hashpool.Hasher {
	return hashpool.GetHasher()
}

func PutHasher(h *hashpool.Hasher) {
	hashpool.PutHasher(h)
}
