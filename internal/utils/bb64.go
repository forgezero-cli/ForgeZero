/*
 * Copyright (c) 2026 forgezero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package utils

import (
	"encoding/binary"
	"math/bits"

	"golang.org/x/sys/cpu"
)

const bb64Mul uint64 = 0x9e3779b97f4a7c15
const bb64Mul32 uint64 = 0x7f4a7c15

func HashBB64(data []byte, seed uint64) uint64 {
	if cpu.X86.HasAVX2 {
		return HashBB64Asm(data, seed)
	}
	return hashBB64Scalar(data, seed)
}

func hashBB64Scalar(data []byte, seed uint64) uint64 {
	var lanes [4]uint64
	for offset := 0; offset+32 <= len(data); offset += 32 {
		for i := range lanes {
			x := binary.LittleEndian.Uint64(data[offset+i*8:]) ^ seed
			mixed := x ^ (x >> 29) ^ (x << 35)
			x = mixed
			x ^= uint64(uint32(x) * uint32(bb64Mul32))
			lanes[i] ^= x
		}
	}
	tailData := data[len(data)&^31:]
	tail := seed
	for _, value := range tailData {
		tail ^= uint64(value)
		tail = bits.RotateLeft64(tail, 13) * bb64Mul
	}
	hash := lanes[0] ^ lanes[1] ^ lanes[2] ^ lanes[3] ^ tail
	hash ^= hash >> 33
	hash *= bb64Mul
	hash ^= hash >> 29
	return hash
}

func expandBB64Seed(data []byte, kind uint8, index int, base uint64, output *[32]byte) {
	seed := base ^ uint64(kind)<<56 ^ uint64(index)*0xd6e8feb86659fd93 ^ 0x4242363400000001
	hash := HashBB64(data, seed)
	for i := 0; i < 4; i++ {
		hash ^= hash >> 29
		hash *= bb64Mul
		binary.LittleEndian.PutUint64(output[i*8:], hash)
	}
}

func expandBB64(data []byte, kind uint8, index int, output *[32]byte) {
	expandBB64Seed(data, kind, index, 0, output)
}

func hashBB64Pair(left, right [32]byte) [32]byte {
	return hashBB64PairSeed(left, right, 0x4242363400000001)
}

func hashBB64PairSeed(left, right [32]byte, seed uint64) [32]byte {
	var data [64]byte
	copy(data[:32], left[:])
	copy(data[32:], right[:])
	var output [32]byte
	hash := hashBB64Scalar(data[:], seed) ^ hashBB64Scalar(data[:], seed^0xd6e8feb86659fd93)
	for i := 0; i < 4; i++ {
		hash ^= hash >> 29
		hash *= bb64Mul
		binary.LittleEndian.PutUint64(output[i*8:], hash)
	}
	return output
}
