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

package gloria

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func toHex(b []byte) string {
	return hex.EncodeToString(b)
}

func TestEmitMovRegToStack(t *testing.T) {
	tests := []struct {
		name     string
		srcReg   int
		offset   int
		expected []byte
	}{
		{
			name:     "mov [rbp-8], rdi (reg 7)",
			srcReg:   7,
			offset:   -8,
			expected: []byte{0x48, 0x89, 0xBD, 0xF8, 0xFF, 0xFF, 0xFF},
		},
		{
			name:     "mov [rbp-16], rsi (reg 6)",
			srcReg:   6,
			offset:   -16,
			expected: []byte{0x48, 0x89, 0xB5, 0xF0, 0xFF, 0xFF, 0xFF},
		},
		{
			name:     "mov [rbp-24], rax (reg 0)",
			srcReg:   0,
			offset:   -24,
			expected: []byte{0x48, 0x89, 0x85, 0xE8, 0xFF, 0xFF, 0xFF},
		},
		{
			name:     "mov [rbp+32], rcx (reg 1)",
			srcReg:   1,
			offset:   32,
			expected: []byte{0x48, 0x89, 0x8D, 0x20, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := emitMovRegToStack(nil, tt.srcReg, tt.offset)
			if !bytes.Equal(out, tt.expected) {
				t.Errorf("emitMovRegToStack(%d, %d) = %s, want %s",
					tt.srcReg, tt.offset, toHex(out), toHex(tt.expected))
			}
		})
	}
}

func TestEmitMovStackToReg(t *testing.T) {
	tests := []struct {
		name     string
		dstReg   int
		offset   int
		expected []byte
	}{
		{
			name:     "mov rax, [rbp-8]",
			dstReg:   0,
			offset:   -8,
			expected: []byte{0x48, 0x8B, 0x85, 0xF8, 0xFF, 0xFF, 0xFF},
		},
		{
			name:     "mov rcx, [rbp-16]",
			dstReg:   1,
			offset:   -16,
			expected: []byte{0x48, 0x8B, 0x8D, 0xF0, 0xFF, 0xFF, 0xFF},
		},
		{
			name:     "mov rdx, [rbp+64]",
			dstReg:   2,
			offset:   64,
			expected: []byte{0x48, 0x8B, 0x95, 0x40, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := emitMovStackToReg(nil, tt.dstReg, tt.offset)
			if !bytes.Equal(out, tt.expected) {
				t.Errorf("emitMovStackToReg(%d, %d) = %s, want %s",
					tt.dstReg, tt.offset, toHex(out), toHex(tt.expected))
			}
		})
	}
}

func TestEmitCmpRegToReg(t *testing.T) {
	tests := []struct {
		name     string
		src      int
		dst      int
		expected []byte
	}{
		{
			name:     "cmp rax, rcx",
			src:      1,
			dst:      0,
			expected: []byte{0x48, 0x39, 0xC8},
		},
		{
			name:     "cmp rdi, rsi",
			src:      6,
			dst:      7,
			expected: []byte{0x48, 0x39, 0xF7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := emitCmpRegToReg(nil, tt.src, tt.dst)
			if !bytes.Equal(out, tt.expected) {
				t.Errorf("emitCmpRegToReg(%d, %d) = %s, want %s",
					tt.src, tt.dst, toHex(out), toHex(tt.expected))
			}
		})
	}
}

func TestEmitCondJmp(t *testing.T) {
	tests := []struct {
		name        string
		op          byte
		expected    []byte
		expectedIdx int
	}{
		{
			name:        "jge (0x7D)",
			op:          0x7D,
			expected:    []byte{0x7D, 0x00},
			expectedIdx: 1,
		},
		{
			name:        "jle (0x7E)",
			op:          0x7E,
			expected:    []byte{0x7E, 0x00},
			expectedIdx: 1,
		},
		{
			name:        "jne (0x75)",
			op:          0x75,
			expected:    []byte{0x75, 0x00},
			expectedIdx: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, out := emitCondJmp(nil, tt.op)
			if !bytes.Equal(out, tt.expected) {
				t.Errorf("emitCondJmp(0x%02X) = %s, want %s",
					tt.op, toHex(out), toHex(tt.expected))
			}
			if idx != tt.expectedIdx {
				t.Errorf("emitCondJmp index = %d, want %d", idx, tt.expectedIdx)
			}
		})
	}
}

