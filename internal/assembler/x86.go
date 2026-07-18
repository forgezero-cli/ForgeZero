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

package assembler

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type registerInfo struct {
	code     byte
	rexIndex byte
	width    int
}

type modRMRegRM struct {
	modRM byte
	reg   byte
	rm    byte
	rex   byte
}

func parseRegister(tok []byte) (byte, int, bool) {
	if len(tok) == 0 {
		return 0, 0, false
	}
	if tok[0] == 'R' || tok[0] == 'r' {
	}
	var s [16]byte
	copy(s[:], tok)
	for i := 0; i < len(tok) && i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			s[i] = c + ('a' - 'A')
		}
	}
	name := s[:len(tok)]

	codeFromBase := func(code byte, width int) (byte, int, bool) {
		return code & 7, width, true
	}

	switch {
	case bytesEqual(name, []byte("al")):
		return 0, 8, true
	case bytesEqual(name, []byte("cl")):
		return 1, 8, true
	case bytesEqual(name, []byte("dl")):
		return 2, 8, true
	case bytesEqual(name, []byte("bl")):
		return 3, 8, true
	case bytesEqual(name, []byte("ah")):
		return 4, 8, true
	case bytesEqual(name, []byte("ch")):
		return 5, 8, true
	case bytesEqual(name, []byte("dh")):
		return 6, 8, true
	case bytesEqual(name, []byte("bh")):
		return 7, 8, true
	case bytesEqual(name, []byte("ax")):
		return 0, 16, true
	case bytesEqual(name, []byte("cx")):
		return 1, 16, true
	case bytesEqual(name, []byte("dx")):
		return 2, 16, true
	case bytesEqual(name, []byte("bx")):
		return 3, 16, true
	case bytesEqual(name, []byte("sp")):
		return 4, 16, true
	case bytesEqual(name, []byte("bp")):
		return 5, 16, true
	case bytesEqual(name, []byte("si")):
		return 6, 16, true
	case bytesEqual(name, []byte("di")):
		return 7, 16, true
	case bytesEqual(name, []byte("eax")):
		return codeFromBase(0, 32)
	case bytesEqual(name, []byte("ecx")):
		return codeFromBase(1, 32)
	case bytesEqual(name, []byte("edx")):
		return codeFromBase(2, 32)
	case bytesEqual(name, []byte("ebx")):
		return codeFromBase(3, 32)
	case bytesEqual(name, []byte("esp")):
		return codeFromBase(4, 32)
	case bytesEqual(name, []byte("ebp")):
		return codeFromBase(5, 32)
	case bytesEqual(name, []byte("esi")):
		return codeFromBase(6, 32)
	case bytesEqual(name, []byte("edi")):
		return codeFromBase(7, 32)
	case bytesEqual(name, []byte("rax")):
		return codeFromBase(0, 64)
	case bytesEqual(name, []byte("rcx")):
		return codeFromBase(1, 64)
	case bytesEqual(name, []byte("rdx")):
		return codeFromBase(2, 64)
	case bytesEqual(name, []byte("rbx")):
		return codeFromBase(3, 64)
	case bytesEqual(name, []byte("rsp")):
		return codeFromBase(4, 64)
	case bytesEqual(name, []byte("rbp")):
		return codeFromBase(5, 64)
	case bytesEqual(name, []byte("rsi")):
		return codeFromBase(6, 64)
	case bytesEqual(name, []byte("rdi")):
		return codeFromBase(7, 64)
	}

	n := len(name)
	if n >= 2 && name[0] == 'r' {
		if name[1] >= '0' && name[1] <= '9' {
			idx := int(name[1] - '0')
			pos := 2
			if idx >= 10 {
				return 0, 0, false
			}
			var width int
			if pos < n {
				suf := name[pos:]
				_ = suf
			}
			if n == 2 {
				width = 64
				if idx < 8 {
					return 0, 0, false
				}
				return codeFromBase(byte(idx), width)
			}
			if n == 3 {
				suf := name[2]
				if idx < 8 {
					return 0, 0, false
				}
				switch suf {
				case 'b':
					width = 8
				case 'w':
					width = 16
				case 'd':
					width = 32
				default:
					return 0, 0, false
				}
				return codeFromBase(byte(idx), width)
			}
			if n == 4 {
				if idx != 1 {
				}
			}
		}
	}

	if len(name) == 3 && name[0] == 'r' && name[1] == '1' && name[2] >= '0' && name[2] <= '5' {
	}
	if (len(name) == 2 || len(name) == 3 || len(name) == 4) && name[0] == 'r' && name[1] == '1' {
		idx := int((name[2] - '0') + 10)
		if idx < 10 || idx > 15 {
			return 0, 0, false
		}
		var width int
		width = 64
		if len(name) == 4 {
			suf := name[3]
			switch suf {
			case 'b':
				width = 8
			case 'w':
				width = 16
			case 'd':
				width = 32
			default:
				return 0, 0, false
			}
		}
		return codeFromBase(byte(idx), width)
	}
	if bytesHasPrefix(name, []byte("r8")) && (len(name) == 3 && (name[2] == 'b' || name[2] == 'w' || name[2] == 'd')) {

	}
	return 0, 0, false
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func bytesHasPrefix(a, prefix []byte) bool {
	if len(prefix) > len(a) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if a[i] != prefix[i] {
			return false
		}
	}
	return true
}

