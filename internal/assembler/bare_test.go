package assembler

import (
	"bytes"
	"context"
	"debug/elf"
	"os"
	"path/filepath"
	"testing"
)

func TestFormatFlagBin(t *testing.T) {
	oldFmt := OutputFormat
	oldTgt := Target
	defer func() {
		OutputFormat = oldFmt
		Target = oldTgt
	}()
	OutputFormat = "bin"
	Target = "x86_64-linux-gnu"
	if got := formatFlagForTarget(); got != "-fbin" {
		t.Fatalf("formatFlagForTarget() = %q, want -fbin", got)
	}
}

func TestIsBareMetalTarget(t *testing.T) {
	old := Target
	defer func() { Target = old }()
	Target = "x86_64-unknown-elf"
	if !IsBareMetalTarget() {
		t.Fatal("expected bare-metal for unknown-elf")
	}
	Target = "cortex-m3"
	if !IsBareMetalTarget() {
		t.Fatal("expected bare-metal for cortex-m3")
	}
	Target = "x86_64-linux-gnu"
	if IsBareMetalTarget() {
		t.Fatal("linux-gnu is not bare-metal")
	}
}

func TestAssembleNASMBootSector(t *testing.T) {
	oldFmt := OutputFormat
	defer func() { OutputFormat = oldFmt }()
	OutputFormat = "bin"
	dir := t.TempDir()
	src := filepath.Join(dir, "boot.asm")
	asm := `section .text
start:
    db 0x90
    resb 509
    dw 0xAA55
`
	if err := os.WriteFile(src, []byte(asm), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "boot.bin")
	if err := Assemble(context.Background(), src, out, false, false, "raw"); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() != 512 {
		t.Fatalf("boot sector size = %d, want 512", st.Size())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 512 {
		t.Fatal("short boot image")
	}
	if data[510] != 0x55 || data[511] != 0xAA {
		t.Fatalf("missing boot signature: %02x %02x", data[510], data[511])
	}
}

func TestEmitSourceObjectELF(t *testing.T) {
	src := []byte("start:\n    db 0x90\n    global foo\nfoo:\n    db 0xCC\n")
	out, err := emitSourceObject(src, selfTargetProfile("x86_64-linux-gnu"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := elf.NewFile(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sect := f.Section(".text")
	if sect == nil {
		t.Fatal("missing .text section")
	}
	if sect.Size != 2 {
		t.Fatalf(".text size = %d, want 2", sect.Size)
	}
	syms, err := f.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, sym := range syms {
		found[sym.Name] = true
	}
	if !found["start"] {
		t.Fatal("missing start symbol")
	}
	if !found["foo"] {
		t.Fatal("missing foo symbol")
	}
}

func TestSkipLinker(t *testing.T) {
	old := OutputFormat
	defer func() { OutputFormat = old }()
	OutputFormat = "bin"
	if !SkipLinker() {
		t.Fatal("expected skip linker for bin format")
	}
	OutputFormat = "elf64"
	if SkipLinker() {
		t.Fatal("elf64 should not skip linker")
	}
}
