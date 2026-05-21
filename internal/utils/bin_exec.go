package utils

import (
	"runtime"
	"unsafe"
)

//go:noescape
func callRaw0(code uintptr)

//go:noescape
func callRaw2(code uintptr, p *byte, n uintptr) uint8

func ExecRaw(bin []byte) {
	if len(bin) == 0 {
		return
	}
	mem, err := execRawMap(len(bin))
	if err != nil {
		return
	}
	copy(mem, bin)
	if err := execRawProtect(mem); err != nil {
		execRawUnmap(mem)
		return
	}
	callRaw0(uintptr(unsafe.Pointer(&mem[0])))
	runtime.KeepAlive(mem)
	execRawUnmap(mem)
}

func ExecRawXor(data []byte) uint8 {
	var x uint8
	for _, b := range data {
		x ^= b
	}
	return x
}