type operandType int

const (
	opReg operandType = iota
	opImm
	opMem
	opLabel
)

type operand struct {
	typ   operandType
	reg   byte
	imm   uint64
	disp  int64
	base  byte
	index byte
	scale byte
	label []byte
}

func parseOperand(tok []byte) (operand, error) {
	tok = trimSpace(tok)
	if len(tok) == 0 {
		return operand{}, errors.New("empty operand")
	}
	if code, _, ok := parseRegister(tok); ok {
		return operand{typ: opReg, reg: code}, nil
	}
	if len(tok) > 2 && tok[0] == '[' && tok[len(tok)-1] == ']' {
		inner := trimSpace(tok[1 : len(tok)-1])
		if len(inner) == 0 {
			return operand{}, errors.New("empty memory operand")
		}
		var base byte = 255
		var index byte = 255
		var scale byte = 1
		var disp int64 = 0

		i := 0
		for i < len(inner) {
			for i < len(inner) && (inner[i] == ' ' || inner[i] == '\t') {
				i++
			}
			if i >= len(inner) {
				break
			}
			sign := int64(1)
			if inner[i] == '+' || inner[i] == '-' {
				if inner[i] == '-' {
					sign = -1
				}
				i++
				for i < len(inner) && (inner[i] == ' ' || inner[i] == '\t') {
					i++
				}
				if i >= len(inner) {
					return operand{}, errors.New("expected number after sign")
				}
			}
			start := i
			for i < len(inner) && inner[i] != '+' && inner[i] != '-' {
				i++
			}
			part := trimSpace(inner[start:i])
			if len(part) == 0 {
				continue
			}
			if num, err := parseNumber(part); err == nil {
				disp += sign * int64(num)
				continue
			}
			if code, _, ok := parseRegister(part); ok {
				if base == 255 {
					base = code
				} else if index == 255 {
					index = code
				} else {
					return operand{}, errors.New("too many registers")
				}
				continue
			}
			if idx := bytes.IndexByte(part, '*'); idx != -1 {
				regPart := trimSpace(part[:idx])
				scalePart := trimSpace(part[idx+1:])
				code, _, ok := parseRegister(regPart)
				if !ok {
					return operand{}, errors.New("invalid index register")
				}
				scaleVal, err := parseNumber(scalePart)
				if err != nil || (scaleVal != 1 && scaleVal != 2 && scaleVal != 4 && scaleVal != 8) {
					return operand{}, errors.New("invalid scale")
				}
				if index != 255 {
					return operand{}, errors.New("index already set")
				}
				index = code
				scale = byte(scaleVal)
				continue
			}
			return operand{}, errors.New("unsupported memory operand format")
		}
		if base == 255 && index == 255 {
			return operand{}, errors.New("memory operand must have at least base or index")
		}
		if base == 255 {
			base = 5
		}
		return operand{typ: opMem, base: base, index: index, scale: scale, disp: disp}, nil
	}
	if num, err := parseNumber(tok); err == nil {
		return operand{typ: opImm, imm: num}, nil
	}
	return operand{typ: opLabel, label: tok}, nil
}

