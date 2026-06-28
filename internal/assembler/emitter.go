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
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"
)

const (
	elfClass32        = 1
	elfClass64        = 2
	elfData2LSB       = 1
	elfVersion1       = 1
	elfTypeRel        = 1
	elfMachine386     = 3
	elfMachineX86_64  = 62
	elfMachineARM     = 40
	elfMachineAARCH64 = 183
	elfMachineRISCV   = 243
	rX86_64_64        = 1
	rX86_64_PC32      = 2
	shTypeNull        = 0
	shTypeProgBits    = 1
	shTypeSymTab      = 2
	shTypeStrTab      = 3
	shTypeRela        = 4
	shTypeNoBits      = 8
	shFlagAlloc       = 0x2
	shFlagExecInstr   = 0x4
	shFlagWrite       = 0x1
	stBindLocal       = 0
	stBindGlobal      = 1
	stTypeNotype      = 0
	shnUnDef          = 0
	elf32ShdrSize     = 40
	elf64ShdrSize     = 64
)

var parserPool = sync.Pool{
	New: func() any {
		p := &parser{}
		p.text.data = make([]byte, 0, 4096)
		p.data.data = make([]byte, 0, 1024)
		p.bss.data = make([]byte, 0, 1024)
		p.relocs = make([]struct {
			sec    uint16
			off    uint64
			sym    int
			typ    uint32
			addend int64
		}, 0, 32)
		p.argsPool = make([][]byte, 0, 16)
		return p
	},
}

var emitterBufferPool = sync.Pool{New: func() any { b := make([]byte, 0, 65536); return &b }}

var (
	nameEmpty    = []byte("")
	nameShstrtab = []byte(".shstrtab")
	nameSymtab   = []byte(".symtab")
	nameStrtab   = []byte(".strtab")
	reusableOut  *[]byte
)

func assembleBareMetalObject(ctx context.Context, src, obj string) error {
	srcBuf, release, err := loadSourcePooled(src)
	if err != nil {
		return err
	}
	defer release()
	profile := selfTargetProfile(Target)
	out, err := emitSourceObject(srcBuf, profile)
	if err != nil {
		return err
	}
	if len(out) == 0 {
		return errors.New("assembler produced empty object: " + src)
	}
	return os.WriteFile(obj, out, 0o644)
}

type TargetProfile = targetEmitterProfile

func TargetProfileFromTarget(target string) TargetProfile {
	return selfTargetProfile(target)
}

func EmitSourceObject(src []byte, profile TargetProfile) ([]byte, error) {
	return emitSourceObject(src, profile)
}

func loadSourcePooled(sourcePath string) ([]byte, func(), error) {
	p := emitterBufferPool.Get().(*[]byte)
	buf := *p
	if cap(buf) < 65536 {
		buf = make([]byte, 0, 65536)
	}
	buf = buf[:cap(buf)]
	f, err := os.Open(sourcePath)
	if err != nil {
		emitterBufferPool.Put(p)
		return nil, nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		emitterBufferPool.Put(p)
		return nil, nil, err
	}
	if info.Size() > int64(cap(buf)) {
		emitterBufferPool.Put(p)
		return nil, nil, errors.New("source too large for internal emitter")
	}
	n, err := io.ReadFull(f, buf[:info.Size()])
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		emitterBufferPool.Put(p)
		return nil, nil, err
	}
	buf = buf[:n]
	*p = buf
	return buf, func() {
		*p = (*p)[:0]
		emitterBufferPool.Put(p)
	}, nil
}

type targetEmitterProfile struct {
	elfClass    uint8
	pointerSize uint32
	align       uint32
	eMachine    uint16
}

func selfTargetProfile(target string) targetEmitterProfile {
	if containsFold(target, "x86_64") {
		return targetEmitterProfile{elfClass: elfClass64, pointerSize: 8, align: 16, eMachine: elfMachineX86_64}
	}
	if containsFold(target, "aarch64") {
		return targetEmitterProfile{elfClass: elfClass64, pointerSize: 8, align: 16, eMachine: elfMachineAARCH64}
	}
	if containsFold(target, "riscv") {
		return targetEmitterProfile{elfClass: elfClass64, pointerSize: 8, align: 4, eMachine: elfMachineRISCV}
	}
	if containsFold(target, "arm") || containsFold(target, "cortex-") {
		return targetEmitterProfile{elfClass: elfClass32, pointerSize: 4, align: 4, eMachine: elfMachineARM}
	}
	if containsFold(target, "i386") || containsFold(target, "i686") {
		return targetEmitterProfile{elfClass: elfClass32, pointerSize: 4, align: 4, eMachine: elfMachine386}
	}
	return targetEmitterProfile{elfClass: elfClass64, pointerSize: 8, align: 16, eMachine: elfMachineX86_64}
}

type sectionState struct {
	name  []byte
	data  []byte
	size  uint64
	flags uint32
	align uint32
}

type symbolState struct {
	name       []byte
	nameOffset uint32
	value      uint64
	shndx      uint16
	bind       byte
	typ        byte
}

type parser struct {
	src      []byte
	pos      int
	text     sectionState
	data     sectionState
	bss      sectionState
	current  *sectionState
	symbols  [4096]symbolState
	symCount int
	relocs   []struct {
		sec    uint16
		off    uint64
		sym    int
		typ    uint32
		addend int64
	}
	argsPool [][]byte
}

func emitSourceObject(src []byte, profile targetEmitterProfile) ([]byte, error) {
	p := parserPool.Get().(*parser)
	p.reset()
	defer parserPool.Put(p)

	p.text.name = []byte(".text")
	p.text.flags = shFlagAlloc | shFlagExecInstr
	p.text.align = profile.align
	p.data.name = []byte(".data")
	p.data.flags = shFlagAlloc | shFlagWrite
	p.data.align = profile.align
	p.bss.name = []byte(".bss")
	p.bss.flags = shFlagAlloc | shFlagWrite
	p.bss.align = profile.align
	p.current = &p.text
	p.src = src

	for p.pos < len(p.src) {
		line := p.readLine()
		if err := p.parseLine(line); err != nil {
			return nil, err
		}
	}
	if len(p.text.data) == 0 && len(p.data.data) == 0 && p.bss.size == 0 {
		return nil, errors.New("no code generated for source")
	}
	return p.emit(profile)
}

func (p *parser) reset() {
	p.src = nil
	p.pos = 0
	p.symCount = 0
	p.text.data = p.text.data[:0]
	p.text.size = 0
	p.data.data = p.data.data[:0]
	p.data.size = 0
	p.bss.data = p.bss.data[:0]
	p.bss.size = 0
	p.current = &p.text
	p.relocs = p.relocs[:0]
	p.argsPool = p.argsPool[:0]
}

func emitSourceRaw(src []byte, profile targetEmitterProfile) ([]byte, error) {
	p := parserPool.Get().(*parser)
	p.reset()
	defer parserPool.Put(p)

	p.text.name = []byte(".text")
	p.text.flags = shFlagAlloc | shFlagExecInstr
	p.text.align = profile.align
	p.data.name = []byte(".data")
	p.data.flags = shFlagAlloc | shFlagWrite
	p.data.align = profile.align
	p.bss.name = []byte(".bss")
	p.bss.flags = shFlagAlloc | shFlagWrite
	p.bss.align = profile.align
	p.current = &p.text
	p.src = src

	for p.pos < len(p.src) {
		line := p.readLine()
		if err := p.parseLine(line); err != nil {
			return nil, err
		}
	}
	return p.emitRaw(profile)
}

