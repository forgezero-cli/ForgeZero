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
	"debug/elf"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAssembleRawELF(t *testing.T) {
	old := OutputFormat
	defer func() { OutputFormat = old }()
	OutputFormat = "elf64"

	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.asm", `section .text
global start
start:
    db 0x90, 0x90, 0x90
`)
	obj := filepath.Join(dir, "test.o")

	if err := Assemble(context.Background(), src, obj, false, false, "raw"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(obj); err != nil {
		t.Fatal(err)
	}
}

func TestAssembleRawELFCallAndJump(t *testing.T) {
	old := OutputFormat
	defer func() { OutputFormat = old }()
	OutputFormat = "elf64"

	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.asm", `global _start
section .text
_start:
    mov rdi, 5
    call factorial_iterative
    mov rdi, rax
    mov rax, 60
    syscall
factorial_iterative:
    mov rax, 1
    cmp rdi, 1
    jle .done
.loop:
    imul rax, rdi
    dec rdi
    cmp rdi, 1
    jg .loop
.done:
    ret
`)
	obj := filepath.Join(dir, "test.o")

	if err := Assemble(context.Background(), src, obj, false, false, "raw"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(obj)
	if err != nil {
		t.Fatal(err)
	}
	f, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	syms, err := f.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	var foundStart, foundFactorial bool
	for _, sym := range syms {
		switch sym.Name {
		case "_start":
			if elf.SymBind(sym.Info>>4) == elf.STB_GLOBAL {
				foundStart = true
			}
		case "factorial_iterative":
			foundFactorial = true
		case "mov", "call", "cmp", "jle", "jg", "imul", "dec", "syscall":
			t.Fatalf("unexpected instruction symbol in symtab: %s", sym.Name)
		}
	}
	if !foundStart {
		t.Fatal("missing _start symbol")
	}
	if !foundFactorial {
		t.Fatal("missing factorial_iterative symbol")
	}
}

func TestAssembleRawBinary(t *testing.T) {
	old := OutputFormat
	defer func() { OutputFormat = old }()
	OutputFormat = "bin"

	oldForce := ForceInternalAsm
	defer func() { ForceInternalAsm = oldForce }()
	ForceInternalAsm = true

	dir := t.TempDir()
	src := writeTempFile(t, dir, "boot.asm", `section .text
    db 0x90
    resb 509
    dw 0xAA55
`)
	out := filepath.Join(dir, "boot.bin")

	if err := Assemble(context.Background(), src, out, false, false, "raw"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 512 {
		t.Fatalf("binary image size = %d, want 512", info.Size())
	}
}

func TestAssembleUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.txt", "not assembly")
	obj := filepath.Join(dir, "test.o")

	err := Assemble(context.Background(), src, obj, false, false, "raw")
	if err == nil {
		t.Fatal("expected unsupported file extension error")
	}
}

func TestAssembleWasmUnsupported(t *testing.T) {
	old := Target
	defer func() { Target = old }()
	Target = "wasm32-unknown-unknown"

	dir := t.TempDir()
	src := writeTempFile(t, dir, "test.asm", `section .text
global start
start:
    db 0x90
`)
	obj := filepath.Join(dir, "test.o")

	err := Assemble(context.Background(), src, obj, false, false, "raw")
	if err == nil {
		t.Fatal("expected wasm assembly error")
	}
}

func TestEmitZeroAllocs_x86_64(t *testing.T) {
	profile := TargetProfileFromTarget("x86_64-linux-gnu")
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
	p.text.data = []byte{0x90, 0x90, 0x90}
	p.data.data = []byte{1, 2, 3, 4}

	for i := 0; i < 20; i++ {
		if _, err := p.emit(profile); err != nil {
			t.Fatalf("warmup emit failed: %v", err)
		}
	}

	ehSize := 64
	if profile.elfClass == elfClass32 {
		ehSize = 52
	}
	shdrSize := elf64ShdrSize
	if profile.elfClass == elfClass32 {
		shdrSize = elf32ShdrSize
	}
	shstrtabLen := 1 + (len(nameEmpty) + 1) + (len(p.text.name) + 1) + (len(p.data.name) + 1) + (len(p.bss.name) + 1) + (len(nameShstrtab) + 1) + (len(nameSymtab) + 1) + (len(nameStrtab) + 1)
	strtabLen := 1
	for i := 0; i < p.symCount; i++ {
		strtabLen += len(p.symbols[i].name) + 1
	}
	symEntrySize := 24
	if profile.elfClass == elfClass32 {
		symEntrySize = 16
	}
	symtabSize := (p.symCount + 1) * symEntrySize
	need := ehSize + len(p.text.data) + len(p.data.data) + symtabSize + strtabLen + shstrtabLen + shdrSize*7
	need += 1024
	b := make([]byte, 0, need)
	reusableOut = &b
	defer func() { reusableOut = nil }()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	for i := 0; i < 50; i++ {
		if _, err := p.emit(profile); err != nil {
			t.Fatalf("emit failed: %v", err)
		}
	}
	runtime.ReadMemStats(&m2)
	allocs := (m2.Mallocs - m1.Mallocs) / 50
	if allocs != 0 {
		t.Fatalf("allocs per run = %d, want 0", allocs)
	}
}
