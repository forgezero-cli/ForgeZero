package assembler

import (
	"bytes"
	"testing"
)

func TestParseLineSectionDirective(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.flags = shFlagAlloc | shFlagExecInstr
	p.text.align = 16
	p.data.name = []byte(".data")
	p.data.flags = shFlagAlloc | shFlagWrite
	p.data.align = 16
	p.bss.name = []byte(".bss")
	p.bss.flags = shFlagAlloc | shFlagWrite
	p.bss.align = 16
	p.current = &p.text

	tests := []struct {
		line []byte
		want string
	}{
		{[]byte("section .text"), ".text"},
		{[]byte("section .data"), ".data"},
		{[]byte("section .bss"), ".bss"},
		{[]byte("text"), ".text"},
		{[]byte("data"), ".data"},
		{[]byte("bss"), ".bss"},
	}
	for _, tt := range tests {
		if err := p.parseLine(tt.line); err != nil {
			t.Fatal(err)
		}
		var got string
		switch p.current {
		case &p.text:
			got = ".text"
		case &p.data:
			got = ".data"
		case &p.bss:
			got = ".bss"
		}
		if got != tt.want {
			t.Errorf("parseLine(%q) current = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestParseLineGlobalDirective(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.current = &p.text
	if err := p.parseLine([]byte("global _start")); err != nil {
		t.Fatal(err)
	}
	if p.symCount != 1 {
		t.Fatalf("symCount = %d, want 1", p.symCount)
	}
	if string(p.symbols[0].name) != "_start" {
		t.Errorf("symbol name = %s, want _start", p.symbols[0].name)
	}
	if p.symbols[0].bind != stBindGlobal {
		t.Errorf("bind = %d, want global", p.symbols[0].bind)
	}
	if p.symbols[0].shndx != shnUnDef {
		t.Errorf("shndx = %d, want UNDEF", p.symbols[0].shndx)
	}
}

func TestParseLineDbDirective(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("db 0x90, 0x90, 0x90")); err != nil {
		t.Fatal(err)
	}
	want := []byte{0x90, 0x90, 0x90}
	if !bytes.Equal(p.text.data, want) {
		t.Errorf("db = %x, want %x", p.text.data, want)
	}
}

func TestParseLineDwDirective(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("dw 0x1234, 0x5678")); err != nil {
		t.Fatal(err)
	}
	want := []byte{0x34, 0x12, 0x78, 0x56}
	if !bytes.Equal(p.text.data, want) {
		t.Errorf("dw = %x, want %x", p.text.data, want)
	}
}

func TestParseLineDdDirective(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("dd 0x12345678")); err != nil {
		t.Fatal(err)
	}
	want := []byte{0x78, 0x56, 0x34, 0x12}
	if !bytes.Equal(p.text.data, want) {
		t.Errorf("dd = %x, want %x", p.text.data, want)
	}
}

func TestParseLineDqDirective(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("dq 0x1234567890ABCDEF")); err != nil {
		t.Fatal(err)
	}
	want := []byte{0xEF, 0xCD, 0xAB, 0x90, 0x78, 0x56, 0x34, 0x12}
	if !bytes.Equal(p.text.data, want) {
		t.Errorf("dq = %x, want %x", p.text.data, want)
	}
}

func TestParseLineResbDirective(t *testing.T) {
	p := &parser{}
	p.bss.name = []byte(".bss")
	p.bss.flags = shFlagAlloc | shFlagWrite
	p.current = &p.bss
	if err := p.parseLine([]byte("resb 10")); err != nil {
		t.Fatal(err)
	}
	if p.bss.size != 10 {
		t.Errorf("bss.size = %d, want 10", p.bss.size)
	}
}

func TestParseLineReswDirective(t *testing.T) {
	p := &parser{}
	p.bss.name = []byte(".bss")
	p.current = &p.bss
	if err := p.parseLine([]byte("resw 5")); err != nil {
		t.Fatal(err)
	}
	if p.bss.size != 10 {
		t.Errorf("bss.size = %d, want 10", p.bss.size)
	}
}

func TestParseLineResdDirective(t *testing.T) {
	p := &parser{}
	p.bss.name = []byte(".bss")
	p.current = &p.bss
	if err := p.parseLine([]byte("resd 3")); err != nil {
		t.Fatal(err)
	}
	if p.bss.size != 12 {
		t.Errorf("bss.size = %d, want 12", p.bss.size)
	}
}

func TestParseLineResqDirective(t *testing.T) {
	p := &parser{}
	p.bss.name = []byte(".bss")
	p.current = &p.bss
	if err := p.parseLine([]byte("resq 2")); err != nil {
		t.Fatal(err)
	}
	if p.bss.size != 16 {
		t.Errorf("bss.size = %d, want 16", p.bss.size)
	}
}

func TestParseLineAlignDirective(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = []byte{0x90}
	p.current = &p.text
	if err := p.parseLine([]byte("align 16")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) != 16 {
		t.Errorf("text size = %d, want 16", len(p.text.data))
	}
	for i := 1; i < 16; i++ {
		if p.text.data[i] != 0 {
			t.Errorf("padding byte %d = %d, want 0", i, p.text.data[i])
		}
	}
}