func (p *parser) emitRaw(profile targetEmitterProfile) ([]byte, error) {
	var outPtr *[]byte
	if reusableOut != nil {
		outPtr = reusableOut
	} else {
		outPtr = emitterBufferPool.Get().(*[]byte)
	}
	out := *outPtr
	out = out[:0]
	if len(p.text.data) > 0 {
		out = append(out, p.text.data...)
	}
	if len(p.data.data) > 0 {
		out = append(out, p.data.data...)
	}
	for i := uint64(0); i < p.bss.size; i++ {
		out = append(out, 0)
	}
	*outPtr = out
	if reusableOut == nil {
		emitterBufferPool.Put(outPtr)
	}
	return out, nil
}

func (p *parser) readLine() []byte {
	if p.pos >= len(p.src) {
		return nil
	}
	start := p.pos

	idx := bytes.IndexByte(p.src[p.pos:], '\n')
	if idx == -1 {
		p.pos = len(p.src)
		return p.src[start:]
	}

	p.pos += idx
	line := p.src[start:p.pos]
	p.pos++
	return line
}

func (p *parser) parseLine(line []byte) error {
	line = trimSpace(removeComment(line))
	if len(line) == 0 {
		return nil
	}
	if line[len(line)-1] == ':' {
		name := trimSpace(line[:len(line)-1])
		return p.defineLabel(name)
	}
	tok, rest := readToken(line)
	if len(tok) > 0 && tok[len(tok)-1] == ':' {
		name := tok[:len(tok)-1]
		if err := p.defineLabel(name); err != nil {
			return err
		}
		return p.parseLine(rest)
	}
	switch {
	case equalWord(tok, "section"):
		name, _ := readToken(rest)
		return p.switchSection(name)
	case equalWord(tok, ".text"), equalWord(tok, "text"):
		p.current = &p.text
		return nil
	case equalWord(tok, ".data"), equalWord(tok, "data"):
		p.current = &p.data
		return nil
	case equalWord(tok, ".bss"), equalWord(tok, "bss"):
		p.current = &p.bss
		return nil
	case equalWord(tok, "global"):
		name, _ := readToken(rest)
		return p.defineGlobal(name)
	case equalWord(tok, "db"), equalWord(tok, "byte"):
		return p.emitBytes(rest)
	case equalWord(tok, "dw"), equalWord(tok, "word"):
		return p.emitWords(rest, 2)
	case equalWord(tok, "dd"), equalWord(tok, "dword"):
		return p.emitWords(rest, 4)
	case equalWord(tok, "dq"), equalWord(tok, "qword"):
		return p.emitWords(rest, 8)
	case equalWord(tok, "resb"):
		return p.reserve(rest, 1)
	case equalWord(tok, "resw"):
		return p.reserve(rest, 2)
	case equalWord(tok, "resd"):
		return p.reserve(rest, 4)
	case equalWord(tok, "resq"):
		return p.reserve(rest, 8)
	case equalWord(tok, "align"):
		n, _ := readToken(rest)
		return p.alignSection(n)
	case equalWord(tok, "bits"), equalWord(tok, "org"):
		return nil
	case equalWord(tok, "add"):
		return p.emitArith(0x00, 0x81, rest)
	case equalWord(tok, "sub"):
		return p.emitArith(0x28, 0x81, rest)
	case equalWord(tok, "and"):
		return p.emitArith(0x20, 0x81, rest)
	case equalWord(tok, "or"):
		return p.emitArith(0x08, 0x81, rest)
	case equalWord(tok, "xor"):
		return p.emitArith(0x30, 0x81, rest)
	case equalWord(tok, "adc"):
		return p.emitArith(0x10, 0x81, rest)
	case equalWord(tok, "sbb"):
		return p.emitArith(0x18, 0x81, rest)
	case equalWord(tok, "inc"):
		return p.emitInc(rest)
	case equalWord(tok, "dec"):
		return p.emitDec(rest)
	case equalWord(tok, "neg"):
		return p.emitUnary(0xF7, 3, rest)
	case equalWord(tok, "not"):
		return p.emitUnary(0xF7, 2, rest)
	case equalWord(tok, "push"):
		return p.emitPush(rest)
	case equalWord(tok, "pop"):
		return p.emitPop(rest)
	case equalWord(tok, "pushf"):
		p.current.data = append(p.current.data, 0x9C)
		return nil
	case equalWord(tok, "popf"):
		p.current.data = append(p.current.data, 0x9D)
		return nil
	case equalWord(tok, "pusha"):
		p.current.data = append(p.current.data, 0x60)
		return nil
	case equalWord(tok, "popa"):
		p.current.data = append(p.current.data, 0x61)
		return nil
	case equalWord(tok, "lea"):
		return p.emitLea(rest)
	case equalWord(tok, "jmp"):
		return p.emitJmp(rest)
	case equalWord(tok, "je"):
		return p.emitJcc(0x84, rest)
	case equalWord(tok, "jne"):
		return p.emitJcc(0x85, rest)
	case equalWord(tok, "jl"):
		return p.emitJcc(0x8C, rest)
	case equalWord(tok, "jge"):
		return p.emitJcc(0x8D, rest)
	case equalWord(tok, "ja"):
		return p.emitJcc(0x87, rest)
	case equalWord(tok, "jb"):
		return p.emitJcc(0x82, rest)
	case equalWord(tok, "jbe"):
		return p.emitJcc(0x86, rest)
	case equalWord(tok, "jae"):
		return p.emitJcc(0x83, rest)
	case equalWord(tok, "jc"):
		return p.emitJcc(0x82, rest)
	case equalWord(tok, "jnc"):
		return p.emitJcc(0x83, rest)
	case equalWord(tok, "jz"):
		return p.emitJcc(0x84, rest)
	case equalWord(tok, "jnz"):
		return p.emitJcc(0x85, rest)
	case equalWord(tok, "jo"):
		return p.emitJcc(0x80, rest)
	case equalWord(tok, "jno"):
		return p.emitJcc(0x81, rest)
	case equalWord(tok, "js"):
		return p.emitJcc(0x88, rest)
	case equalWord(tok, "jns"):
		return p.emitJcc(0x89, rest)
	case equalWord(tok, "jp"):
		return p.emitJcc(0x8A, rest)
	case equalWord(tok, "jnp"):
		return p.emitJcc(0x8B, rest)
	case equalWord(tok, "test"):
		return p.emitTest(rest)
	case equalWord(tok, "shl"):
		return p.emitShift(0xD2, 4, rest)
	case equalWord(tok, "shr"):
		return p.emitShift(0xD2, 5, rest)
	case equalWord(tok, "sar"):
		return p.emitShift(0xD2, 7, rest)
	case equalWord(tok, "sal"):
		return p.emitShift(0xD2, 4, rest)
	case equalWord(tok, "rol"):
		return p.emitShift(0xD2, 0, rest)
	case equalWord(tok, "ror"):
		return p.emitShift(0xD2, 1, rest)
	case equalWord(tok, "rcl"):
		return p.emitShift(0xD2, 2, rest)
	case equalWord(tok, "rcr"):
		return p.emitShift(0xD2, 3, rest)
	case equalWord(tok, "mul"):
		return p.emitMulDiv(0xF7, 4, rest)
	case equalWord(tok, "div"):
		return p.emitMulDiv(0xF7, 6, rest)
	case equalWord(tok, "idiv"):
		return p.emitMulDiv(0xF7, 7, rest)
	case equalWord(tok, "mov"):
		return p.emitMov(rest)
	case equalWord(tok, "movzx"):
		return p.emitMovzx(rest)
	case equalWord(tok, "movsx"):
		return p.emitMovsx(rest)
	case equalWord(tok, "xchg"):
		return p.emitXchg(rest)
	case equalWord(tok, "call"):
		return p.emitCall(rest)
	case equalWord(tok, "imul"):
		return p.emitImul(rest)
	case equalWord(tok, "cmp"):
		return p.emitCmp(rest)
	case equalWord(tok, "jle"):
		return p.emitJump(0x8E, rest)
	case equalWord(tok, "jg"):
		return p.emitJump(0x8F, rest)
	case equalWord(tok, "nop"):
		p.current.data = append(p.current.data, 0x90)
		return nil
	case equalWord(tok, "hlt"):
		p.current.data = append(p.current.data, 0xF4)
		return nil
	case equalWord(tok, "cli"):
		p.current.data = append(p.current.data, 0xFA)
		return nil
	case equalWord(tok, "sti"):
		p.current.data = append(p.current.data, 0xFB)
		return nil
	case equalWord(tok, "iret"):
		p.current.data = append(p.current.data, 0xCF)
		return nil
	case equalWord(tok, "int"):
		tok2, _ := readToken(rest)
		num, err := parseNumber(tok2)
		if err != nil {
			return err
		}
		if num == 3 {
			p.current.data = append(p.current.data, 0xCC)
		} else {
			p.current.data = append(p.current.data, 0xCD, byte(num))
		}
		return nil
	case equalWord(tok, "ud2"):
		p.current.data = append(p.current.data, 0x0F, 0x0B)
		return nil
	case equalWord(tok, "cpuid"):
		p.current.data = append(p.current.data, 0x0F, 0xA2)
		return nil
	case equalWord(tok, "rdtsc"):
		p.current.data = append(p.current.data, 0x0F, 0x31)
		return nil
	case equalWord(tok, "syscall"):
		p.current.data = append(p.current.data, 0x0F, 0x05)
		return nil
	case equalWord(tok, "sysret"):
		p.current.data = append(p.current.data, 0x0F, 0x07)
		return nil
	case equalWord(tok, "swapgs"):
		p.current.data = append(p.current.data, 0x0F, 0x01, 0xF8)
		return nil
	case equalWord(tok, "ret"):
		p.current.data = append(p.current.data, 0xC3)
		return nil
	case equalWord(tok, "retf"):
		p.current.data = append(p.current.data, 0xCB)
		return nil
	case equalWord(tok, "enter"):
		tok2, rest2 := readToken(rest)
		bytes, err := parseNumber(tok2)
		if err != nil {
			return err
		}
		level, _ := parseNumber(trimSpace(rest2))
		p.current.data = append(p.current.data, 0xC8)
		p.current.data = append(p.current.data, byte(bytes), byte(bytes>>8))
		p.current.data = append(p.current.data, byte(level))
		return nil
	case equalWord(tok, "leave"):
		p.current.data = append(p.current.data, 0xC9)
		return nil
	case equalWord(tok, "pause"):
		p.current.data = append(p.current.data, 0xF3, 0x90)
		return nil
	case equalWord(tok, "lock"):
		p.current.data = append(p.current.data, 0xF0)
		tok2, rest2 := readToken(rest)
		return p.parseLine(append(tok2, rest2...))
	}
	return errors.New("unsupported assembler directive")
}