func encodeModRM(mod byte, reg byte, rm byte) byte {
	return (mod << 6) | ((reg & 7) << 3) | (rm & 7)
}

func encodeSIB(scale byte, index byte, base byte) byte {
	return ((scale & 3) << 6) | ((index & 7) << 3) | (base & 7)
}

func (p *parser) writeMemOperand(mem operand, reg byte, isLea bool) {
	base := mem.base
	index := mem.index
	scale := mem.scale
	dispVal := mem.disp

	if index != 255 {
		if dispVal == 0 {
			p.current.AppendBytes([]byte{encodeModRM(0, reg, 4), encodeSIB(scale, index, base)})
			return
		}
		if dispVal >= -128 && dispVal <= 127 {
			p.current.AppendBytes([]byte{encodeModRM(1, reg, 4), encodeSIB(scale, index, base), byte(dispVal)})
			return
		}
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(dispVal))
		p.current.AppendBytes([]byte{encodeModRM(2, reg, 4), encodeSIB(scale, index, base)})
		p.current.AppendBytes(buf[:])
		return
	}
	if base == 255 {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(dispVal))
		p.current.AppendByte(encodeModRM(0, reg, 5))
		p.current.AppendBytes(buf[:])
		return
	}
	if dispVal == 0 {
		p.current.AppendByte(encodeModRM(0, reg, base))
		return
	}
	if dispVal >= -128 && dispVal <= 127 {
		p.current.AppendBytes([]byte{encodeModRM(1, reg, base), byte(dispVal)})
		return
	}
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(dispVal))
	p.current.AppendByte(encodeModRM(2, reg, base))
	p.current.AppendBytes(buf[:])
}

func (p *parser) emitArith(opRegReg byte, opRegImm byte, rest []byte) error {
	args := p.splitArgs(trimSpace(rest))
	if len(args) != 2 {
		return errors.New("invalid operands")
	}
	dst, err := parseOperand(args[0])
	if err != nil {
		return err
	}
	src, err := parseOperand(args[1])
	if err != nil {
		return err
	}
	if dst.typ == opReg && src.typ == opReg {
		p.current.AppendBytes([]byte{0x48, opRegReg, encodeModRM(3, src.reg, dst.reg)})
		return nil
	}
	if dst.typ == opReg && src.typ == opImm {
		p.current.AppendBytes([]byte{0x48, opRegImm, encodeModRM(3, 0, dst.reg)})
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(src.imm))
		p.current.AppendBytes(buf[:])
		return nil
	}
	if dst.typ == opMem && src.typ == opReg {
		p.current.AppendBytes([]byte{0x48, opRegReg})
		p.writeMemOperand(dst, src.reg, false)
		return nil
	}
	if dst.typ == opReg && src.typ == opMem {
		p.current.AppendBytes([]byte{0x48, opRegReg})
		p.writeMemOperand(src, dst.reg, false)
		return nil
	}
	if dst.typ == opMem && src.typ == opImm {
		p.current.AppendBytes([]byte{0x48, opRegImm})
		p.writeMemOperand(dst, 0, false)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(src.imm))
		p.current.AppendBytes(buf[:])
		return nil
	}
	return errors.New("unsupported operand combination for arithmetic")
}

