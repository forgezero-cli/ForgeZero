package assembler

import (
	"context"
	"fmt"
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
	shTypeNull        = 0
	shTypeProgBits    = 1
	shTypeSymTab      = 2
	shTypeStrTab      = 3
	shTypeNoBits      = 8
	shFlagAlloc       = 0x2
	shFlagExecInstr   = 0x4
	shFlagWrite       = 0x1
	stBindLocal       = 0
	stBindGlobal      = 1
	stTypeNotype      = 0
	shnUnDef          = 0
)

var emitterBufferPool = sync.Pool{New: func() any { b := make([]byte, 0, 65536); return &b }}

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
		return nil, nil, fmt.Errorf("source too large for internal emitter")
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
	symbols  [128]symbolState
	symCount int
}

func emitSourceObject(src []byte, profile targetEmitterProfile) ([]byte, error) {
	p := parser{}
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
	return p.emit(profile)
}

func emitSourceRaw(src []byte, profile targetEmitterProfile) ([]byte, error) {
	p := parser{}
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
	outPtr := emitterBufferPool.Get().(*[]byte)
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
	start := p.pos
	for p.pos < len(p.src) && p.src[p.pos] != '\n' {
		p.pos++
	}
	line := p.src[start:p.pos]
	if p.pos < len(p.src) && p.src[p.pos] == '\n' {
		p.pos++
	}
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
	}
	return fmt.Errorf("unsupported assembler directive: %s", string(tok))
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
	return fmt.Errorf("unknown section: %s", string(name))
}

func (p *parser) defineLabel(name []byte) error {
	if len(name) == 0 {
		return fmt.Errorf("empty label")
	}
	return p.addSymbol(name, currentOffset(p.current), sectionIndex(p.current), stBindLocal)
}