func (p *parser) switchSection(name []byte) error {
	if equalWord(name, ".text") || equalWord(name, "text") {
		p.current = &p.text
		return nil
	}
	if equalWord(name, ".data") || equalWord(name, "data") {
		p.current = &p.data
		return nil
	}
	if equalWord(name, ".bss") || equalWord(name, "bss") {
		p.current = &p.bss
		return nil
	}
	return errors.New("unknown section")
}

func (p *parser) defineLabel(name []byte) error {
	if len(name) == 0 {
		return errors.New("empty label")
	}
	return p.addSymbol(name, currentOffset(p.current), sectionIndex(p.current), stBindLocal)
}

func (p *parser) defineGlobal(name []byte) error {
	if len(name) == 0 {
		return errors.New("empty global")
	}
	return p.addSymbol(name, 0, shnUnDef, stBindGlobal)
}

func sectionIndex(s *sectionState) uint16 {
	switch s.name[1] {
	case 't':
		return 1
	case 'd':
		return 2
	case 'b':
		return 3
	}
	return shnUnDef
}

func currentOffset(s *sectionState) uint64 {
	if s == nil {
		return 0
	}
	if len(s.name) > 1 && s.name[1] == 'b' {
		return s.size
	}
	return uint64(len(s.data))
}

func (p *parser) addSymbol(name []byte, value uint64, shndx uint16, bind byte) error {
	idx := p.findSymbol(name)
	if idx >= 0 {
		sym := &p.symbols[idx]
		if shndx != shnUnDef {
			sym.value = value
			sym.shndx = shndx
		}
		if bind == stBindGlobal {
			sym.bind = stBindGlobal
		}
		return nil
	}
	if p.symCount >= len(p.symbols) {
		return errors.New("symbol table capacity exceeded")
	}
	p.symbols[p.symCount] = symbolState{name: name, value: value, shndx: shndx, bind: bind, typ: stTypeNotype}
	p.symCount++
	return nil
}

func (p *parser) findSymbol(name []byte) int {
	for i := 0; i < p.symCount; i++ {
		if equalBytes(p.symbols[i].name, name) {
			return i
		}
	}
	return -1
}

func (p *parser) emitBytes(rest []byte) error {
	items := p.splitArgs(rest)
	for _, item := range items {
		item = trimSpace(item)
		if len(item) == 0 {
			continue
		}
		if item[0] == '"' || item[0] == '\'' {
			val, err := parseString(item)
			if err != nil {
				return err
			}
			p.current.data = append(p.current.data, val...)
			continue
		}
		v, err := parseNumber(item)
		if err != nil {
			return err
		}
		p.current.data = append(p.current.data, byte(v))
	}
	return nil
}

func (p *parser) emitWords(rest []byte, width int) error {
	items := p.splitArgs(rest)
	for _, item := range items {
		item = trimSpace(item)
		if len(item) == 0 {
			continue
		}
		v, err := parseNumber(item)
		if err != nil {
			return err
		}
		for i := 0; i < width; i++ {
			p.current.data = append(p.current.data, byte(v>>uint(i*8)))
		}
	}
	return nil
}

func (p *parser) reserve(rest []byte, width int) error {
	item, _ := readToken(rest)
	v, err := parseNumber(item)
	if err != nil {
		return err
	}
	if p.current == &p.bss {
		p.bss.size += v * uint64(width)
		return nil
	}
	for i := uint64(0); i < v*uint64(width); i++ {
		p.current.data = append(p.current.data, 0)
	}
	return nil
}

func (p *parser) alignSection(arg []byte) error {
	v, err := parseNumber(trimSpace(arg))
	if err != nil {
		return err
	}
	if v == 0 {
		return nil
	}
	if p.current == &p.bss {
		p.bss.size = alignValue(p.bss.size, uint64(v))
		return nil
	}
	pad := alignValue(uint64(len(p.current.data)), uint64(v)) - uint64(len(p.current.data))
	for i := uint64(0); i < pad; i++ {
		p.current.data = append(p.current.data, 0)
	}
	return nil
}

