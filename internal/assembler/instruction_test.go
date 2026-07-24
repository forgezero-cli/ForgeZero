/*
 * Copyright (c) 2026 ForgeZero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package assembler

import (
	"testing"
)

func TestEmitArithAdd(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("add rax, rbx")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitArithSub(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("sub rax, rbx")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitArithAnd(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("and rax, rbx")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitArithOr(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("or rax, rbx")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitArithXor(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("xor rax, rbx")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitArithImm(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("add rax, 5")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitInc(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("inc rax")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitDec(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("dec rax")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitNeg(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("neg rax")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitNot(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("not rax")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitLea(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("lea rax, [rbx+4]")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitMovRegImm(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("mov rax, 5")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitMovRegReg(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("mov rax, rbx")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitCmpRegReg(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("cmp rax, rbx")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitCmpImm(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("cmp rax, 5")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitTestRegReg(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("test rax, rbx")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitTestImm(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("test rax, 5")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitShiftShl(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("shl rax, 1")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitShiftShr(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("shr rax, 2")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitShiftSar(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("sar rax, 3")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitMul(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("mul rax")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitDiv(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("div rax")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitIdiv(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("idiv rax")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitImulRegReg(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("imul rax, rbx")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitXchgRegReg(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("xchg rax, rbx")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitMovzx(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("movzx rax, [rbx]")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitMovsx(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("movsx rax, [rbx]")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitPushReg(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("push rax")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitPushImm(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("push 5")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitImmOverflowRejected(t *testing.T) {
	for _, instr := range []string{"add rax, 4294967296", "push 4294967296", "cmp rax, 4294967296"} {
		p := &parser{}
		p.text.name = []byte(".text")
		p.text.data = make([]byte, 0, 1024)
		p.current = &p.text
		if err := p.parseLine([]byte(instr)); err == nil {
			t.Fatalf("expected error for %q", instr)
		}
	}
}

func TestEmitPop(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("pop rax")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("no code generated")
	}
}

func TestEmitPushfPopf(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("pushf")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("pushf no code")
	}
	p.text.data = p.text.data[:0]
	if err := p.parseLine([]byte("popf")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("popf no code")
	}
}

func TestEmitPushaPopa(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("pusha")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("pusha no code")
	}
	p.text.data = p.text.data[:0]
	if err := p.parseLine([]byte("popa")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("popa no code")
	}
}

func TestEmitJccJe(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("_start:")); err != nil {
		t.Fatal(err)
	}
	p.text.data = p.text.data[:0]
	if err := p.parseLine([]byte("je _start")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("je no code")
	}
	if len(p.relocs) == 0 {
		t.Error("je missing relocation")
	}
}

func TestEmitJccJne(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("_start:")); err != nil {
		t.Fatal(err)
	}
	p.text.data = p.text.data[:0]
	if err := p.parseLine([]byte("jne _start")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("jne no code")
	}
}

func TestEmitJumpJle(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("_start:")); err != nil {
		t.Fatal(err)
	}
	p.text.data = p.text.data[:0]
	if err := p.parseLine([]byte("jle _start")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("jle no code")
	}
}

func TestEmitJumpJg(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("_start:")); err != nil {
		t.Fatal(err)
	}
	p.text.data = p.text.data[:0]
	if err := p.parseLine([]byte("jg _start")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("jg no code")
	}
}

func TestEmitCall(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("_start:")); err != nil {
		t.Fatal(err)
	}
	p.text.data = p.text.data[:0]
	if err := p.parseLine([]byte("call _start")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("call no code")
	}
	if len(p.relocs) == 0 {
		t.Error("call missing relocation")
	}
}

func TestParseOperandMemoryIndex(t *testing.T) {
	op, err := parseOperand([]byte("[rax+rbx*2]"))
	if err != nil {
		t.Fatal(err)
	}
	if op.typ != opMem {
		t.Error("not memory operand")
	}
}

func TestParseOperandDispNegative(t *testing.T) {
	op, err := parseOperand([]byte("[rbp-8]"))
	if err != nil {
		t.Fatal(err)
	}
	if op.typ != opMem {
		t.Error("not memory operand")
	}
}

func TestParseOperandEmpty(t *testing.T) {
	_, err := parseOperand([]byte(""))
	if err == nil {
		t.Error("empty operand should fail")
	}
}

func TestParseOperandInvalid(t *testing.T) {
	_, err := parseOperand([]byte("[rax+*]"))
	if err == nil {
		t.Error("invalid operand should fail")
	}
}

func TestSplitArgsEmpty(t *testing.T) {
	p := &parser{}
	got := p.splitArgs([]byte(""))
	if len(got) != 1 {
		t.Errorf("splitArgs empty = %d parts, want 1", len(got))
	}
}

func TestReadTokenSpaces(t *testing.T) {
	tok, rest := readToken([]byte("   mov   rax"))
	if string(tok) != "mov" {
		t.Errorf("readToken = %q, want mov", tok)
	}
	if string(rest) != "   rax" {
		t.Errorf("rest = %q", rest)
	}
}

func TestRemoveCommentSemi(t *testing.T) {
	got := removeComment([]byte("mov rax, rbx ; comment"))
	if string(got) != "mov rax, rbx " {
		t.Errorf("removeComment = %q, want 'mov rax, rbx '", got)
	}
}

func TestRemoveCommentHash(t *testing.T) {
	got := removeComment([]byte("add rdi, 5 # comment"))
	if string(got) != "add rdi, 5 " {
		t.Errorf("removeComment = %q", got)
	}
}

func TestTrimSpaceEmpty(t *testing.T) {
	got := trimSpace([]byte{})
	if len(got) != 0 {
		t.Errorf("trimSpace empty = %v", got)
	}
}

func TestParseNumberHex(t *testing.T) {
	v, err := parseNumber([]byte("0xABC"))
	if err != nil || v != 0xABC {
		t.Errorf("parseNumber hex = %d, want 0xABC", v)
	}
}

func TestParseNumberBinary(t *testing.T) {
	v, err := parseNumber([]byte("0b1010"))
	if err != nil || v != 10 {
		t.Errorf("parseNumber bin = %d, want 10", v)
	}
}

func TestParseNumberOctal(t *testing.T) {
	v, err := parseNumber([]byte("0o755"))
	if err != nil || v != 493 {
		t.Errorf("parseNumber oct = %d, want 493", v)
	}
}

func TestParseNumberNegative(t *testing.T) {
	v, err := parseNumber([]byte("-123"))
	if err != nil || v != 123 {
		t.Errorf("parseNumber neg = %d, want 123", v)
	}
}

func TestParseNumberInvalidDigit(t *testing.T) {
	_, err := parseNumber([]byte("0xG"))
	if err == nil {
		t.Error("0xG should fail")
	}
}

func TestParseNumberEmpty(t *testing.T) {
	_, err := parseNumber([]byte(""))
	if err == nil {
		t.Error("empty should fail")
	}
}

func TestEncodeModRM(t *testing.T) {
	if encodeModRM(3, 0, 0) != 0xC0 {
		t.Error("encodeModRM(3,0,0) != C0")
	}
	if encodeModRM(0, 1, 2) != 0x0A {
		t.Error("encodeModRM(0,1,2) != 0A")
	}
}

func TestEncodeSIB(t *testing.T) {
	if encodeSIB(0, 0, 0) != 0x00 {
		t.Error("encodeSIB(0,0,0) != 00")
	}
	if encodeSIB(1, 1, 2) != 0x4A {
		t.Error("encodeSIB(1,1,2) != 4A")
	}
}

func TestSelfTargetProfileDefault(t *testing.T) {
	p := selfTargetProfile("unknown")
	if p.eMachine != elfMachineX86_64 {
		t.Errorf("default machine = %d, want %d", p.eMachine, elfMachineX86_64)
	}
}

func TestSelfTargetProfile32(t *testing.T) {
	p := selfTargetProfile("i386-linux")
	if p.eMachine != elfMachine386 || p.elfClass != elfClass32 {
		t.Error("i386 profile wrong")
	}
}

func TestSelfTargetProfileARM(t *testing.T) {
	p := selfTargetProfile("arm-linux")
	if p.eMachine != elfMachineARM || p.elfClass != elfClass32 {
		t.Error("arm profile wrong")
	}
}

func TestSelfTargetProfileRISCV(t *testing.T) {
	p := selfTargetProfile("riscv64")
	if p.eMachine != elfMachineRISCV || p.elfClass != elfClass64 {
		t.Error("riscv profile wrong")
	}
}

func TestSelfTargetProfileAARCH64(t *testing.T) {
	p := selfTargetProfile("aarch64")
	if p.eMachine != elfMachineAARCH64 || p.elfClass != elfClass64 {
		t.Error("aarch64 profile wrong")
	}
}

func TestSectionIndexUnknown(t *testing.T) {
	if idx := sectionIndex(&sectionState{name: []byte(".foo")}); idx != shnUnDef {
		t.Errorf("sectionIndex unknown = %d, want UNDEF", idx)
	}
}

func TestEmitJmpReg(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("jmp rax")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("jmp reg no code")
	}
}

func TestEmitJmpMem(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("jmp [rax]")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("jmp mem no code")
	}
}

func TestEmitJmpLabel(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("_start:")); err != nil {
		t.Fatal(err)
	}
	p.text.data = p.text.data[:0]
	if err := p.parseLine([]byte("jmp _start")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("jmp label no code")
	}
	if len(p.relocs) == 0 {
		t.Error("jmp label missing relocation")
	}
}

func TestEmitMovMemReg(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("mov [rax], rbx")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("mov mem reg no code")
	}
}

func TestEmitMovRegMem(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("mov rax, [rbx]")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("mov reg mem no code")
	}
}

func TestEmitLock(t *testing.T) {
	p := &parser{}
	p.text.name = []byte(".text")
	p.text.data = make([]byte, 0, 1024)
	p.current = &p.text
	if err := p.parseLine([]byte("lock add rax, rbx")); err != nil {
		t.Fatal(err)
	}
	if len(p.text.data) == 0 {
		t.Error("lock no code")
	}
	if p.text.data[0] != 0xF0 {
		t.Error("lock prefix missing")
	}
}
