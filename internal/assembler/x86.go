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
	code  byte
	width int
}

func parseRegister(tok []byte) (registerInfo, bool) {
	if len(tok) == 0 || len(tok) > 4 {
		return registerInfo{}, false
	}
	var key uint32
	for i, b := range tok {
		key |= uint32(b) << (8 * i)
	}
	switch key {
	case uint32('a') | uint32('l')<<8:
		return registerInfo{code: 0, width: 8}, true
	case uint32('c') | uint32('l')<<8:
		return registerInfo{code: 1, width: 8}, true
	case uint32('d') | uint32('l')<<8:
		return registerInfo{code: 2, width: 8}, true
	case uint32('b') | uint32('l')<<8:
		return registerInfo{code: 3, width: 8}, true
	case uint32('a') | uint32('h')<<8:
		return registerInfo{code: 4, width: 8}, true
	case uint32('c') | uint32('h')<<8:
		return registerInfo{code: 5, width: 8}, true
	case uint32('d') | uint32('h')<<8:
		return registerInfo{code: 6, width: 8}, true
	case uint32('b') | uint32('h')<<8:
		return registerInfo{code: 7, width: 8}, true
	case uint32('a') | uint32('x')<<8:
		return registerInfo{code: 0, width: 16}, true
	case uint32('c') | uint32('x')<<8:
		return registerInfo{code: 1, width: 16}, true
	case uint32('d') | uint32('x')<<8:
		return registerInfo{code: 2, width: 16}, true
	case uint32('b') | uint32('x')<<8:
		return registerInfo{code: 3, width: 16}, true
	case uint32('s') | uint32('p')<<8:
		return registerInfo{code: 4, width: 16}, true
	case uint32('b') | uint32('p')<<8:
		return registerInfo{code: 5, width: 16}, true
	case uint32('s') | uint32('i')<<8:
		return registerInfo{code: 6, width: 16}, true
	case uint32('d') | uint32('i')<<8:
		return registerInfo{code: 7, width: 16}, true
	case uint32('r') | uint32('8')<<8:
		return registerInfo{code: 8, width: 64}, true
	case uint32('r') | uint32('9')<<8:
		return registerInfo{code: 9, width: 64}, true
	case uint32('r') | uint32('1')<<8 | uint32('0')<<16:
		return registerInfo{code: 10, width: 64}, true
	case uint32('r') | uint32('1')<<8 | uint32('1')<<16:
		return registerInfo{code: 11, width: 64}, true
	case uint32('r') | uint32('1')<<8 | uint32('2')<<16:
		return registerInfo{code: 12, width: 64}, true
	case uint32('r') | uint32('1')<<8 | uint32('3')<<16:
		return registerInfo{code: 13, width: 64}, true
	case uint32('r') | uint32('1')<<8 | uint32('4')<<16:
		return registerInfo{code: 14, width: 64}, true
	case uint32('r') | uint32('1')<<8 | uint32('5')<<16:
		return registerInfo{code: 15, width: 64}, true
	case uint32('r') | uint32('a')<<8 | uint32('x')<<16:
		return registerInfo{code: 0, width: 64}, true
	case uint32('r') | uint32('c')<<8 | uint32('x')<<16:
		return registerInfo{code: 1, width: 64}, true
	case uint32('r') | uint32('d')<<8 | uint32('x')<<16:
		return registerInfo{code: 2, width: 64}, true
	case uint32('r') | uint32('b')<<8 | uint32('x')<<16:
		return registerInfo{code: 3, width: 64}, true
	case uint32('r') | uint32('s')<<8 | uint32('p')<<16:
		return registerInfo{code: 4, width: 64}, true
	case uint32('r') | uint32('b')<<8 | uint32('p')<<16:
		return registerInfo{code: 5, width: 64}, true
	case uint32('r') | uint32('s')<<8 | uint32('i')<<16:
		return registerInfo{code: 6, width: 64}, true
	case uint32('r') | uint32('d')<<8 | uint32('i')<<16:
		return registerInfo{code: 7, width: 64}, true
	case uint32('e') | uint32('a')<<8 | uint32('x')<<16:
		return registerInfo{code: 0, width: 32}, true
	case uint32('e') | uint32('c')<<8 | uint32('x')<<16:
		return registerInfo{code: 1, width: 32}, true
	case uint32('e') | uint32('d')<<8 | uint32('x')<<16:
		return registerInfo{code: 2, width: 32}, true
	case uint32('e') | uint32('b')<<8 | uint32('x')<<16:
		return registerInfo{code: 3, width: 32}, true
	case uint32('e') | uint32('s')<<8 | uint32('p')<<16:
		return registerInfo{code: 4, width: 32}, true
	case uint32('e') | uint32('b')<<8 | uint32('p')<<16:
		return registerInfo{code: 5, width: 32}, true
	case uint32('e') | uint32('s')<<8 | uint32('i')<<16:
		return registerInfo{code: 6, width: 32}, true
	case uint32('e') | uint32('d')<<8 | uint32('i')<<16:
		return registerInfo{code: 7, width: 32}, true
	case uint32('r') | uint32('8')<<8 | uint32('d')<<16:
		return registerInfo{code: 8, width: 32}, true
	case uint32('r') | uint32('9')<<8 | uint32('d')<<16:
		return registerInfo{code: 9, width: 32}, true
	case uint32('r') | uint32('1')<<8 | uint32('0')<<16 | uint32('d')<<24:
		return registerInfo{code: 10, width: 32}, true
	case uint32('r') | uint32('1')<<8 | uint32('1')<<16 | uint32('d')<<24:
		return registerInfo{code: 11, width: 32}, true
	case uint32('r') | uint32('1')<<8 | uint32('2')<<16 | uint32('d')<<24:
		return registerInfo{code: 12, width: 32}, true
	case uint32('r') | uint32('1')<<8 | uint32('3')<<16 | uint32('d')<<24:
		return registerInfo{code: 13, width: 32}, true
	case uint32('r') | uint32('1')<<8 | uint32('4')<<16 | uint32('d')<<24:
		return registerInfo{code: 14, width: 32}, true
	case uint32('r') | uint32('1')<<8 | uint32('5')<<16 | uint32('d')<<24:
		return registerInfo{code: 15, width: 32}, true
	}
	return registerInfo{}, false
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
	if reg, ok := parseRegister(tok); ok {
		return operand{typ: opReg, reg: reg.code}, nil
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
			if reg, ok := parseRegister(part); ok {
				if base == 255 {
					base = reg.code
				} else if index == 255 {
					index = reg.code
				} else {
					return operand{}, errors.New("too many registers")
				}
				continue
			}
			if idx := bytes.IndexByte(part, '*'); idx != -1 {
				regPart := trimSpace(part[:idx])
				scalePart := trimSpace(part[idx+1:])
				reg, ok := parseRegister(regPart)
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
				index = reg.code
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
			p.current.data = append(p.current.data, encodeModRM(0, reg, 4), encodeSIB(scale, index, base))
			return
		}
		if dispVal >= -128 && dispVal <= 127 {
			p.current.data = append(p.current.data, encodeModRM(1, reg, 4), encodeSIB(scale, index, base), byte(dispVal))
			return
		}
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(dispVal))
		p.current.data = append(p.current.data, encodeModRM(2, reg, 4), encodeSIB(scale, index, base))
		p.current.data = append(p.current.data, buf[:]...)
		return
	}
	if base == 255 {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(dispVal))
		p.current.data = append(p.current.data, encodeModRM(0, reg, 5))
		p.current.data = append(p.current.data, buf[:]...)
		return
	}
	if dispVal == 0 {
		p.current.data = append(p.current.data, encodeModRM(0, reg, base))
		return
	}
	if dispVal >= -128 && dispVal <= 127 {
		p.current.data = append(p.current.data, encodeModRM(1, reg, base), byte(dispVal))
		return
	}
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(dispVal))
	p.current.data = append(p.current.data, encodeModRM(2, reg, base))
	p.current.data = append(p.current.data, buf[:]...)
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
		p.current.data = append(p.current.data, 0x48, opRegReg, encodeModRM(3, src.reg, dst.reg))
		return nil
	}
	if dst.typ == opReg && src.typ == opImm {
		p.current.data = append(p.current.data, 0x48, opRegImm, encodeModRM(3, 0, dst.reg))
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(src.imm))
		p.current.data = append(p.current.data, buf[:]...)
		return nil
	}
	if dst.typ == opMem && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, opRegReg)
		p.writeMemOperand(dst, src.reg, false)
		return nil
	}
	if dst.typ == opReg && src.typ == opMem {
		p.current.data = append(p.current.data, 0x48, opRegReg)
		p.writeMemOperand(src, dst.reg, false)
		return nil
	}
	if dst.typ == opMem && src.typ == opImm {
		p.current.data = append(p.current.data, 0x48, opRegImm)
		p.writeMemOperand(dst, 0, false)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(src.imm))
		p.current.data = append(p.current.data, buf[:]...)
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
		p.current.data = append(p.current.data, 0x48, 0xFF, encodeModRM(3, 0, op.reg))
		return nil
	}
	if op.typ == opMem {
		p.current.data = append(p.current.data, 0x48, 0xFF)
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
		p.current.data = append(p.current.data, 0x48, 0xFF, encodeModRM(3, 1, op.reg))
		return nil
	}
	if op.typ == opMem {
		p.current.data = append(p.current.data, 0x48, 0xFF)
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
		p.current.data = append(p.current.data, 0x48, opcode, encodeModRM(3, ext, op.reg))
		return nil
	}
	if op.typ == opMem {
		p.current.data = append(p.current.data, 0x48, opcode)
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
		p.current.data = append(p.current.data, 0x50+op.reg)
		return nil
	}
	if op.typ == opImm {
		p.current.data = append(p.current.data, 0x68)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(op.imm))
		p.current.data = append(p.current.data, buf[:]...)
		return nil
	}
	if op.typ == opMem {
		p.current.data = append(p.current.data, 0xFF)
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
		p.current.data = append(p.current.data, 0x58+op.reg)
		return nil
	}
	if op.typ == opMem {
		p.current.data = append(p.current.data, 0x8F)
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
		_ = p.addSymbol(name, 0, shnUnDef, stBindLocal)
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
	_ = p.addSymbol(name, 0, shnUnDef, stBindLocal)
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
		_ = p.addSymbol(name, 0, shnUnDef, stBindLocal)
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
	_ = p.addSymbol(name, 0, shnUnDef, stBindLocal)
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
	_ = p.addSymbol(name, 0, shnUnDef, stBindLocal)
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