func (p *parser) emit(profile targetEmitterProfile) ([]byte, error) {
	shstrtabNames0 := nameEmpty
	shstrtabNames1 := p.text.name
	shstrtabNames2 := p.data.name
	shstrtabNames3 := p.bss.name
	shstrtabNames4 := nameShstrtab
	shstrtabNames5 := nameSymtab
	shstrtabNames6 := nameStrtab
	shstrtabNames7 := []byte(".rela.text")
	shstrtabLen := 1 + (len(shstrtabNames0) + 1) + (len(shstrtabNames1) + 1) + (len(shstrtabNames2) + 1) + (len(shstrtabNames3) + 1) + (len(shstrtabNames4) + 1) + (len(shstrtabNames5) + 1) + (len(shstrtabNames6) + 1) + (len(shstrtabNames7) + 1)
	strtabLen := 1
	for i := 0; i < p.symCount; i++ {
		strtabLen += len(p.symbols[i].name) + 1
	}
	symEntrySize := 24
	if profile.elfClass == elfClass32 {
		symEntrySize = 16
	}
	symtabSize := (p.symCount + 1) * symEntrySize
	var outPtr *[]byte
	if reusableOut != nil {
		outPtr = reusableOut
	} else {
		outPtr = emitterBufferPool.Get().(*[]byte)
	}
	out := *outPtr
	out = out[:0]
	ehSize := 64
	if profile.elfClass == elfClass32 {
		ehSize = 52
	}
	shdrSize := elf64ShdrSize
	if profile.elfClass == elfClass32 {
		shdrSize = elf32ShdrSize
	}
	need := ehSize + len(p.text.data) + len(p.data.data) + symtabSize + strtabLen + shstrtabLen + shdrSize*8
	need += 512
	if cap(out) < need {
		out = make([]byte, 0, need)
	}
	for i := 0; i < ehSize; i++ {
		out = append(out, 0)
	}
	textOffset := 0
	dataOffset := 0
	if len(p.text.data) > 0 {
		out = alignOut(out, uint64(p.text.align))
		textOffset = len(out)
		out = append(out, p.text.data...)
	}
	if len(p.data.data) > 0 {
		out = alignOut(out, uint64(p.data.align))
		dataOffset = len(out)
		out = append(out, p.data.data...)
	}
	align := uint64(8)
	if profile.elfClass == elfClass32 {
		align = 4
	}
	out = alignOut(out, align)
	symtabOffset := len(out)
	base := len(out)
	out = out[:base+symtabSize]
	strtabOffset := len(out)
	out = append(out, 0)
	for i := 0; i < p.symCount; i++ {
		p.symbols[i].nameOffset = uint32(len(out) - strtabOffset)
		out = append(out, p.symbols[i].name...)
		out = append(out, 0)
	}
	if profile.elfClass == elfClass64 {
		off := symtabOffset
		writeElf64SymAt(out, off, 0, 0, 0, 0, 0, 0)
		off += 24
		for i := 0; i < p.symCount; i++ {
			if p.symbols[i].bind != stBindLocal {
				continue
			}
			info := byte(p.symbols[i].bind<<4 | p.symbols[i].typ)
			writeElf64SymAt(out, off, p.symbols[i].nameOffset, info, 0, p.symbols[i].shndx, p.symbols[i].value, 0)
			off += 24
		}
		for i := 0; i < p.symCount; i++ {
			if p.symbols[i].bind == stBindLocal {
				continue
			}
			info := byte(p.symbols[i].bind<<4 | p.symbols[i].typ)
			writeElf64SymAt(out, off, p.symbols[i].nameOffset, info, 0, p.symbols[i].shndx, p.symbols[i].value, 0)
			off += 24
		}
	} else {
		off := symtabOffset
		writeElf32SymAt(out, off, 0, 0, 0, 0, 0, 0)
		off += 16
		for i := 0; i < p.symCount; i++ {
			if p.symbols[i].bind != stBindLocal {
				continue
			}
			info := byte(p.symbols[i].bind<<4 | p.symbols[i].typ)
			writeElf32SymAt(out, off, p.symbols[i].nameOffset, info, 0, p.symbols[i].shndx, uint32(p.symbols[i].value), 0)
			off += 16
		}
		for i := 0; i < p.symCount; i++ {
			if p.symbols[i].bind == stBindLocal {
				continue
			}
			info := byte(p.symbols[i].bind<<4 | p.symbols[i].typ)
			writeElf32SymAt(out, off, p.symbols[i].nameOffset, info, 0, p.symbols[i].shndx, uint32(p.symbols[i].value), 0)
			off += 16
		}
	}
	shstrtabOffset := len(out)
	out = append(out, 0)
	out = append(out, shstrtabNames0...)
	out = append(out, 0)
	out = append(out, shstrtabNames1...)
	out = append(out, 0)
	out = append(out, shstrtabNames2...)
	out = append(out, 0)
	out = append(out, shstrtabNames3...)
	out = append(out, 0)
	out = append(out, shstrtabNames4...)
	out = append(out, 0)
	out = append(out, shstrtabNames5...)
	out = append(out, 0)
	out = append(out, shstrtabNames6...)
	out = append(out, 0)
	out = append(out, shstrtabNames7...)
	out = append(out, 0)
	shstrtab := out[shstrtabOffset : shstrtabOffset+shstrtabLen]
	shOff := alignOutOffset(len(out), uint64(shdrSize))
	out = alignOut(out, uint64(shdrSize))
	numSections := 8
	shSize := shdrSize
	for i := 0; i < numSections; i++ {
		for j := 0; j < shSize; j++ {
			out = append(out, 0)
		}
	}
	if profile.elfClass == elfClass64 {
		mapIdx := make([]int, p.symCount)
		idx := 1
		for i := 0; i < p.symCount; i++ {
			if p.symbols[i].bind == stBindLocal {
				mapIdx[i] = idx
				idx++
			}
		}
		for i := 0; i < p.symCount; i++ {
			if p.symbols[i].bind == stBindLocal {
				continue
			}
			mapIdx[i] = idx
			idx++
		}
		relaOffset := 0
		relaSize := 0
		if len(p.relocs) > 0 {
			cnt := 0
			for i := 0; i < len(p.relocs); i++ {
				if p.relocs[i].sec == 1 {
					cnt++
				}
			}
			if cnt > 0 {
				out = alignOut(out, 8)
				relaOffset = len(out)
				for i := 0; i < len(p.relocs); i++ {
					r := p.relocs[i]
					if r.sec != 1 {
						continue
					}
					out = appendUint64(out, r.off)
					out = appendUint64(out, (uint64(mapIdx[r.sym])<<32)|uint64(r.typ))
					out = appendUint64(out, uint64(r.addend))
				}
				relaSize = (cnt * 24)
			}
		}
		populateELF64Header(out, profile, uint64(shOff), uint16(shSize), uint16(numSections), 4)
		populateELF64SectionHeaders(out[shOff:], out, p, uint64(textOffset), uint64(dataOffset), uint64(symtabOffset), uint64(strtabOffset), uint64(shstrtabOffset), uint64(symtabSize), uint64(strtabLen), uint64(len(shstrtab)), uint64(relaOffset), uint64(relaSize))
	} else {
		populateELF32Header(out, profile, uint32(shOff), uint16(shSize), uint16(numSections), 4)
		populateELF32SectionHeaders(out[shOff:], out, p, uint32(textOffset), uint32(dataOffset), uint32(symtabOffset), uint32(strtabOffset), uint32(shstrtabOffset), uint32(symtabSize), uint32(strtabLen), uint32(len(shstrtab)), 0, 0)
	}
	*outPtr = out
	return out, nil
}