func (p *parser) defineGlobal(name []byte) error {
	if len(name) == 0 {
		return fmt.Errorf("empty global")
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
		return fmt.Errorf("symbol table capacity exceeded")
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
	items := splitArgs(rest)
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
	items := splitArgs(rest)
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
	shstrtab := buildStringTable([][]byte{[]byte(""), p.text.name, p.data.name, p.bss.name, []byte(".shstrtab"), []byte(".symtab"), []byte(".strtab")})
	strtab := buildSymbolStringTable(p)
	symtab := buildSymbolTable(p, strtab, profile)
	outPtr := emitterBufferPool.Get().(*[]byte)
	out := *outPtr
	out = out[:0]
	headerSize := 64
	if profile.elfClass == elfClass32 {
		headerSize = 52
	}
	for i := 0; i < headerSize; i++ {
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
	symtabOffset := alignOutOffset(len(out), align)
	out = alignOut(out, align)
	symtabOffset = len(out)
	out = append(out, symtab...)
	strtabOffset := len(out)
	out = append(out, strtab...)
	shstrtabOffset := len(out)
	out = append(out, shstrtab...)
	shOff := alignOutOffset(len(out), uint64(headerSize))
	out = alignOut(out, uint64(headerSize))
	numSections := 7
	shSize := headerSize
	for i := 0; i < numSections; i++ {
		for j := 0; j < shSize; j++ {
			out = append(out, 0)
		}
	}
	if profile.elfClass == elfClass64 {
		populateELF64Header(out, profile, uint64(shOff), uint16(shSize), uint16(numSections), 4)
		populateELF64SectionHeaders(out[shOff:], out, p, uint64(textOffset), uint64(dataOffset), uint64(symtabOffset), uint64(strtabOffset), uint64(shstrtabOffset), uint64(len(symtab)), uint64(len(strtab)), uint64(len(shstrtab)))
	} else {
		populateELF32Header(out, profile, uint32(shOff), uint16(shSize), uint16(numSections), 4)
		populateELF32SectionHeaders(out[shOff:], out, p, uint32(textOffset), uint32(dataOffset), uint32(symtabOffset), uint32(strtabOffset), uint32(shstrtabOffset), uint32(len(symtab)), uint32(len(strtab)), uint32(len(shstrtab)))
	}
	*outPtr = out
	return out, nil
}

func buildStringTable(names [][]byte) []byte {
	outPtr := emitterBufferPool.Get().(*[]byte)
	out := *outPtr
	out = out[:0]
	out = append(out, 0)
	for _, name := range names {
		out = append(out, name...)
		out = append(out, 0)
	}
	*outPtr = out
	return out
}

func buildSymbolStringTable(p *parser) []byte {
	outPtr := emitterBufferPool.Get().(*[]byte)
	out := *outPtr
	out = out[:0]
	out = append(out, 0)
	for i := 0; i < p.symCount; i++ {
		p.symbols[i].nameOffset = uint32(len(out))
		out = append(out, p.symbols[i].name...)
		out = append(out, 0)
	}
	*outPtr = out
	return out
}

func buildSymbolTable(p *parser, strtab []byte, profile targetEmitterProfile) []byte {
	outPtr := emitterBufferPool.Get().(*[]byte)
	out := *outPtr
	out = out[:0]
	out = appendElf64Sym(out, 0, 0, 0, 0, 0, 0)
	if profile.elfClass == elfClass32 {
		for i := 0; i < p.symCount; i++ {
			out = appendElf32Sym(out, p.symbols[i].nameOffset, p.symbols[i].bind<<4|p.symbols[i].typ, 0, p.symbols[i].shndx, uint32(p.symbols[i].value), 0)
		}
	} else {
		for i := 0; i < p.symCount; i++ {
			out = appendElf64Sym(out, p.symbols[i].nameOffset, p.symbols[i].bind<<4|p.symbols[i].typ, 0, p.symbols[i].shndx, p.symbols[i].value, 0)
		}
	}
	*outPtr = out
	return out
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

func appendElf32Sym(out []byte, name uint32, info byte, other byte, shndx uint16, value uint32, size uint32) []byte {
	out = appendUint32(out, name)
	out = appendByte(out, info)
	out = appendByte(out, other)
	out = appendUint16(out, shndx)
	out = appendUint32(out, value)
	out = appendUint32(out, size)
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
	out = out[:0]
	out = appendUint32(out, name)
	out = appendUint32(out, typ)
	out = appendUint64(out, flags)
	out = appendUint64(out, addr)
	out = appendUint64(out, offset)
	out = appendUint64(out, size)
	out = appendUint32(out, link)
	out = appendUint32(out, info)
	out = appendUint64(out, addralign)
	out = appendUint64(out, entsize)
}

func writeELF32Section(out []byte, name uint32, typ uint32, flags uint32, addr uint32, offset uint32, size uint32, link uint32, info uint32, addralign uint32, entsize uint32) {
	out = out[:0]
	out = appendUint32(out, name)
	out = appendUint32(out, typ)
	out = appendUint32(out, flags)
	out = appendUint32(out, addr)
	out = appendUint32(out, offset)
	out = appendUint32(out, size)
	out = appendUint32(out, link)
	out = appendUint32(out, info)
	out = appendUint32(out, addralign)
	out = appendUint32(out, entsize)
}

func populateELF64Header(out []byte, profile targetEmitterProfile, shoff uint64, shentsize uint16, shnum uint16, shstrndx uint16) {
	copy(out[0:4], []byte{0x7f, 'E', 'L', 'F'})
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
	out = appendUint16(out[:16], elfTypeRel)
	out = appendUint16(out[:18], profile.eMachine)
	out = appendUint32(out[:20], elfVersion1)
	out = appendUint64(out[:24], 0)
	out = appendUint64(out[:32], 0)
	out = appendUint64(out[:40], shoff)
	out = appendUint32(out[:48], 0)
	out = appendUint16(out[:52], 64)
	out = appendUint16(out[:54], 0)
	out = appendUint16(out[:56], 0)
	out = appendUint16(out[:58], shentsize)
	out = appendUint16(out[:60], shnum)
	out = appendUint16(out[:62], shstrndx)
}

func populateELF32Header(out []byte, profile targetEmitterProfile, shoff uint32, shentsize uint16, shnum uint16, shstrndx uint16) {
	copy(out[0:4], []byte{0x7f, 'E', 'L', 'F'})
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
	out = appendUint16(out[:16], elfTypeRel)
	out = appendUint16(out[:18], profile.eMachine)
	out = appendUint32(out[:20], elfVersion1)
	out = appendUint32(out[:24], 0)
	out = appendUint32(out[:28], 0)
	out = appendUint32(out[:32], shoff)
	out = appendUint32(out[:36], 0)
	out = appendUint16(out[:40], 52)
	out = appendUint16(out[:42], 0)
	out = appendUint16(out[:44], 0)
	out = appendUint16(out[:46], shentsize)
	out = appendUint16(out[:48], shnum)
	out = appendUint16(out[:50], shstrndx)
}

func populateELF64SectionHeaders(sec []byte, full []byte, p *parser, textOffset, dataOffset, symtabOffset, strtabOffset, shstrtabOffset uint64, symtabSize, strtabSize, shstrtabSize uint64) {
	shstr := full[shstrtabOffset : shstrtabOffset+shstrtabSize]
	writeELF64Section(sec[0:], 0, shTypeNull, 0, 0, 0, 0, 0, 0, 0, 0)
	writeELF64Section(sec[64:], uint32(offsetOfNameInShstr(p.text.name, shstr)), shTypeProgBits, uint64(p.text.flags), 0, uint64(textOffset), uint64(len(p.text.data)), 0, 0, uint64(p.text.align), 0)
	writeELF64Section(sec[128:], uint32(offsetOfNameInShstr(p.data.name, shstr)), shTypeProgBits, uint64(p.data.flags), 0, uint64(dataOffset), uint64(len(p.data.data)), 0, 0, uint64(p.data.align), 0)
	writeELF64Section(sec[192:], uint32(offsetOfNameInShstr(p.bss.name, shstr)), shTypeNoBits, uint64(p.bss.flags), 0, 0, p.bss.size, 0, 0, uint64(p.bss.align), 0)
	writeELF64Section(sec[256:], uint32(offsetOfNameInShstr([]byte(".shstrtab"), shstr)), shTypeStrTab, 0, 0, shstrtabOffset, shstrtabSize, 0, 0, 1, 0)
	writeELF64Section(sec[320:], uint32(offsetOfNameInShstr([]byte(".symtab"), shstr)), shTypeSymTab, 0, 0, symtabOffset, symtabSize, 6, uint32(symbolLocalCount(p)+1), 8, 24)
	writeELF64Section(sec[384:], uint32(offsetOfNameInShstr([]byte(".strtab"), shstr)), shTypeStrTab, 0, 0, strtabOffset, strtabSize, 0, 0, 1, 0)
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

func populateELF32SectionHeaders(sec []byte, full []byte, p *parser, textOffset, dataOffset, symtabOffset, strtabOffset, shstrtabOffset uint32, symtabSize, strtabSize, shstrtabSize uint32) {
	shstr := full[shstrtabOffset : shstrtabOffset+shstrtabSize]
	writeELF32Section(sec[0:], 0, shTypeNull, 0, 0, 0, 0, 0, 0, 0, 0)
	writeELF32Section(sec[40:], uint32(offsetOfNameInShstr(p.text.name, shstr)), shTypeProgBits, p.text.flags, 0, textOffset, uint32(len(p.text.data)), 0, 0, p.text.align, 0)
	writeELF32Section(sec[80:], uint32(offsetOfNameInShstr(p.data.name, shstr)), shTypeProgBits, p.data.flags, 0, dataOffset, uint32(len(p.data.data)), 0, 0, p.data.align, 0)
	writeELF32Section(sec[120:], uint32(offsetOfNameInShstr(p.bss.name, shstr)), shTypeNoBits, p.bss.flags, 0, 0, uint32(p.bss.size), 0, 0, p.bss.align, 0)
	writeELF32Section(sec[160:], uint32(offsetOfNameInShstr([]byte(".shstrtab"), shstr)), shTypeStrTab, 0, 0, shstrtabOffset, shstrtabSize, 0, 0, 1, 0)
	writeELF32Section(sec[200:], uint32(offsetOfNameInShstr([]byte(".symtab"), shstr)), shTypeSymTab, 0, 0, symtabOffset, symtabSize, 6, uint32(symbolLocalCount(p)+1), 8, 16)
	writeELF32Section(sec[240:], uint32(offsetOfNameInShstr([]byte(".strtab"), shstr)), shTypeStrTab, 0, 0, strtabOffset, strtabSize, 0, 0, 1, 0)
}

func offsetOfName(name []byte, shstrtabOffset uint64, full []byte) uint32 {
	base := int(shstrtabOffset)
	if base < 0 || base >= len(full) {
		return 0
	}
	for i := base; i+len(name) <= len(full); i++ {
		if matchBytes(full[i:i+len(name)], name) {
			return uint32(i - base)
		}
	}
	return 0
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

func splitArgs(data []byte) [][]byte {
	var parts [][]byte
	start := 0
	for i := 0; i <= len(data); i++ {
		if i == len(data) || data[i] == ',' {
			parts = append(parts, data[start:i])
			start = i + 1
		}
	}
	return parts
}

func trimSpace(data []byte) []byte {
	start := 0
	end := len(data)
	for start < end && (data[start] == ' ' || data[start] == '\t' || data[start] == '\r' || data[start] == '\n') {
		start++
	}
	for end > start && (data[end-1] == ' ' || data[end-1] == '\t' || data[end-1] == '\r' || data[end-1] == '\n') {
		end--
	}
	return data[start:end]
}

func removeComment(line []byte) []byte {
	for i := 0; i < len(line); i++ {
		if line[i] == ';' || line[i] == '#' {
			return line[:i]
		}
	}
	return line
}

func parseString(token []byte) ([]byte, error) {
	if len(token) < 2 {
		return nil, fmt.Errorf("invalid string")
	}
	quote := token[0]
	if token[len(token)-1] != quote {
		return nil, fmt.Errorf("unterminated string")
	}
	return token[1 : len(token)-1], nil
}

func parseNumber(token []byte) (uint64, error) {
	if len(token) == 0 {
		return 0, fmt.Errorf("empty number")
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
			return 0, fmt.Errorf("invalid digit %c", c)
		}
		if digit >= uint64(base) {
			return 0, fmt.Errorf("invalid digit %c for base %d", c, base)
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
