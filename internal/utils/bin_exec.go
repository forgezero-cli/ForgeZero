package utils

import (
	"encoding/binary"
	"runtime"
	"unsafe"
)

//go:noescape
func callRaw0(code uintptr)

//go:noescape
func callRaw2(code uintptr, p *byte, n uintptr) uint8

//go:noescape
func callRawRet(code uintptr) uint64

func ExecRaw(bin []byte) {
	if len(bin) == 0 {
		return
	}
	mem, err := execRawMap(len(bin))
	if err != nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			execRawUnmap(mem)
			panic(r)
		}
	}()
	copy(mem, bin)
	if err := execRawProtect(mem); err != nil {
		execRawUnmap(mem)
		return
	}
	callRaw0(uintptr(unsafe.Pointer(&mem[0])))
	runtime.KeepAlive(mem)
	execRawUnmap(mem)
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
	mem, err := execRawMap(len(bin))
	if err != nil {
		return 0
	}
	defer func() {
		if r := recover(); r != nil {
			execRawUnmap(mem)
			panic(r)
		}
	}()
	copy(mem, bin)
	if err := execRawProtect(mem); err != nil {
		execRawUnmap(mem)
		return 0
	}
	out := callRawRet(uintptr(unsafe.Pointer(&mem[0])))
	runtime.KeepAlive(mem)
	execRawUnmap(mem)
	return out
}

func ExecRawXor(data []byte) uint8 {
	var x uint8
	for _, b := range data {
		x ^= b
	}
	return x
}