func appendElf64Sym(out []byte, name uint32, info byte, other byte, shndx uint16, value uint64, size uint64) []byte {
	out = appendUint32(out, name)
	out = appendByte(out, info)
	out = appendByte(out, other)
	out = appendUint16(out, shndx)
	out = appendUint64(out, value)
	out = appendUint64(out, size)
	return out
}

func symbolLocalCount(p *parser) int {
	count := 0
	for i := 0; i < p.symCount; i++ {
		if p.symbols[i].bind == stBindLocal {
			count++
		}
	}
	return count
}

func matchBytes(data, pattern []byte) bool {
	if len(data) < len(pattern) {
		return false
	}
	for i := 0; i < len(pattern); i++ {
		if data[i] != pattern[i] {
			return false
		}
	}
	return true
}

func writeELF64Section(out []byte, name uint32, typ uint32, flags uint64, addr uint64, offset uint64, size uint64, link uint32, info uint32, addralign uint64, entsize uint64) {
	if len(out) < elf64ShdrSize {
		return
	}
	writeUint32At(out, 0, name)
	writeUint32At(out, 4, typ)
	writeUint64At(out, 8, flags)
	writeUint64At(out, 16, addr)
	writeUint64At(out, 24, offset)
	writeUint64At(out, 32, size)
	writeUint32At(out, 40, link)
	writeUint32At(out, 44, info)
	writeUint64At(out, 48, addralign)
	writeUint64At(out, 56, entsize)
}

func writeELF32Section(out []byte, name uint32, typ uint32, flags uint32, addr uint32, offset uint32, size uint32, link uint32, info uint32, addralign uint32, entsize uint32) {
	if len(out) < elf32ShdrSize {
		return
	}
	writeUint32At(out, 0, name)
	writeUint32At(out, 4, typ)
	writeUint32At(out, 8, flags)
	writeUint32At(out, 12, addr)
	writeUint32At(out, 16, offset)
	writeUint32At(out, 20, size)
	writeUint32At(out, 24, link)
	writeUint32At(out, 28, info)
	writeUint32At(out, 32, addralign)
	writeUint32At(out, 36, entsize)
}

func writeUint16At(b []byte, off int, v uint16) {
	b[off] = byte(v)
	b[off+1] = byte(v >> 8)
}

func writeUint32At(b []byte, off int, v uint32) {
	b[off] = byte(v)
	b[off+1] = byte(v >> 8)
	b[off+2] = byte(v >> 16)
	b[off+3] = byte(v >> 24)
}

func writeUint64At(b []byte, off int, v uint64) {
	b[off] = byte(v)
	b[off+1] = byte(v >> 8)
	b[off+2] = byte(v >> 16)
	b[off+3] = byte(v >> 24)
	b[off+4] = byte(v >> 32)
	b[off+5] = byte(v >> 40)
	b[off+6] = byte(v >> 48)
	b[off+7] = byte(v >> 56)
}

func writeElf64SymAt(dst []byte, off int, name uint32, info byte, other byte, shndx uint16, value uint64, size uint64) {
	writeUint32At(dst, off+0, name)
	dst[off+4] = info
	dst[off+5] = other
	writeUint16At(dst, off+6, shndx)
	writeUint64At(dst, off+8, value)
	writeUint64At(dst, off+16, size)
}

func writeElf32SymAt(dst []byte, off int, name uint32, info byte, other byte, shndx uint16, value uint32, size uint32) {
	writeUint32At(dst, off+0, name)
	dst[off+4] = info
	dst[off+5] = other
	writeUint16At(dst, off+6, shndx)
	writeUint32At(dst, off+8, value)
	writeUint32At(dst, off+12, size)
}

func populateELF64Header(out []byte, profile targetEmitterProfile, shoff uint64, shentsize uint16, shnum uint16, shstrndx uint16) {
	out[0] = 0x7f
	out[1] = 'E'
	out[2] = 'L'
	out[3] = 'F'
	out[4] = elfClass64
	out[5] = elfData2LSB
	out[6] = elfVersion1
	out[7] = 0
	out[8] = 0
	out[9] = 0
	out[10] = 0
	out[11] = 0
	out[12] = 0
	out[13] = 0
	out[14] = 0
	out[15] = 0
	writeUint16At(out, 16, elfTypeRel)
	writeUint16At(out, 18, profile.eMachine)
	writeUint32At(out, 20, elfVersion1)
	writeUint64At(out, 24, 0)
	writeUint64At(out, 32, 0)
	writeUint64At(out, 40, shoff)
	writeUint32At(out, 48, 0)
	writeUint16At(out, 52, 64)
	writeUint16At(out, 54, 0)
	writeUint16At(out, 56, 0)
	writeUint16At(out, 58, shentsize)
	writeUint16At(out, 60, shnum)
	writeUint16At(out, 62, shstrndx)
}

func populateELF32Header(out []byte, profile targetEmitterProfile, shoff uint32, shentsize uint16, shnum uint16, shstrndx uint16) {
	out[0] = 0x7f
	out[1] = 'E'
	out[2] = 'L'
	out[3] = 'F'
	out[4] = elfClass32
	out[5] = elfData2LSB
	out[6] = elfVersion1
	out[7] = 0
	out[8] = 0
	out[9] = 0
	out[10] = 0
	out[11] = 0
	out[12] = 0
	out[13] = 0
	out[14] = 0
	out[15] = 0
	writeUint16At(out, 16, elfTypeRel)
	writeUint16At(out, 18, profile.eMachine)
	writeUint32At(out, 20, elfVersion1)
	writeUint32At(out, 24, 0)
	writeUint32At(out, 28, 0)
	writeUint32At(out, 32, shoff)
	writeUint32At(out, 36, 0)
	writeUint16At(out, 40, 52)
	writeUint16At(out, 42, 0)
	writeUint16At(out, 44, 0)
	writeUint16At(out, 46, shentsize)
	writeUint16At(out, 48, shnum)
	writeUint16At(out, 50, shstrndx)
}

