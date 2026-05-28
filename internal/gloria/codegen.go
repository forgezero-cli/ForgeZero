package gloria

// rdi = 7, rsi = 6, rdx = 2, rcx = 1, r8 = 8, r9 = 9
var abiArgRegs = []int{7, 6, 2, 1, 8, 9}

func emitMovRegToStack(out []byte, srcReg int, offset int) []byte {
	modrm := byte(0x80 | (srcReg << 3) | 5)
	out = append(out, 0x48, 0x89, modrm)
	disp := int32(offset)
	out = append(out, byte(disp), byte(disp>>8), byte(disp>>16), byte(disp>>24))
	return out
}

func emitMovStackToReg(out []byte, dstReg int, offset int) []byte {
	modrm := byte(0x80 | (dstReg << 3) | 5)
	out = append(out, 0x48, 0x8B, modrm)
	disp := int32(offset)
	out = append(out, byte(disp), byte(disp>>8), byte(disp>>16), byte(disp>>24))
	return out
}

func emitCmpRegToReg(out []byte, src, dst int) []byte {
	modrm := byte(0xC0 | (src << 3) | dst)
	return append(out, 0x48, 0x39, modrm)
}

func emitCondJmp(out []byte, op byte) (int, []byte) {
	out = append(out, op, 0x00)
	return len(out) - 1, out
}

func emitMovImm64ToReg(out []byte, reg int, v uint64) []byte {
	out = append(out, 0x48, 0xB8+byte(reg))
	out = append(
		out,
		byte(v),
		byte(v>>8),
		byte(v>>16),
		byte(v>>24),
		byte(v>>32),
		byte(v>>40),
		byte(v>>48),
		byte(v>>56),
	)
	return out
}

func emitMovRegToReg(out []byte, src, dst int) []byte {
	modrm := byte(0xC0 | (src << 3) | dst)
	return append(out, 0x48, 0x89, modrm)
}

func emitAddRegToReg(out []byte, src, dst int) []byte {
	modrm := byte(0xC0 | (src << 3) | dst)
	return append(out, 0x48, 0x01, modrm)
}

func emitSubRegToReg(out []byte, src, dst int) []byte {
	modrm := byte(0xC0 | (src << 3) | dst)
	return append(out, 0x48, 0x29, modrm)
}

func emitAddImm64ToReg(out []byte, reg int, v uint64) []byte {
	out = append(out, 0x48, 0x81, byte(0xC0|reg))
	out = append(out, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
	return out
}

func emitSubImm64ToReg(out []byte, reg int, v uint64) []byte {
	out = append(out, 0x48, 0x81, byte(0xE8|reg))
	out = append(out, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
	return out
}

func emitLowLevelPrint(out []byte, str string) []byte {
	var strBytes []byte
	for i := 0; i < len(str); i++ {
		if str[i] == '\\' && i+1 < len(str) {
			if str[i+1] == 'n' {
				strBytes = append(strBytes, 10)
				i++
				continue
			} else if str[i+1] == 't' {
				strBytes = append(strBytes, 9)
				i++
				continue
			}
		}
		strBytes = append(strBytes, str[i])
	}
	strLen := len(strBytes)

	out = append(out, 0xEB, byte(strLen))
	out = append(out, strBytes...)

	disp := int32(-(strLen + 7))
	out = append(out, 0x48, 0x8D, 0x35, byte(disp), byte(disp>>8), byte(disp>>16), byte(disp>>24))

	out = append(out, 0x50) // push rax
	out = append(out, 0x57) // push rdi
	out = append(out, 0x52) // push rdx
	out = append(out, 0x56) // push rsi

	out = append(out, 0x48, 0xC7, 0xC7, 0x01, 0x00, 0x00, 0x00)
	out = append(out, 0x48, 0xC7, 0xC2, byte(strLen), byte(strLen>>8), byte(strLen>>16), byte(strLen>>24))
	out = append(out, 0x48, 0xC7, 0xC0, 0x01, 0x00, 0x00, 0x00)
	out = append(out, 0x0F, 0x05) // syscall

	out = append(out, 0x5E) // pop rsi
	out = append(out, 0x5A) // pop rdx
	out = append(out, 0x5F) // pop rdi
	out = append(out, 0x58) // pop rax

	return out
}

func peephole(ins []byte) []byte {
	out := make([]byte, 0, len(ins))
	i := 0
	for i < len(ins) {
		if i+3 <= len(ins) && ins[i] == 0x48 && ins[i+1] == 0x89 {
			mod := ins[i+2]
			src := (mod >> 3) & 7
			dst := mod & 7
			if src == dst {
				i += 3
				continue
			}
		}
		out = append(out, ins[i])
		i++
	}
	return out
}
