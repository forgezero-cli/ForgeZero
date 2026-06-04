// Copyright (c) 2026 AlexVoste. All Rights Reserved.

package gloria

// rdi = 7, rsi = 6, rdx = 2, rcx = 1, r8 = 8, r9 = 9
// r15 = 15 (reserved for VGA cursor in bare-metal mode)
var abiArgRegs = []int{7, 6, 2, 1, 8, 9}

// Register constants for convenience
const (
	regRAX = 0
	regRCX = 1
	regRDX = 2
	regRBX = 3
	regRSP = 4
	regRBP = 5
	regRSI = 6
	regRDI = 7
	regR8  = 8
	regR9  = 9
	regR10 = 10
	regR11 = 11
	regR12 = 12
	regR13 = 13
	regR14 = 14
	regR15 = 15 // VGA cursor offset
)

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

func emitMovImm16ToReg(out []byte, reg int, v uint16) []byte {
	out = append(out, 0x66, 0xB8+byte(reg))
	out = append(out, byte(v), byte(v>>8))
	return out
}

func emitMovImm8ToReg(out []byte, reg int, v byte) []byte {
	out = append(out, 0xB0+byte(reg), v)
	return out
}

func parseStringLiteral(str string) []byte {
	var outBytes []byte
	for i := 0; i < len(str); i++ {
		if str[i] == '\\' && i+1 < len(str) {
			switch str[i+1] {
			case 'n':
				outBytes = append(outBytes, 10)
				i++
				continue
			case 't':
				outBytes = append(outBytes, 9)
				i++
				continue
			}
		}
		outBytes = append(outBytes, str[i])
	}
	return outBytes
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

func emitBareMetalPrint(out []byte, str string) []byte {
	strBytes := parseStringLiteral(str)
	out = emitMovImm64ToReg(out, 2, 0xB8000)
	out = emitMovImm64ToReg(out, 1, uint64(len(strBytes)))
	out = append(out, 0x48, 0x8D, 0x35, 0, 0, 0, 0)
	leaDispPos := len(out) - 4
	loopStart := len(out)
	out = append(
		out,
		0x8A, 0x06,
		0xB4, 0x0A,
		0x66, 0x89, 0x02,
		0x48, 0x83, 0xC2, 0x02,
		0x48, 0xFF, 0xC6,
		0x48, 0xFF, 0xC9,
		0x0F, 0x85, 0, 0, 0, 0,
	)
	jnzDispPos := len(out) - 4
	jmpDisp := int32(loopStart - (jnzDispPos + 4))
	out[jnzDispPos] = byte(jmpDisp)
	out[jnzDispPos+1] = byte(jmpDisp >> 8)
	out[jnzDispPos+2] = byte(jmpDisp >> 16)
	out[jnzDispPos+3] = byte(jmpDisp >> 24)
	out = append(out, 0xEB, byte(len(strBytes)))

	dataStart := len(out)
	out = append(out, strBytes...)

	disp := int32(dataStart - (leaDispPos + 4))
	out[leaDispPos] = byte(disp)
	out[leaDispPos+1] = byte(disp >> 8)
	out[leaDispPos+2] = byte(disp >> 16)
	out[leaDispPos+3] = byte(disp >> 24)
	return out
}

func emitLowLevelPrint(out []byte, str string, kernelMode bool) []byte {
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

	if kernelMode {
		disp := int32(-(strLen + 7))
		out = append(out, 0x48, 0x8D, 0x3D, byte(disp), byte(disp>>8), byte(disp>>16), byte(disp>>24))
		out = append(out, 0xE8, 0x00, 0x00, 0x00, 0x00)
	} else {
		disp := int32(-(strLen + 7))
		out = append(out, 0x48, 0x8D, 0x35, byte(disp), byte(disp>>8), byte(disp>>16), byte(disp>>24))
		out = append(out, 0x50)
		out = append(out, 0x57)
		out = append(out, 0x52)
		out = append(out, 0x56)

		out = append(out, 0x48, 0xC7, 0xC7, 0x01, 0x00, 0x00, 0x00)
		out = append(out, 0x48, 0xC7, 0xC2, byte(strLen), byte(strLen>>8), byte(strLen>>16), byte(strLen>>24))
		out = append(out, 0x48, 0xC7, 0xC0, 0x01, 0x00, 0x00, 0x00)
		out = append(out, 0x0F, 0x05)

		out = append(out, 0x5E)
		out = append(out, 0x5A)
		out = append(out, 0x5F)
		out = append(out, 0x58)

	}
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

func emitPushReg(out []byte, reg int) []byte {
	if reg >= 8 {
		out = append(out, 0x41)
		out = append(out, 0x50+byte(reg-8))
	} else {
		out = append(out, 0x50+byte(reg))
	}
	return out
}

func emitPopReg(out []byte, reg int) []byte {
	if reg >= 8 {
		out = append(out, 0x41)
		out = append(out, 0x58+byte(reg-8))
	} else {
		out = append(out, 0x58+byte(reg))
	}
	return out
}

func emitIn8WithPreserve(out []byte, portVal uint16) []byte {
	out = emitPushReg(out, regRCX)
	out = emitPushReg(out, regRDI)

	out = emitMovImm16ToReg(out, regRDX, portVal)

	out = append(out, 0xEC)

	out = append(out, 0x48, 0x0F, 0xB6, 0xC0)

	out = emitPopReg(out, regRDI)
	out = emitPopReg(out, regRCX)

	return out
}

func emitOut8WithPreserve(out []byte, portVal uint16, dataVal byte) []byte {
	out = emitPushReg(out, regRCX)
	out = emitPushReg(out, regRDI)
	out = emitPushReg(out, regRAX)

	out = emitMovImm16ToReg(out, regRDX, portVal)

	out = emitMovImm8ToReg(out, regRAX, dataVal)

	out = append(out, 0xEE)

	out = emitPopReg(out, regRAX)
	out = emitPopReg(out, regRDI)
	out = emitPopReg(out, regRCX)

	return out
}

func emitNumberPrint(out []byte, numLit string) ([]byte, error) {
	var num int64
	i := 0
	for i < len(numLit) && numLit[i] >= '0' && numLit[i] <= '9' {
		num = num*10 + int64(numLit[i]-'0')
		i++
	}

	if num == 0 {
		out = append(out, 0x6A, '0')
		out = append(out, 0x48, 0xC7, 0xC0, 0x01, 0x00, 0x00, 0x00)
		out = append(out, 0x48, 0xC7, 0xC7, 0x01, 0x00, 0x00, 0x00)
		out = append(out, 0x48, 0x8D, 0x74, 0x24, 0x00)
		out = append(out, 0x48, 0xC7, 0xC2, 0x01, 0x00, 0x00, 0x00)
		out = append(out, 0x0F, 0x05)
		out = append(out, 0x48, 0x83, 0xC4, 0x08)
		return out, nil
	}

	var digits [20]byte
	n := num
	idx := 20
	for n > 0 {
		idx--
		digits[idx] = byte('0' + n%10)
		n /= 10
	}

	for j := idx; j < 20; j++ {
		out = append(out, 0x6A, digits[j])
		out = append(out, 0x48, 0xC7, 0xC0, 0x01, 0x00, 0x00, 0x00)
		out = append(out, 0x48, 0xC7, 0xC7, 0x01, 0x00, 0x00, 0x00)
		out = append(out, 0x48, 0x8D, 0x74, 0x24, 0x00)
		out = append(out, 0x48, 0xC7, 0xC2, 0x01, 0x00, 0x00, 0x00)
		out = append(out, 0x0F, 0x05)
		out = append(out, 0x48, 0x83, 0xC4, 0x08)
	}
	return out, nil
}

func emitRegisterPrint(out []byte, regName string) ([]byte, error) {
	regMap := map[string]int{
		"rax": 0, "rcx": 1, "rdx": 2, "rbx": 3,
		"rsi": 6, "rdi": 7, "r8": 8, "r9": 9,
		"r10": 10, "r11": 11, "r12": 12, "r13": 13, "r14": 14, "r15": 15,
	}
	regIdx, ok := regMap[regName]
	if !ok {
		return out, nil
	}

	out = emitPushReg(out, regRAX)
	out = emitPushReg(out, regRCX)
	out = emitPushReg(out, regRDX)

	if regIdx != regRAX {
		if regIdx <= 7 {
			out = emitMovRegToReg(out, regIdx, regRAX)
		} else {
			out = append(out, 0x49, 0x89, 0xC0+byte(regIdx-8))
		}
	}

	out = emitSyscallPrintReg(out)

	out = emitPopReg(out, regRDX)
	out = emitPopReg(out, regRCX)
	out = emitPopReg(out, regRAX)

	return out, nil
}

func emitSyscallPrintReg(out []byte) []byte {
	out = append(out, 0x50)
	out = append(out, 0x48, 0xC7, 0xC0, 0x01, 0x00, 0x00, 0x00)
	out = append(out, 0x48, 0xC7, 0xC7, 0x01, 0x00, 0x00, 0x00)
	out = emitPopReg(out, regRSI)
	out = append(out, 0x48, 0xC7, 0xC2, 0x01, 0x00, 0x00, 0x00)
	out = append(out, 0x0F, 0x05)
	return out
}