func TestEmitMovImm64ToReg(t *testing.T) {
	tests := []struct {
		name     string
		reg      int
		value    uint64
		expected []byte
	}{
		{
			name:     "mov rax, 0",
			reg:      0,
			value:    0,
			expected: []byte{0x48, 0xB8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:     "mov rax, 100",
			reg:      0,
			value:    100,
			expected: []byte{0x48, 0xB8, 0x64, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:     "mov rax, 0xFFFFFFFFFFFFFFFF",
			reg:      0,
			value:    0xFFFFFFFFFFFFFFFF,
			expected: []byte{0x48, 0xB8, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name:     "mov rcx (reg 1), 0x12345678",
			reg:      1,
			value:    0x12345678,
			expected: []byte{0x48, 0xB9, 0x78, 0x56, 0x34, 0x12, 0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := emitMovImm64ToReg(nil, tt.reg, tt.value)
			if !bytes.Equal(out, tt.expected) {
				t.Errorf("emitMovImm64ToReg(%d, 0x%X) = %s, want %s",
					tt.reg, tt.value, toHex(out), toHex(tt.expected))
			}
		})
	}
}

func TestEmitMovRegToReg(t *testing.T) {
	tests := []struct {
		name     string
		src      int
		dst      int
		expected []byte
	}{
		{
			name:     "mov rax, rcx",
			src:      1,
			dst:      0,
			expected: []byte{0x48, 0x89, 0xC8},
		},
		{
			name:     "mov rdi, rsi",
			src:      6,
			dst:      7,
			expected: []byte{0x48, 0x89, 0xF7},
		},
		{
			name:     "mov rdx, rax",
			src:      0,
			dst:      2,
			expected: []byte{0x48, 0x89, 0xC2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := emitMovRegToReg(nil, tt.src, tt.dst)
			if !bytes.Equal(out, tt.expected) {
				t.Errorf("emitMovRegToReg(%d, %d) = %s, want %s",
					tt.src, tt.dst, toHex(out), toHex(tt.expected))
			}
		})
	}
}

func TestEmitAddSubRegToReg(t *testing.T) {
	tests := []struct {
		name     string
		fn       func([]byte, int, int) []byte
		src      int
		dst      int
		expected []byte
	}{
		{
			name:     "add rax, rcx",
			fn:       emitAddRegToReg,
			src:      1,
			dst:      0,
			expected: []byte{0x48, 0x01, 0xC8},
		},
		{
			name:     "sub rax, rcx",
			fn:       emitSubRegToReg,
			src:      1,
			dst:      0,
			expected: []byte{0x48, 0x29, 0xC8},
		},
		{
			name:     "add rdi, rsi",
			fn:       emitAddRegToReg,
			src:      6,
			dst:      7,
			expected: []byte{0x48, 0x01, 0xF7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := tt.fn(nil, tt.src, tt.dst)
			if !bytes.Equal(out, tt.expected) {
				t.Errorf("%s(%d, %d) = %s, want %s",
					tt.name, tt.src, tt.dst, toHex(out), toHex(tt.expected))
			}
		})
	}
}

func TestEmitAddSubImm64ToReg(t *testing.T) {
	tests := []struct {
		name     string
		fn       func([]byte, int, uint64) []byte
		reg      int
		value    uint64
		expected []byte
	}{
		{
			name:     "add rax, 10",
			fn:       emitAddImm64ToReg,
			reg:      0,
			value:    10,
			expected: []byte{0x48, 0x81, 0xC0, 0x0A, 0x00, 0x00, 0x00},
		},
		{
			name:     "sub rax, 255",
			fn:       emitSubImm64ToReg,
			reg:      0,
			value:    255,
			expected: []byte{0x48, 0x81, 0xE8, 0xFF, 0x00, 0x00, 0x00},
		},
		{
			name:     "add rcx, 0x1000",
			fn:       emitAddImm64ToReg,
			reg:      1,
			value:    0x1000,
			expected: []byte{0x48, 0x81, 0xC1, 0x00, 0x10, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := tt.fn(nil, tt.reg, tt.value)
			if !bytes.Equal(out, tt.expected) {
				t.Errorf("%s(%d, 0x%X) = %s, want %s",
					tt.name, tt.reg, tt.value, toHex(out), toHex(tt.expected))
			}
		})
	}
}

func TestEmitLowLevelPrint(t *testing.T) {
	t.Run("userspace mode with plain string", func(t *testing.T) {
		out := emitLowLevelPrint(nil, "hello", false)
		if len(out) == 0 {
			t.Fatal("emitLowLevelPrint returned empty output")
		}
		strLen := len("hello")
		if out[0] != 0xEB || out[1] != byte(strLen) {
			t.Errorf("expected short jump 0xEB %d, got %s", strLen, toHex(out[:2]))
		}
		if string(out[2:2+strLen]) != "hello" {
			t.Errorf("expected string 'hello' at offset 2, got %q", out[2:2+strLen])
		}
		foundSyscall := false
		for i := 0; i < len(out)-1; i++ {
			if out[i] == 0x0F && out[i+1] == 0x05 {
				foundSyscall = true
				break
			}
		}
		if !foundSyscall {
			t.Error("expected syscall instruction (0x0F 0x05) in output")
		}
	})

	t.Run("escape sequences: \\n and \\t", func(t *testing.T) {
		out := emitLowLevelPrint(nil, "line1\\nline2\\ttab", false)
		strStart := 2
		strEnd := strStart + int(out[1])
		strBytes := out[strStart:strEnd]
		expected := []byte("line1")
		expected = append(expected, 10)
		expected = append(expected, []byte("line2")...)
		expected = append(expected, 9)
		expected = append(expected, []byte("tab")...)
		if !bytes.Equal(strBytes, expected) {
			t.Errorf("escape processing failed: got %v, want %v", strBytes, expected)
		}
	})

	t.Run("kernel mode generates CALL instead of syscall", func(t *testing.T) {
		out := emitLowLevelPrint(nil, "kmsg", true)
		foundCall := false
		for i := 0; i < len(out)-4; i++ {
			if out[i] == 0xE8 && out[i+1] == 0x00 && out[i+2] == 0x00 &&
				out[i+3] == 0x00 && out[i+4] == 0x00 {
				foundCall = true
				break
			}
		}
		if !foundCall {
			t.Error("expected CALL 0x00000000 in kernel mode output")
		}
		for i := 0; i < len(out)-1; i++ {
			if out[i] == 0x0F && out[i+1] == 0x05 {
				t.Error("unexpected syscall instruction in kernel mode")
			}
		}
	})
}

func TestAbiArgRegs(t *testing.T) {
	expected := []int{7, 6, 2, 1, 8, 9}
	if len(abiArgRegs) != len(expected) {
		t.Errorf("abiArgRegs length = %d, want %d", len(abiArgRegs), len(expected))
	}
	for i, v := range expected {
		if i < len(abiArgRegs) && abiArgRegs[i] != v {
			t.Errorf("abiArgRegs[%d] = %d, want %d", i, abiArgRegs[i], v)
		}
	}
}

func TestPeephole(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "remove mov rax, rax",
			input:    []byte{0x48, 0x89, 0xC0},
			expected: []byte{},
		},
		{
			name:     "keep mov rax, rcx",
			input:    []byte{0x48, 0x89, 0xC8},
			expected: []byte{0x48, 0x89, 0xC8},
		},
		{
			name:     "mixed instructions",
			input:    []byte{0x48, 0x89, 0xC0, 0x48, 0x89, 0xC8, 0x48, 0x89, 0xC0},
			expected: []byte{0x48, 0x89, 0xC8},
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := peephole(tt.input)
			if !bytes.Equal(out, tt.expected) {
				t.Errorf("peephole(%s) = %s, want %s",
					toHex(tt.input), toHex(out), toHex(tt.expected))
			}
		})
	}
}