func (p *parser) emitInc(rest []byte) error {
	tok := trimSpace(rest)
	op, err := parseOperand(tok)
	if err != nil {
		return err
	}
	if op.typ == opReg {
		p.current.AppendBytes([]byte{0x48, 0xFF, encodeModRM(3, 0, op.reg)})
		return nil
	}
	if op.typ == opMem {
		p.current.AppendBytes([]byte{0x48, 0xFF})
		p.writeMemOperand(op, 0, false)
		return nil
	}
	return errors.New("invalid inc operand")
}

func (p *parser) emitDec(rest []byte) error {
	tok := trimSpace(rest)
	op, err := parseOperand(tok)
	if err != nil {
		return err
	}
	if op.typ == opReg {
		p.current.AppendBytes([]byte{0x48, 0xFF, encodeModRM(3, 1, op.reg)})
		return nil
	}
	if op.typ == opMem {
		p.current.AppendBytes([]byte{0x48, 0xFF})
		p.writeMemOperand(op, 1, false)
		return nil
	}
	return errors.New("invalid dec operand")
}

func (p *parser) emitUnary(opcode byte, ext byte, rest []byte) error {
	tok := trimSpace(rest)
	op, err := parseOperand(tok)
	if err != nil {
		return err
	}
	if op.typ == opReg {
		p.current.AppendBytes([]byte{0x48, opcode, encodeModRM(3, ext, op.reg)})
		return nil
	}
	if op.typ == opMem {
		p.current.AppendBytes([]byte{0x48, opcode})
		p.writeMemOperand(op, ext, false)
		return nil
	}
	return errors.New("invalid unary operand")
}

func (p *parser) emitPush(rest []byte) error {
	tok := trimSpace(rest)
	op, err := parseOperand(tok)
	if err != nil {
		return err
	}
	if op.typ == opReg {
		p.current.AppendByte(0x50 + op.reg)
		return nil
	}
	if op.typ == opImm {
		p.current.AppendByte(0x68)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(op.imm))
		p.current.AppendBytes(buf[:])
		return nil
	}
	if op.typ == opMem {
		p.current.AppendByte(0xFF)
		p.writeMemOperand(op, 6, false)
		return nil
	}
	return errors.New("invalid push operand")
}

func (p *parser) emitPop(rest []byte) error {
	tok := trimSpace(rest)
	op, err := parseOperand(tok)
	if err != nil {
		return err
	}
	if op.typ == opReg {
		p.current.AppendByte(0x58 + op.reg)
		return nil
	}
	if op.typ == opMem {
		p.current.AppendByte(0x8F)
		p.writeMemOperand(op, 0, false)
		return nil
	}
	return errors.New("invalid pop operand")
}

func (p *parser) emitLea(rest []byte) error {
	args := p.splitArgs(trimSpace(rest))
	if len(args) != 2 {
		return errors.New("invalid lea operands")
	}
	dst, err := parseOperand(args[0])
	if err != nil {
		return err
	}
	src, err := parseOperand(args[1])
	if err != nil {
		return err
	}
	if dst.typ != opReg || src.typ != opMem {
		return errors.New("lea requires register destination and memory source")
	}
	p.current.data = append(p.current.data, 0x48, 0x8D)
	p.writeMemOperand(src, dst.reg, true)
	return nil
}