func populateELF64SectionHeaders(sec []byte, full []byte, p *parser, textOffset, dataOffset, symtabOffset, strtabOffset, shstrtabOffset uint64, symtabSize, strtabSize, shstrtabSize uint64, relaOffset, relaSize uint64) {
	shstr := full[shstrtabOffset : shstrtabOffset+shstrtabSize]
	writeELF64Section(sec[0:elf64ShdrSize], 0, shTypeNull, 0, 0, 0, 0, 0, 0, 0, 0)
	writeELF64Section(sec[elf64ShdrSize:elf64ShdrSize*2], uint32(offsetOfNameInShstr(p.text.name, shstr)), shTypeProgBits, uint64(p.text.flags), 0, uint64(textOffset), uint64(len(p.text.data)), 0, 0, uint64(p.text.align), 0)
	writeELF64Section(sec[elf64ShdrSize*2:elf64ShdrSize*3], uint32(offsetOfNameInShstr(p.data.name, shstr)), shTypeProgBits, uint64(p.data.flags), 0, uint64(dataOffset), uint64(len(p.data.data)), 0, 0, uint64(p.data.align), 0)
	writeELF64Section(sec[elf64ShdrSize*3:elf64ShdrSize*4], uint32(offsetOfNameInShstr(p.bss.name, shstr)), shTypeNoBits, uint64(p.bss.flags), 0, 0, p.bss.size, 0, 0, uint64(p.bss.align), 0)
	writeELF64Section(sec[elf64ShdrSize*4:elf64ShdrSize*5], uint32(offsetOfNameInShstr(nameShstrtab, shstr)), shTypeStrTab, 0, 0, shstrtabOffset, shstrtabSize, 0, 0, 1, 0)
	writeELF64Section(sec[elf64ShdrSize*5:elf64ShdrSize*6], uint32(offsetOfNameInShstr(nameSymtab, shstr)), shTypeSymTab, 0, 0, symtabOffset, symtabSize, 6, uint32(symbolLocalCount(p)+1), 8, 24)
	writeELF64Section(sec[elf64ShdrSize*6:elf64ShdrSize*7], uint32(offsetOfNameInShstr(nameStrtab, shstr)), shTypeStrTab, 0, 0, strtabOffset, strtabSize, 0, 0, 1, 0)
	writeELF64Section(sec[elf64ShdrSize*7:elf64ShdrSize*8], uint32(offsetOfNameInShstr([]byte(".rela.text"), shstr)), shTypeRela, 0, 0, relaOffset, relaSize, 5, 1, 8, 24)
}

func offsetOfNameInShstr(name []byte, shstrtab []byte) uint32 {
	if len(name) == 0 {
		return 0
	}
	for i := 0; i+len(name) <= len(shstrtab); i++ {
		if matchBytes(shstrtab[i:i+len(name)], name) {
			return uint32(i)
		}
	}
	return 0
}

func populateELF32SectionHeaders(sec []byte, full []byte, p *parser, textOffset, dataOffset, symtabOffset, strtabOffset, shstrtabOffset uint32, symtabSize, strtabSize, shstrtabSize uint32, relaOffset uint32, relaSize uint32) {
	shstr := full[shstrtabOffset : shstrtabOffset+shstrtabSize]
	writeELF32Section(sec[0:elf32ShdrSize], 0, shTypeNull, 0, 0, 0, 0, 0, 0, 0, 0)
	writeELF32Section(sec[elf32ShdrSize:elf32ShdrSize*2], uint32(offsetOfNameInShstr(p.text.name, shstr)), shTypeProgBits, p.text.flags, 0, textOffset, uint32(len(p.text.data)), 0, 0, p.text.align, 0)
	writeELF32Section(sec[elf32ShdrSize*2:elf32ShdrSize*3], uint32(offsetOfNameInShstr(p.data.name, shstr)), shTypeProgBits, p.data.flags, 0, dataOffset, uint32(len(p.data.data)), 0, 0, p.data.align, 0)
	writeELF32Section(sec[elf32ShdrSize*3:elf32ShdrSize*4], uint32(offsetOfNameInShstr(p.bss.name, shstr)), shTypeNoBits, p.bss.flags, 0, 0, uint32(p.bss.size), 0, 0, p.bss.align, 0)
	writeELF32Section(sec[elf32ShdrSize*4:elf32ShdrSize*5], uint32(offsetOfNameInShstr(nameShstrtab, shstr)), shTypeStrTab, 0, 0, shstrtabOffset, shstrtabSize, 0, 0, 1, 0)
	writeELF32Section(sec[elf32ShdrSize*5:elf32ShdrSize*6], uint32(offsetOfNameInShstr(nameSymtab, shstr)), shTypeSymTab, 0, 0, symtabOffset, symtabSize, 6, uint32(symbolLocalCount(p)+1), 8, 16)
	writeELF32Section(sec[elf32ShdrSize*6:elf32ShdrSize*7], uint32(offsetOfNameInShstr(nameStrtab, shstr)), shTypeStrTab, 0, 0, strtabOffset, strtabSize, 0, 0, 1, 0)
	writeELF32Section(sec[elf32ShdrSize*7:elf32ShdrSize*8], uint32(offsetOfNameInShstr([]byte(".rela.text"), shstr)), shTypeRela, 0, 0, relaOffset, relaSize, 5, 1, 4, 12)
}

