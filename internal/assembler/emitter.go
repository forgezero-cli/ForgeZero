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
	"errors"
	"io"
	"os"
	"strconv"
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
			return nil, p.formatError(err)
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
			return nil, p.formatError(err)
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
		p.current.AppendByte(0x9C)
		return nil
	case equalWord(tok, "popf"):
		p.current.AppendByte(0x9D)
		return nil
	case equalWord(tok, "pusha"):
		p.current.AppendByte(0x60)
		return nil
	case equalWord(tok, "popa"):
		p.current.AppendByte(0x61)
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
		p.current.AppendByte(0x90)
		return nil
	case equalWord(tok, "hlt"):
		p.current.AppendByte(0xF4)
		return nil
	case equalWord(tok, "cli"):
		p.current.AppendByte(0xFA)
		return nil
	case equalWord(tok, "sti"):
		p.current.AppendByte(0xFB)
		return nil
	case equalWord(tok, "iret"):
		p.current.AppendByte(0xCF)
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
		p.current.AppendBytes([]byte{0x0F, 0x0B})
		return nil
	case equalWord(tok, "cpuid"):
		p.current.AppendBytes([]byte{0x0F, 0xA2})
		return nil
	case equalWord(tok, "rdtsc"):
		p.current.AppendBytes([]byte{0x0F, 0x31})
		return nil
	case equalWord(tok, "syscall"):
		p.current.AppendBytes([]byte{0x0F, 0x05})
		return nil
	case equalWord(tok, "sysret"):
		p.current.AppendBytes([]byte{0x0F, 0x07})
		return nil
	case equalWord(tok, "swapgs"):
		p.current.AppendBytes([]byte{0x0F, 0x01, 0xF8})
		return nil
	case equalWord(tok, "ret"):
		p.current.AppendByte(0xC3)
		return nil
	case equalWord(tok, "retf"):
		p.current.AppendByte(0xCB)
		return nil
	case equalWord(tok, "enter"):
		tok2, rest2 := readToken(rest)
		bytes, err := parseNumber(tok2)
		if err != nil {
			return err
		}
		level, _ := parseNumber(trimSpace(rest2))
		p.current.AppendByte(0xC8)
		p.current.AppendByte(byte(bytes))
		p.current.AppendByte(byte(bytes >> 8))
		p.current.AppendByte(byte(level))
		return nil
	case equalWord(tok, "leave"):
		p.current.AppendByte(0xC9)
		return nil
	case equalWord(tok, "pause"):
		p.current.AppendBytes([]byte{0xF3, 0x90})
		return nil
	case equalWord(tok, "lock"):
		p.current.AppendByte(0xF0)
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

func (p *parser) formatError(err error) error {
	if err == nil {
		return nil
	}
	lineNum := bytes.Count(p.src[:p.pos], []byte{'\n'})
	bptr := emitterBufferPool.Get().(*[]byte)
	buf := *bptr
	buf = buf[:0]
	buf = append(buf, "assembler: line "...)
	buf = strconv.AppendInt(buf, int64(lineNum+1), 10)
	buf = append(buf, ": "...)
	buf = append(buf, err.Error()...)
	s := string(buf)
	emitterBufferPool.Put(bptr)
	return errors.New(s)
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
			if p.symbols[i].value > uint64(^uint32(0)) {
				return nil, errors.New("symbol value overflow for 32-bit ELF")
			}
			writeElf32SymAt(out, off, p.symbols[i].nameOffset, info, 0, p.symbols[i].shndx, uint32(p.symbols[i].value), 0)
			off += 16
		}
		for i := 0; i < p.symCount; i++ {
			if p.symbols[i].bind == stBindLocal {
				continue
			}
			info := byte(p.symbols[i].bind<<4 | p.symbols[i].typ)
			if p.symbols[i].value > uint64(^uint32(0)) {
				return nil, errors.New("symbol value overflow for 32-bit ELF")
			}
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
	if p.bss.size > uint64(^uint64(0)) {
		return
	}
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
	if p.bss.size > uint64(^uint32(0)) {
		return
	}
	writeELF32Section(sec[elf32ShdrSize*3:elf32ShdrSize*4], uint32(offsetOfNameInShstr(p.bss.name, shstr)), shTypeNoBits, p.bss.flags, 0, 0, uint32(p.bss.size), 0, 0, p.bss.align, 0)
	writeELF32Section(sec[elf32ShdrSize*4:elf32ShdrSize*5], uint32(offsetOfNameInShstr(nameShstrtab, shstr)), shTypeStrTab, 0, 0, shstrtabOffset, shstrtabSize, 0, 0, 1, 0)
	writeELF32Section(sec[elf32ShdrSize*5:elf32ShdrSize*6], uint32(offsetOfNameInShstr(nameSymtab, shstr)), shTypeSymTab, 0, 0, symtabOffset, symtabSize, 6, uint32(symbolLocalCount(p)+1), 8, 16)
	writeELF32Section(sec[elf32ShdrSize*6:elf32ShdrSize*7], uint32(offsetOfNameInShstr(nameStrtab, shstr)), shTypeStrTab, 0, 0, strtabOffset, strtabSize, 0, 0, 1, 0)
	writeELF32Section(sec[elf32ShdrSize*7:elf32ShdrSize*8], uint32(offsetOfNameInShstr([]byte(".rela.text"), shstr)), shTypeRela, 0, 0, relaOffset, relaSize, 5, 1, 4, 12)
}