func (p *parser) emitJmp(rest []byte) error {
	tok := trimSpace(rest)
	op, err := parseOperand(tok)
	if err != nil {
		return err
	}
	if op.typ == opLabel {
		name := op.label
		if err := p.addSymbol(name, 0, shnUnDef, stBindLocal); err != nil {
			return err
		}
		symIdx := p.findSymbol(name)
		cur := len(p.current.data)
		p.current.data = append(p.current.data, 0xE9, 0, 0, 0, 0)
		p.relocs = append(p.relocs, struct {
			sec    uint16
			off    uint64
			sym    int
			typ    uint32
			addend int64
		}{sec: sectionIndex(p.current), off: uint64(cur + 1), sym: symIdx, typ: rX86_64_PC32, addend: -4})
		return nil
	}
	if op.typ == opReg {
		p.current.data = append(p.current.data, 0xFF, 0xE0+op.reg)
		return nil
	}
	if op.typ == opMem {
		p.current.data = append(p.current.data, 0xFF)
		p.writeMemOperand(op, 4, false)
		return nil
	}
	return errors.New("invalid jmp operand")
}

func (p *parser) emitJcc(opcode byte, rest []byte) error {
	tok := trimSpace(rest)
	op, err := parseOperand(tok)
	if err != nil {
		return err
	}
	if op.typ != opLabel {
		return errors.New("jcc requires label")
	}
	name := op.label
	if err := p.addSymbol(name, 0, shnUnDef, stBindLocal); err != nil {
		return err
	}
	symIdx := p.findSymbol(name)
	cur := len(p.current.data)
	p.current.data = append(p.current.data, 0x0F, opcode, 0, 0, 0, 0)
	p.relocs = append(p.relocs, struct {
		sec    uint16
		off    uint64
		sym    int
		typ    uint32
		addend int64
	}{sec: sectionIndex(p.current), off: uint64(cur + 2), sym: symIdx, typ: rX86_64_PC32, addend: -4})
	return nil
}

func (p *parser) emitTest(rest []byte) error {
	args := p.splitArgs(trimSpace(rest))
	if len(args) != 2 {
		return errors.New("invalid test operands")
	}
	dst, err := parseOperand(args[0])
	if err != nil {
		return err
	}
	src, err := parseOperand(args[1])
	if err != nil {
		return err
	}
	if dst.typ == opReg && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, 0x85, encodeModRM(3, src.reg, dst.reg))
		return nil
	}
	if dst.typ == opReg && src.typ == opImm {
		p.current.data = append(p.current.data, 0x48, 0xF7, encodeModRM(3, 0, dst.reg))
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(src.imm))
		p.current.data = append(p.current.data, buf[:]...)
		return nil
	}
	if dst.typ == opMem && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, 0x85)
		p.writeMemOperand(dst, src.reg, false)
		return nil
	}
	return errors.New("unsupported test operands")
}

func (p *parser) emitShift(opcode byte, ext byte, rest []byte) error {
	args := p.splitArgs(trimSpace(rest))
	if len(args) != 2 {
		return errors.New("invalid shift operands")
	}
	dst, err := parseOperand(args[0])
	if err != nil {
		return err
	}
	src, err := parseOperand(args[1])
	if err != nil {
		return err
	}
	if dst.typ == opReg && src.typ == opImm {
		cnt := byte(1)
		if src.imm != 1 {
			cnt = byte(src.imm & 0xFF)
		}
		p.current.data = append(p.current.data, 0x48, opcode, encodeModRM(3, ext, dst.reg), cnt)
		return nil
	}
	if dst.typ == opMem && src.typ == opImm {
		cnt := byte(1)
		if src.imm != 1 {
			cnt = byte(src.imm & 0xFF)
		}
		p.current.data = append(p.current.data, 0x48, opcode)
		p.writeMemOperand(dst, ext, false)
		p.current.data = append(p.current.data, cnt)
		return nil
	}
	return errors.New("unsupported shift operands")
}

func (p *parser) emitMulDiv(opcode byte, ext byte, rest []byte) error {
	tok := trimSpace(rest)
	op, err := parseOperand(tok)
	if err != nil {
		return err
	}
	if op.typ == opReg {
		p.current.data = append(p.current.data, 0x48, opcode, encodeModRM(3, ext, op.reg))
		return nil
	}
	if op.typ == opMem {
		p.current.data = append(p.current.data, 0x48, opcode)
		p.writeMemOperand(op, ext, false)
		return nil
	}
	return errors.New("invalid mul/div operand")
}

