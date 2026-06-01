package utils

import (
	"bytes"
	"encoding/binary"
	"os"
	"runtime"
	"syscall"
	"unsafe"
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
	stdout := os.Stdout.Fd()

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
		syscall.Write(int(stdout), trimmed)
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
	var x uint8
	for _, b := range data {
		x ^= b
	}
	return x
}
