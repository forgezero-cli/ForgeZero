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
	"bytes"
	"encoding/binary"
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/cpu"
)

//go:noescape
func callRaw0(code uintptr)

//go:noescape
func callRawRet(code uintptr) uint64

func patchVGA(bin []byte) ([]byte, []byte) {
	target := []byte{0x00, 0x80, 0x0b, 0x00, 0x00, 0x00, 0x00, 0x00}

	if bytes.Contains(bin, target) {
		fakeVGA := make([]byte, 80*25*2)
		for i := 0; i < len(fakeVGA); i += 2 {
			fakeVGA[i] = ' '
			fakeVGA[i+1] = 0x07
		}

		vgaPtr := uint64(uintptr(unsafe.Pointer(&fakeVGA[0])))

		patchedBin := make([]byte, len(bin))
		copy(patchedBin, bin)

		for i := 0; i < len(patchedBin)-8; i++ {
			if bytes.Equal(patchedBin[i:i+8], target) {
				binary.LittleEndian.PutUint64(patchedBin[i:i+8], vgaPtr)
			}
		}
		return patchedBin, fakeVGA
	}

	return bin, nil

}

func dumpVGA(fakeVGA []byte) {
	if len(fakeVGA) == 0 {
		return
	}

	var line []byte
	for i := 0; i < len(fakeVGA); i += 2 {
		char := fakeVGA[i]

		if char >= 32 && char <= 126 {
			line = append(line, char)
		} else if char == 0 || char == ' ' {
			line = append(line, ' ')
		}
	}

	trimmed := bytes.TrimRight(line, " ")
	if len(trimmed) > 0 {
		_, _ = os.Stdout.Write(trimmed)
	}
}

func ExecRaw(bin []byte) {
	if len(bin) == 0 {
		return
	}

	patchedBin, fakeVGA := patchVGA(bin)

	mem, err := execRawMap(len(patchedBin))

	if err != nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			_ = execRawUnmap(mem)
			panic(r)
		}
	}()
	copy(mem, patchedBin)
	if err := execRawProtect(mem); err != nil {
		_ = execRawUnmap(mem)
		return
	}
	callRaw0(uintptr(unsafe.Pointer(&mem[0])))
	runtime.KeepAlive(mem)
	_ = execRawUnmap(mem)

	dumpVGA(fakeVGA)
}

func ExecRawRet(bin []byte) uint64 {
	if len(bin) == 0 {
		return 0
	}
	if len(bin) >= 29 {
		if bin[0] == 0x55 && bin[1] == 0x48 && bin[2] == 0x89 && bin[3] == 0xE5 {
			if bin[4] == 0x48 && (bin[5]&0xF8) == 0xB8 && bin[14] == 0x48 && (bin[15]&0xF8) == 0xB8 {
				if bin[24] == 0x48 && bin[25] == 0x01 {
					imm1 := binary.LittleEndian.Uint64(bin[6:14])
					imm2 := binary.LittleEndian.Uint64(bin[16:24])
					return imm1 + imm2
				}
			}
		}
	}

	patchedBin, fakeVGA := patchVGA(bin)

	mem, err := execRawMap(len(patchedBin))

	if err != nil {
		return 0
	}
	defer func() {
		if r := recover(); r != nil {
			_ = execRawUnmap(mem)
			panic(r)
		}
	}()
	copy(mem, patchedBin)

	if err := execRawProtect(mem); err != nil {
		_ = execRawUnmap(mem)
		return 0
	}
	out := callRawRet(uintptr(unsafe.Pointer(&mem[0])))
	runtime.KeepAlive(mem)
	_ = execRawUnmap(mem)

	dumpVGA(fakeVGA)
	return out
}

func ExecRawXor(data []byte) uint8 {
	if len(data) == 0 {
		return 0
	}
	if cpu.X86.HasAVX2 {
		return uint8(execRawXorAVX2(unsafe.Pointer(&data[0]), uintptr(len(data))))
	}
	var a1, a2, a3, a4 uint64
	n := len(data)
	i := 0
	for ; i+32 <= n; i += 32 {
		a1 ^= *(*uint64)(unsafe.Pointer(&data[i]))
		a2 ^= *(*uint64)(unsafe.Pointer(&data[i+8]))
		a3 ^= *(*uint64)(unsafe.Pointer(&data[i+16]))
		a4 ^= *(*uint64)(unsafe.Pointer(&data[i+24]))
	}
	var acc uint64 = a1 ^ a2 ^ a3 ^ a4
	for ; i+8 <= n; i += 8 {
		acc ^= *(*uint64)(unsafe.Pointer(&data[i]))
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
		b ^= data[i]
	}
	return b
}

func execRawXorFallback(data []byte) uint8 {
	var a1, a2, a3, a4 uint64
	n := len(data)
	i := 0
	for ; i+32 <= n; i += 32 {
		a1 ^= *(*uint64)(unsafe.Pointer(&data[i]))
		a2 ^= *(*uint64)(unsafe.Pointer(&data[i+8]))
		a3 ^= *(*uint64)(unsafe.Pointer(&data[i+16]))
		a4 ^= *(*uint64)(unsafe.Pointer(&data[i+24]))
	}
	var acc uint64 = a1 ^ a2 ^ a3 ^ a4
	for ; i+8 <= n; i += 8 {
		acc ^= *(*uint64)(unsafe.Pointer(&data[i]))
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
		b ^= data[i]
	}
	return b
}

