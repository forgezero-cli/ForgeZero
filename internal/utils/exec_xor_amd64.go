//go:build amd64
// +build amd64

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

import "unsafe"


func execRawXorAVX2(ptr unsafe.Pointer, n uintptr) uint64 {
    var a1, a2, a3, a4 uint64
    i := uintptr(0)
    for ; i+32 <= n; i += 32 {
        a1 ^= *(*uint64)(unsafe.Pointer(uintptr(ptr) + i))
        a2 ^= *(*uint64)(unsafe.Pointer(uintptr(ptr) + i + 8))
        a3 ^= *(*uint64)(unsafe.Pointer(uintptr(ptr) + i + 16))
        a4 ^= *(*uint64)(unsafe.Pointer(uintptr(ptr) + i + 24))
    }
    acc := a1 ^ a2 ^ a3 ^ a4
    for ; i+8 <= n; i += 8 {
        acc ^= *(*uint64)(unsafe.Pointer(uintptr(ptr) + i))
    }
    var b uint8
    b ^= uint8(acc)
    b ^= uint8(acc >> 8)
    b ^= uint8(acc >> 16)
    b ^= uint8(acc >> 24)
    b ^= uint8(acc >> 32)
    b ^= uint8(acc >> 40)
    b ^= uint8(acc >> 48)
    b ^= uint8(acc >> 56)
    for ; i < n; i++ {
        b ^= *(*byte)(unsafe.Pointer(uintptr(ptr) + i))
    }
    return uint64(b)
}
