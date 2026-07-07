/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 */

package assembler

func EmitMovRegImm(reg byte, imm32 uint32) []byte {
    e := GetEncoder()
    EmitMovRegImmTo(e, reg, imm32)
    out := make([]byte, len(e.Bytes()))
    copy(out, e.Bytes())
    PutEncoder(e)
    return out
}

func EmitAddRegReg(dst byte, src byte) []byte {
    e := GetEncoder()
    EmitAddRegRegTo(e, dst, src)
    out := make([]byte, len(e.Bytes()))
    copy(out, e.Bytes())
    PutEncoder(e)
    return out
}

func EmitMovRegImmTo(e *Encoder, reg byte, imm32 uint32) {
    b := byte(0xB8 + (reg & 7))
    e.WriteByte(b)
    e.Write([]byte{
        byte(imm32),
        byte(imm32 >> 8),
        byte(imm32 >> 16),
        byte(imm32 >> 24),
    })
}

func EmitAddRegRegTo(e *Encoder, dst byte, src byte) {
    e.WriteByte(0x48)
    e.WriteByte(0x01)
    modrm := byte(0xC0 | ((src & 7) << 3) | (dst & 7))
    e.WriteByte(modrm)
}

