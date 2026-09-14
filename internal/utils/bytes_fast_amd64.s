/*
 * Copyright (c) 2026 forgezero-cli
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

#include "textflag.h"

DATA ·delimBraceL+0(SB)/8, $0x7b7b7b7b7b7b7b7b
DATA ·delimBraceL+8(SB)/8, $0x7b7b7b7b7b7b7b7b
GLOBL ·delimBraceL(SB), RODATA|NOPTR, $16
DATA ·delimBraceR+0(SB)/8, $0x7d7d7d7d7d7d7d7d
DATA ·delimBraceR+8(SB)/8, $0x7d7d7d7d7d7d7d7d
GLOBL ·delimBraceR(SB), RODATA|NOPTR, $16
DATA ·delimSemi+0(SB)/8, $0x3b3b3b3b3b3b3b3b
DATA ·delimSemi+8(SB)/8, $0x3b3b3b3b3b3b3b3b
GLOBL ·delimSemi(SB), RODATA|NOPTR, $16
DATA ·delimNewline+0(SB)/8, $0x0a0a0a0a0a0a0a0a
DATA ·delimNewline+8(SB)/8, $0x0a0a0a0a0a0a0a0a
GLOBL ·delimNewline(SB), RODATA|NOPTR, $16

TEXT ·bytesEqualAsm(SB), NOSPLIT, $0-25
	MOVQ left+0(FP), SI
	MOVQ right+8(FP), DI
	MOVQ n+16(FP), CX
	XORQ AX, AX
	CMPQ CX, $16
	JB equal_tail

vec_loop:
	MOVOU (SI), X0
	MOVOU (DI), X1
	PCMPEQB X1, X0
	PMOVMSKB X0, AX
	CMPL AX, $65535
	JNE not_equal
	ADDQ $16, SI
	ADDQ $16, DI
	SUBQ $16, CX
	CMPQ CX, $16
	JAE vec_loop

equal_tail:
	TESTQ CX, CX
	JZ equal
byte_loop:
	MOVBQZX (SI), AX
	MOVBQZX (DI), BX
	CMPB AL, BL
	JNE not_equal
	INCQ SI
	INCQ DI
	DECQ CX
	JNZ byte_loop

equal:
	MOVB $1, ret+24(FP)
	RET
not_equal:
	MOVB $0, ret+24(FP)
	RET

TEXT ·copyHashPairAsm(SB), NOSPLIT, $0-24
	MOVQ dst+0(FP), DI
	MOVQ left+8(FP), SI
	MOVQ right+16(FP), DX
	MOVOU 0(SI), X0
	MOVOU 16(SI), X1
	MOVOU 0(DX), X2
	MOVOU 16(DX), X3
	MOVOU X0, 0(DI)
	MOVOU X1, 16(DI)
	MOVOU X2, 32(DI)
	MOVOU X3, 48(DI)
	RET

TEXT ·findBoomDelimAsm(SB), NOSPLIT, $0-24
	MOVQ data+0(FP), SI
	MOVQ n+8(FP), CX
	MOVQ CX, DX
	XORQ AX, AX
	MOVOU ·delimBraceL(SB), X1
	MOVOU ·delimBraceR(SB), X2
	MOVOU ·delimSemi(SB), X3
	MOVOU ·delimNewline(SB), X4
	CMPQ CX, $16
	JB delim_none
	delim_loop:
	MOVOU (SI), X0
	MOVOU X0, X5
	MOVOU X0, X6
	MOVOU X0, X7
	MOVOU X0, X8
	PCMPEQB X1, X5
	PCMPEQB X2, X6
	PCMPEQB X3, X7
	PCMPEQB X4, X8
	PMOVMSKB X5, AX
	PMOVMSKB X6, BX
	ORQ BX, AX
	PMOVMSKB X7, BX
	ORQ BX, AX
	PMOVMSKB X8, BX
	ORQ BX, AX
	TESTQ AX, AX
	JNZ delim_found
	ADDQ $16, SI
	SUBQ $16, CX
	CMPQ CX, $16
	JAE delim_loop

delim_none:
	MOVQ DX, AX
	SUBQ CX, AX
	MOVQ AX, ret+16(FP)
	RET

delim_found:
	BSFQ AX, AX
	MOVQ DX, BX
	SUBQ CX, BX
	ADDQ BX, AX
	MOVQ AX, ret+16(FP)
	RET