func equalBytes(a, b []byte) bool {
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

func equalWord(token []byte, literal string) bool {
	if len(token) != len(literal) {
		return false
	}
	for i := 0; i < len(token); i++ {
		if token[i] != literal[i] {
			return false
		}
	}
	return true
}

func readToken(line []byte) ([]byte, []byte) {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	start := i
	for i < len(line) && line[i] != ' ' && line[i] != '\t' {
		i++
	}
	return line[start:i], line[i:]
}

func (p *parser) splitArgs(data []byte) [][]byte {
	p.argsPool = p.argsPool[:0]

	start := 0
	in := false
	var q byte
	for i := 0; i < len(data); i++ {
		c := data[i]
		if in {
			if c == q {
				in = false
			}
			continue
		}
		if c == '"' || c == '\'' {
			in = true
			q = c
			continue
		}
		if c == ',' {
			p.argsPool = append(p.argsPool, data[start:i])
			start = i + 1
		}
	}
	if start <= len(data) {
		p.argsPool = append(p.argsPool, data[start:])
	}
	return p.argsPool
}

type registerInfo struct {
	code  byte
	width int
}

func parseRegister(tok []byte) (registerInfo, bool) {
	s := string(tok)
	switch s {
	case "rax":
		return registerInfo{code: 0, width: 64}, true
	case "rcx":
		return registerInfo{code: 1, width: 64}, true
	case "rdx":
		return registerInfo{code: 2, width: 64}, true
	case "rbx":
		return registerInfo{code: 3, width: 64}, true
	case "rsp":
		return registerInfo{code: 4, width: 64}, true
	case "rbp":
		return registerInfo{code: 5, width: 64}, true
	case "rsi":
		return registerInfo{code: 6, width: 64}, true
	case "rdi":
		return registerInfo{code: 7, width: 64}, true
	case "r8":
		return registerInfo{code: 8, width: 64}, true
	case "r9":
		return registerInfo{code: 9, width: 64}, true
	case "r10":
		return registerInfo{code: 10, width: 64}, true
	case "r11":
		return registerInfo{code: 11, width: 64}, true
	case "r12":
		return registerInfo{code: 12, width: 64}, true
	case "r13":
		return registerInfo{code: 13, width: 64}, true
	case "r14":
		return registerInfo{code: 14, width: 64}, true
	case "r15":
		return registerInfo{code: 15, width: 64}, true
	case "eax":
		return registerInfo{code: 0, width: 32}, true
	case "ecx":
		return registerInfo{code: 1, width: 32}, true
	case "edx":
		return registerInfo{code: 2, width: 32}, true
	case "ebx":
		return registerInfo{code: 3, width: 32}, true
	case "esp":
		return registerInfo{code: 4, width: 32}, true
	case "ebp":
		return registerInfo{code: 5, width: 32}, true
	case "esi":
		return registerInfo{code: 6, width: 32}, true
	case "edi":
		return registerInfo{code: 7, width: 32}, true
	case "r8d":
		return registerInfo{code: 8, width: 32}, true
	case "r9d":
		return registerInfo{code: 9, width: 32}, true
	case "r10d":
		return registerInfo{code: 10, width: 32}, true
	case "r11d":
		return registerInfo{code: 11, width: 32}, true
	case "r12d":
		return registerInfo{code: 12, width: 32}, true
	case "r13d":
		return registerInfo{code: 13, width: 32}, true
	case "r14d":
		return registerInfo{code: 14, width: 32}, true
	case "r15d":
		return registerInfo{code: 15, width: 32}, true
	case "ax":
		return registerInfo{code: 0, width: 16}, true
	case "cx":
		return registerInfo{code: 1, width: 16}, true
	case "dx":
		return registerInfo{code: 2, width: 16}, true
	case "bx":
		return registerInfo{code: 3, width: 16}, true
	case "sp":
		return registerInfo{code: 4, width: 16}, true
	case "bp":
		return registerInfo{code: 5, width: 16}, true
	case "si":
		return registerInfo{code: 6, width: 16}, true
	case "di":
		return registerInfo{code: 7, width: 16}, true
	case "al":
		return registerInfo{code: 0, width: 8}, true
	case "cl":
		return registerInfo{code: 1, width: 8}, true
	case "dl":
		return registerInfo{code: 2, width: 8}, true
	case "bl":
		return registerInfo{code: 3, width: 8}, true
	case "ah":
		return registerInfo{code: 4, width: 8}, true
	case "ch":
		return registerInfo{code: 5, width: 8}, true
	case "dh":
		return registerInfo{code: 6, width: 8}, true
	case "bh":
		return registerInfo{code: 7, width: 8}, true
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

		parts := splitByPlus(inner)
		for _, part := range parts {
			part = trimSpace(part)
			if len(part) == 0 {
				continue
			}
			sign := int64(1)
			if part[0] == '+' || part[0] == '-' {
				if part[0] == '-' {
					sign = -1
				}
				part = part[1:]
				part = trimSpace(part)
				if len(part) == 0 {
					return operand{}, errors.New("expected number after sign")
				}
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
				if err != nil {
					return operand{}, errors.New("invalid scale")
				}
				if scaleVal != 1 && scaleVal != 2 && scaleVal != 4 && scaleVal != 8 {
					return operand{}, errors.New("scale must be 1,2,4,8")
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

func splitByPlus(b []byte) [][]byte {
	var parts [][]byte
	start := 0
	for i := 0; i < len(b); i++ {
		if b[i] == '+' || b[i] == '-' {
			if i > start {
				parts = append(parts, b[start:i])
			}
			start = i
		}
	}
	if start < len(b) {
		parts = append(parts, b[start:])
	}
	return parts
}

func encodeModRM(mod byte, reg byte, rm byte) byte {
	return (mod << 6) | ((reg & 7) << 3) | (rm & 7)
}

func encodeSIB(scale byte, index byte, base byte) byte {
	return ((scale & 3) << 6) | ((index & 7) << 3) | (base & 7)
}

func encodeMemOperand(mem operand, reg byte, isLea bool) (modrm byte, sib byte, disp []byte) {
	base := mem.base
	index := mem.index
	scale := mem.scale
	dispVal := mem.disp

	if index != 255 {
		if dispVal == 0 {
			modrm = encodeModRM(0, reg, 4)
			sib = encodeSIB(scale, index, base)
			return modrm, sib, nil
		}
		if dispVal >= -128 && dispVal <= 127 {
			modrm = encodeModRM(1, reg, 4)
			sib = encodeSIB(scale, index, base)
			var b [1]byte
			b[0] = byte(dispVal)
			return modrm, sib, b[:]
		}
		modrm = encodeModRM(2, reg, 4)
		sib = encodeSIB(scale, index, base)
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], uint32(dispVal))
		return modrm, sib, b[:]
	}
	if base == 255 {
		modrm = encodeModRM(0, reg, 5)
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], uint32(dispVal))
		return modrm, 0, b[:]
	}
	if dispVal == 0 {
		modrm = encodeModRM(0, reg, base)
		return modrm, 0, nil
	}
	if dispVal >= -128 && dispVal <= 127 {
		modrm = encodeModRM(1, reg, base)
		var b [1]byte
		b[0] = byte(dispVal)
		return modrm, 0, b[:]
	}
	modrm = encodeModRM(2, reg, base)
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(dispVal))
	return modrm, 0, b[:]
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
		p.current.data = append(p.current.data, 0x48, opRegReg)
		p.current.data = append(p.current.data, encodeModRM(3, src.reg, dst.reg))
		return nil
	}
	if dst.typ == opReg && src.typ == opImm {
		p.current.data = append(p.current.data, 0x48, opRegImm)
		p.current.data = append(p.current.data, encodeModRM(3, 0, dst.reg))
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(src.imm))
		p.current.data = append(p.current.data, buf[:]...)
		return nil
	}
	if dst.typ == opMem && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, opRegReg)
		modrm, sib, disp := encodeMemOperand(dst, src.reg, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
		return nil
	}
	if dst.typ == opReg && src.typ == opMem {
		p.current.data = append(p.current.data, 0x48, opRegReg)
		modrm, sib, disp := encodeMemOperand(src, dst.reg, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
		return nil
	}
	if dst.typ == opMem && src.typ == opImm {
		p.current.data = append(p.current.data, 0x48, opRegImm)
		modrm, sib, disp := encodeMemOperand(dst, 0, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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
		p.current.data = append(p.current.data, 0x48, 0xFF)
		p.current.data = append(p.current.data, encodeModRM(3, 0, op.reg))
		return nil
	}
	if op.typ == opMem {
		p.current.data = append(p.current.data, 0x48, 0xFF)
		modrm, sib, disp := encodeMemOperand(op, 0, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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
		p.current.data = append(p.current.data, 0x48, 0xFF)
		p.current.data = append(p.current.data, encodeModRM(3, 1, op.reg))
		return nil
	}
	if op.typ == opMem {
		p.current.data = append(p.current.data, 0x48, 0xFF)
		modrm, sib, disp := encodeMemOperand(op, 1, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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
		p.current.data = append(p.current.data, 0x48, opcode)
		p.current.data = append(p.current.data, encodeModRM(3, ext, op.reg))
		return nil
	}
	if op.typ == opMem {
		p.current.data = append(p.current.data, 0x48, opcode)
		modrm, sib, disp := encodeMemOperand(op, ext, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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
		modrm, sib, disp := encodeMemOperand(op, 6, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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
		modrm, sib, disp := encodeMemOperand(op, 0, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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
	modrm, sib, disp := encodeMemOperand(src, dst.reg, true)
	p.current.data = append(p.current.data, modrm)
	if sib != 0 {
		p.current.data = append(p.current.data, sib)
	}
	if disp != nil {
		p.current.data = append(p.current.data, disp...)
	}
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
		modrm, sib, disp := encodeMemOperand(op, 4, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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
		p.current.data = append(p.current.data, 0x48, 0x85)
		p.current.data = append(p.current.data, encodeModRM(3, src.reg, dst.reg))
		return nil
	}
	if dst.typ == opReg && src.typ == opImm {
		p.current.data = append(p.current.data, 0x48, 0xF7)
		p.current.data = append(p.current.data, encodeModRM(3, 0, dst.reg))
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(src.imm))
		p.current.data = append(p.current.data, buf[:]...)
		return nil
	}
	if dst.typ == opMem && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, 0x85)
		modrm, sib, disp := encodeMemOperand(dst, src.reg, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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
		p.current.data = append(p.current.data, 0x48, opcode)
		p.current.data = append(p.current.data, encodeModRM(3, ext, dst.reg))
		var cnt byte
		if src.imm == 1 {
			cnt = 1
		} else {
			cnt = byte(src.imm & 0xFF)
		}
		p.current.data = append(p.current.data, cnt)
		return nil
	}
	if dst.typ == opMem && src.typ == opImm {
		p.current.data = append(p.current.data, 0x48, opcode)
		modrm, sib, disp := encodeMemOperand(dst, ext, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
		var cnt byte
		if src.imm == 1 {
			cnt = 1
		} else {
			cnt = byte(src.imm & 0xFF)
		}
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
		p.current.data = append(p.current.data, 0x48, opcode)
		p.current.data = append(p.current.data, encodeModRM(3, ext, op.reg))
		return nil
	}
	if op.typ == opMem {
		p.current.data = append(p.current.data, 0x48, opcode)
		modrm, sib, disp := encodeMemOperand(op, ext, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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
		p.current.data = append(p.current.data, 0x48, 0x89)
		p.current.data = append(p.current.data, encodeModRM(3, src.reg, dst.reg))
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
		modrm, sib, disp := encodeMemOperand(dst, src.reg, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
		return nil
	}
	if dst.typ == opReg && src.typ == opMem {
		p.current.data = append(p.current.data, 0x48, 0x8B)
		modrm, sib, disp := encodeMemOperand(src, dst.reg, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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
	modrm, sib, disp := encodeMemOperand(src, dst.reg, false)
	p.current.data = append(p.current.data, modrm)
	if sib != 0 {
		p.current.data = append(p.current.data, sib)
	}
	if disp != nil {
		p.current.data = append(p.current.data, disp...)
	}
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
	modrm, sib, disp := encodeMemOperand(src, dst.reg, false)
	p.current.data = append(p.current.data, modrm)
	if sib != 0 {
		p.current.data = append(p.current.data, sib)
	}
	if disp != nil {
		p.current.data = append(p.current.data, disp...)
	}
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
		p.current.data = append(p.current.data, 0x48, 0x87)
		p.current.data = append(p.current.data, encodeModRM(3, src.reg, dst.reg))
		return nil
	}
	if dst.typ == opMem && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, 0x87)
		modrm, sib, disp := encodeMemOperand(dst, src.reg, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
		return nil
	}
	if dst.typ == opReg && src.typ == opMem {
		p.current.data = append(p.current.data, 0x48, 0x87)
		modrm, sib, disp := encodeMemOperand(src, dst.reg, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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
		p.current.data = append(p.current.data, 0x48, 0x0F, 0xAF)
		p.current.data = append(p.current.data, encodeModRM(3, src.reg, dst.reg))
		return nil
	}
	if dst.typ == opReg && src.typ == opMem {
		p.current.data = append(p.current.data, 0x48, 0x0F, 0xAF)
		modrm, sib, disp := encodeMemOperand(src, dst.reg, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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
		p.current.data = append(p.current.data, 0x48, 0x39)
		p.current.data = append(p.current.data, encodeModRM(3, src.reg, dst.reg))
		return nil
	}
	if dst.typ == opReg && src.typ == opImm {
		p.current.data = append(p.current.data, 0x48, 0x81)
		p.current.data = append(p.current.data, encodeModRM(3, 7, dst.reg))
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(src.imm))
		p.current.data = append(p.current.data, buf[:]...)
		return nil
	}
	if dst.typ == opMem && src.typ == opReg {
		p.current.data = append(p.current.data, 0x48, 0x39)
		modrm, sib, disp := encodeMemOperand(dst, src.reg, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
		return nil
	}
	if dst.typ == opMem && src.typ == opImm {
		p.current.data = append(p.current.data, 0x48, 0x81)
		modrm, sib, disp := encodeMemOperand(dst, 7, false)
		p.current.data = append(p.current.data, modrm)
		if sib != 0 {
			p.current.data = append(p.current.data, sib)
		}
		if disp != nil {
			p.current.data = append(p.current.data, disp...)
		}
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

var asciiSpaceTable = [256]bool{
	' ':  true,
	'\t': true,
	'\r': true,
	'\n': true,
}

func trimSpace(data []byte) []byte {
	start := 0
	end := len(data)

	for start < end && asciiSpaceTable[data[start]] {
		start++
	}

	for end > start && asciiSpaceTable[data[end-1]] {
		end--
	}

	return data[start:end]
}

func removeComment(line []byte) []byte {
	idx := bytes.IndexAny(line, ";#")

	if idx == -1 {
		return line
	}

	return line[:idx]
}

func parseString(token []byte) ([]byte, error) {
	if len(token) < 2 {
		return nil, errors.New("invalid string")
	}
	quote := token[0]
	if token[len(token)-1] != quote {
		return nil, errors.New("unterminated string")
	}
	return token[1 : len(token)-1], nil
}

func parseNumber(token []byte) (uint64, error) {
	if len(token) == 0 {
		return 0, errors.New("empty number")
	}
	base := 10
	i := 0
	if token[0] == '-' {
		i = 1
	}
	if i+1 < len(token) && token[i] == '0' {
		switch token[i+1] {
		case 'x', 'X':
			base = 16
			i += 2
		case 'b', 'B':
			base = 2
			i += 2
		case 'o', 'O':
			base = 8
			i += 2
		}
	}
	var v uint64
	for ; i < len(token); i++ {
		c := token[i]
		var digit uint64
		if c >= '0' && c <= '9' {
			digit = uint64(c - '0')
		} else if base == 16 && c >= 'a' && c <= 'f' {
			digit = uint64(10 + c - 'a')
		} else if base == 16 && c >= 'A' && c <= 'F' {
			digit = uint64(10 + c - 'A')
		} else {
			return 0, errors.New("invalid digit")
		}
		if digit >= uint64(base) {
			return 0, errors.New("invalid digit for base")
		}
		v = v*uint64(base) + digit
	}
	return v, nil
}

func alignValue(value, align uint64) uint64 {
	if align == 0 {
		return value
	}
	mask := align - 1
	return (value + mask) &^ mask
}

func alignOut(out []byte, align uint64) []byte {
	if align == 0 {
		return out
	}
	pad := int((align - uint64(len(out))%align) % align)
	for i := 0; i < pad; i++ {
		out = append(out, 0)
	}
	return out
}

func alignOutOffset(offset int, align uint64) int {
	if align == 0 {
		return offset
	}
	mask := int(align - 1)
	return (offset + mask) &^ mask
}

func appendByte(out []byte, v byte) []byte {
	return append(out, v)
}

func containsFold(s, substr string) bool {
	slen := len(s)
	subLen := len(substr)
	if subLen == 0 || subLen > slen {
		return false
	}
	for i := 0; i <= slen-subLen; i++ {
		match := true
		for j := 0; j < subLen; j++ {
			sc := s[i+j]
			pc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 'a' - 'A'
			}
			if pc >= 'A' && pc <= 'Z' {
				pc += 'a' - 'A'
			}
			if sc != pc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func appendUint16(out []byte, v uint16) []byte {
	return append(out, byte(v), byte(v>>8))
}

func appendUint32(out []byte, v uint32) []byte {
	return append(out, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func appendUint64(out []byte, v uint64) []byte {
	return append(out, byte(v), byte(v>>8), byte(v>>16), byte(v>>24), byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
}