func (p *parser) emitMov(rest []byte) error {
	args := p.splitArgs(trimSpace(rest))
	if len(args) != 2 {
		return errors.New("invalid mov operands")
	}
	dst, err := parseOperand(args[0])
	if err != nil {
		return err
	}
	src, err := parseOperand(args[1])
	if err != nil {
		return err
	}
	if dst.typ == opReg && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, 0x89, encodeModRM(3, src.reg, dst.reg))
		return nil
	}
	if dst.typ == opReg && src.typ == opImm {
		reg := dst.reg
		if src.imm > 0xFFFFFFFF {
			p.current.data = append(p.current.data, 0x48, 0xB8+reg)
			var buf [8]byte
			binary.LittleEndian.PutUint64(buf[:], src.imm)
			p.current.data = append(p.current.data, buf[:]...)
		} else {
			p.current.data = append(p.current.data, 0xB8+reg)
			var buf [4]byte
			binary.LittleEndian.PutUint32(buf[:], uint32(src.imm))
			p.current.data = append(p.current.data, buf[:]...)
		}
		return nil
	}
	if dst.typ == opMem && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, 0x89)
		p.writeMemOperand(dst, src.reg, false)
		return nil
	}
	if dst.typ == opReg && src.typ == opMem {
		p.current.data = append(p.current.data, 0x48, 0x8B)
		p.writeMemOperand(src, dst.reg, false)
		return nil
	}
	if dst.typ == opReg && src.typ == opLabel {
		name := src.label
		if err := p.addSymbol(name, 0, shnUnDef, stBindLocal); err != nil {
			return err
		}
		symIdx := p.findSymbol(name)
		cur := len(p.current.data)
		p.current.data = append(p.current.data, 0x48, 0xB8+dst.reg)
		var z [8]byte
		p.current.data = append(p.current.data, z[:]...)
		p.relocs = append(p.relocs, struct {
			sec    uint16
			off    uint64
			sym    int
			typ    uint32
			addend int64
		}{sec: sectionIndex(p.current), off: uint64(cur + 2), sym: symIdx, typ: rX86_64_64, addend: 0})
		return nil
	}
	return errors.New("unsupported mov operands")
}

func (p *parser) emitMovzx(rest []byte) error {
	args := p.splitArgs(trimSpace(rest))
	if len(args) != 2 {
		return errors.New("invalid movzx operands")
	}
	dst, err := parseOperand(args[0])
	if err != nil {
		return err
	}
	src, err := parseOperand(args[1])
	if err != nil {
		return err
	}
	if dst.typ != opReg || src.typ != opMem {
		return errors.New("movzx requires register destination and memory source")
	}
	p.current.data = append(p.current.data, 0x0F, 0xB6)
	p.writeMemOperand(src, dst.reg, false)
	return nil
}

func (p *parser) emitMovsx(rest []byte) error {
	args := p.splitArgs(trimSpace(rest))
	if len(args) != 2 {
		return errors.New("invalid movsx operands")
	}
	dst, err := parseOperand(args[0])
	if err != nil {
		return err
	}
	src, err := parseOperand(args[1])
	if err != nil {
		return err
	}
	if dst.typ != opReg || src.typ != opMem {
		return errors.New("movsx requires register destination and memory source")
	}
	p.current.data = append(p.current.data, 0x0F, 0xBE)
	p.writeMemOperand(src, dst.reg, false)
	return nil
}