func TestParseLineBitsDirective(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("bits 64")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) != 0 {
		t.Errorf("bits directive should not emit code, got %d bytes", len(p.text.data))
	}
}

func TestParseLineOrgDirective(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("org 0x1000")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) != 0 {
		t.Errorf("org directive should not emit code, got %d bytes", len(p.text.data))
	}
}

func TestSwitchSectionError(t *testing.T) {
	p := &parser{}
	if err := p.switchSection([]byte(".unknown")); err == nil {
		t.Error("switchSection .unknown should fail")
	}
}

func TestDefineLabelEmpty(t *testing.T) {
	p := &parser{}
	if err := p.defineLabel([]byte("")); err == nil {
		t.Error("defineLabel empty should fail")
	}
}

func TestDefineGlobalEmpty(t *testing.T) {
	p := &parser{}
	if err := p.defineGlobal([]byte("")); err == nil {
		t.Error("defineGlobal empty should fail")
	}
}

func TestCurrentOffset(t *testing.T) {
	text := &sectionState{data: []byte{0x90, 0x90}}
	if off := currentOffset(text); off != 2 {
		t.Errorf("currentOffset(text) = %d, want 2", off)
	}
	bss := &sectionState{name: []byte(".bss"), size: 16}
	if off := currentOffset(bss); off != 16 {
		t.Errorf("currentOffset(bss) = %d, want 16", off)
	}
}

func TestAddSymbolUpdate(t *testing.T) {
	p := &parser{}
	p.symbols[0] = symbolState{name: []byte("foo"), value: 0, shndx: shnUnDef, bind: stBindLocal}
	p.symCount = 1
	if err := p.addSymbol([]byte("foo"), 42, 1, stBindGlobal); err != nil {
		t.Fatal(err)
	}
	if p.symbols[0].value != 42 {
		t.Errorf("value = %d, want 42", p.symbols[0].value)
	}
	if p.symbols[0].shndx != 1 {
		t.Errorf("shndx = %d, want 1", p.symbols[0].shndx)
	}
	if p.symbols[0].bind != stBindGlobal {
		t.Errorf("bind = %d, want global", p.symbols[0].bind)
	}
}

func TestAddSymbolCapacity(t *testing.T) {
	p := &parser{}
	p.symCount = len(p.symbols)
	if err := p.addSymbol([]byte("test"), 0, shnUnDef, stBindLocal); err == nil {
		t.Error("addSymbol should fail on full table")
	}
}

func TestFindSymbol(t *testing.T) {
	p := &parser{}
	p.symbols[0] = symbolState{name: []byte("_start")}
	p.symCount = 1
	if idx := p.findSymbol([]byte("_start")); idx != 0 {
		t.Errorf("findSymbol(_start) = %d, want 0", idx)
	}
	if idx := p.findSymbol([]byte("missing")); idx != -1 {
		t.Errorf("findSymbol(missing) = %d, want -1", idx)
	}
}

func TestEmitBytesMixed(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.emitBytes([]byte(`0x90, "hello", 0xCC`)); err != nil {
		t.Fatal(err)
	}
	want := []byte{0x90, 'h', 'e', 'l', 'l', 'o', 0xCC}
	if !bytes.Equal(p.text.data, want) {
		t.Errorf("emitBytes mixed = %x, want %x", p.text.data, want)
	}
}

func TestEmitWordsWidths(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.emitWords([]byte("0x1234, 0x5678"), 2); err != nil {
		t.Fatal(err)
	}
	want := []byte{0x34, 0x12, 0x78, 0x56}
	if !bytes.Equal(p.text.data, want) {
		t.Errorf("emitWords width 2 = %x, want %x", p.text.data, want)
	}
	p.text.data = p.text.data[:0]
	if err := p.emitWords([]byte("0x12345678"), 4); err != nil {
		t.Fatal(err)
	}
	want = []byte{0x78, 0x56, 0x34, 0x12}
	if !bytes.Equal(p.text.data, want) {
		t.Errorf("emitWords width 4 = %x, want %x", p.text.data, want)
	}
}

func TestGetCompiler(t *testing.T) {
	oldTarget := Target
	defer func() { Target = oldTarget }()
	Target = "x86_64-linux-gnu"
	if got := getCompiler("test.m"); got != "clang" {
		t.Errorf("getCompiler .m = %q, want clang", got)
	}
	if got := getCompiler("test.c"); got != "gcc" {
		t.Errorf("getCompiler .c = %q, want gcc", got)
	}
	Target = "arm-linux-gnueabihf"
	if got := getCompiler("test.c"); got != "arm-linux-gnueabihf-gcc" {
		t.Errorf("getCompiler arm .c = %q, want arm-linux-gnueabihf-gcc", got)
	}
}

func TestParseLineUnsupported(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.current = &p.text
	if err := p.parseLine([]byte("unknown_instruction rax")); err == nil {
		t.Error("parseLine unknown should fail")
	}
}