func (p *parser) emitXchg(rest []byte) error {
	args := p.splitArgs(trimSpace(rest))
	if len(args) != 2 {
		return errors.New("invalid xchg operands")
	}
	dst, err := parseOperand(args[0])
	if err != nil {
		return err
	}
	src, err := parseOperand(args[1])
	if err != nil {
		return err
	}
	if dst.typ == opReg && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, 0x87, encodeModRM(3, src.reg, dst.reg))
		return nil
	}
	if dst.typ == opMem && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, 0x87)
		p.writeMemOperand(dst, src.reg, false)
		return nil
	}
	if dst.typ == opReg && src.typ == opMem {
		p.current.data = append(p.current.data, 0x48, 0x87)
		p.writeMemOperand(src, dst.reg, false)
		return nil
	}
	return errors.New("unsupported xchg operands")
}

func (p *parser) emitCall(rest []byte) error {
	name := trimSpace(rest)
	if len(name) == 0 {
		return errors.New("invalid call target")
	}
	if err := p.addSymbol(name, 0, shnUnDef, stBindLocal); err != nil {
		return err
	}
	symIdx := p.findSymbol(name)
	cur := len(p.current.data)
	p.current.data = append(p.current.data, 0xE8, 0, 0, 0, 0)
	p.relocs = append(p.relocs, struct {
		sec    uint16
		off    uint64
		sym    int
		typ    uint32
		addend int64
	}{sec: sectionIndex(p.current), off: uint64(cur + 1), sym: symIdx, typ: rX86_64_PC32, addend: -4})
	return nil
}

func (p *parser) emitImul(rest []byte) error {
	args := p.splitArgs(trimSpace(rest))
	if len(args) != 2 {
		return errors.New("invalid imul operands")
	}
	dst, err := parseOperand(args[0])
	if err != nil {
		return err
	}
	src, err := parseOperand(args[1])
	if err != nil {
		return err
	}
	if dst.typ == opReg && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, 0x0F, 0xAF, encodeModRM(3, src.reg, dst.reg))
		return nil
	}
	if dst.typ == opReg && src.typ == opMem {
		p.current.data = append(p.current.data, 0x48, 0x0F, 0xAF)
		p.writeMemOperand(src, dst.reg, false)
		return nil
	}
	return errors.New("unsupported imul operands")
}

func (p *parser) emitCmp(rest []byte) error {
	args := p.splitArgs(trimSpace(rest))
	if len(args) != 2 {
		return errors.New("invalid cmp operands")
	}
	dst, err := parseOperand(args[0])
	if err != nil {
		return err
	}
	src, err := parseOperand(args[1])
	if err != nil {
		return err
	}
	if dst.typ == opReg && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, 0x39, encodeModRM(3, src.reg, dst.reg))
		return nil
	}
	if dst.typ == opReg && src.typ == opImm {
		p.current.data = append(p.current.data, 0x48, 0x81, encodeModRM(3, 7, dst.reg))
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(src.imm))
		p.current.data = append(p.current.data, buf[:]...)
		return nil
	}
	if dst.typ == opMem && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, 0x39)
		p.writeMemOperand(dst, src.reg, false)
		return nil
	}
	if dst.typ == opMem && src.typ == opImm {
		p.current.data = append(p.current.data, 0x48, 0x81)
		p.writeMemOperand(dst, 7, false)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(src.imm))
		p.current.data = append(p.current.data, buf[:]...)
		return nil
	}
	return errors.New("unsupported cmp operands")
}

func (p *parser) emitJump(opcode byte, rest []byte) error {
	name := trimSpace(rest)
	if len(name) == 0 {
		return errors.New("invalid jump target")
	}
	if err := p.addSymbol(name, 0, shnUnDef, stBindLocal); err != nil {
		return err
	}
	symIdx := p.findSymbol(name)
	cur := len(p.current.data)
	p.current.data = append(p.current.data, 0x0F, opcode, 0, 0, 0, 0)
	p.relocs = append(p.relocs, struct {
		sec    uint16
		off    uint64
		sym    int
		typ    uint32
		addend int64
	}{sec: sectionIndex(p.current), off: uint64(cur + 2), sym: symIdx, typ: rX86_64_PC32, addend: -4})
	return nil
}
