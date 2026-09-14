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

DATA ·bb64MulVector+0(SB)/8, $0x9e3779b97f4a7c15
DATA ·bb64MulVector+8(SB)/8, $0x9e3779b97f4a7c15
DATA ·bb64MulVector+16(SB)/8, $0x9e3779b97f4a7c15
DATA ·bb64MulVector+24(SB)/8, $0x9e3779b97f4a7c15
GLOBL ·bb64MulVector(SB), RODATA|NOPTR, $32
DATA ·bb64MaskVector+0(SB)/8, $0x00000000ffffffff
DATA ·bb64MaskVector+8(SB)/8, $0x00000000ffffffff
DATA ·bb64MaskVector+16(SB)/8, $0x00000000ffffffff
DATA ·bb64MaskVector+24(SB)/8, $0x00000000ffffffff
GLOBL ·bb64MaskVector(SB), RODATA|NOPTR, $32

TEXT ·HashBB64Asm(SB), NOSPLIT, $0-40
	MOVQ data+0(FP), SI
	MOVQ data_len+8(FP), CX
	MOVQ seed+24(FP), R8
	MOVQ $0x9e3779b97f4a7c15, R11
	MOVQ R8, AX
	VMOVQ AX, X1
	VPBROADCASTQ X1, Y1
	VPXOR Y6, Y6, Y6
	VMOVDQU ·bb64MulVector(SB), Y2
	VMOVDQU ·bb64MaskVector(SB), Y7
	MOVQ CX, DX
	SHRQ $5, DX
	JZ bb64_tail

bb64_loop:
	VMOVDQU (SI), Y0
	VPXOR Y1, Y0, Y0
	VPSRLQ $29, Y0, Y3
	VPSLLQ $35, Y0, Y4
	VPXOR Y3, Y0, Y0
	VPXOR Y4, Y0, Y0
	VPMULUDQ Y2, Y0, Y5
	VPAND Y7, Y5, Y5
	VPXOR Y5, Y0, Y0
	VPXOR Y0, Y6, Y6
	ADDQ $32, SI
	SUBQ $32, CX
	DECQ DX
	JNZ bb64_loop

bb64_tail:
	MOVQ R8, R9
	TESTQ CX, CX
	JZ bb64_fold

bb64_tail_loop:
	MOVBQZX (SI), AX
	XORQ AX, R9
	ROLQ $13, R9
	IMULQ R11, R9
	INCQ SI
	DECQ CX
	JNZ bb64_tail_loop

bb64_fold:
	VEXTRACTI128 $0, Y6, X0
	VEXTRACTI128 $1, Y6, X1
	PXOR X1, X0
	PSHUFD $0x4e, X0, X1
	PXOR X1, X0
	MOVQ X0, AX
	XORQ R9, AX
	MOVQ AX, R10
	SHRQ $33, R10
	XORQ R10, AX
	IMULQ R11, AX
	MOVQ AX, R10
	SHRQ $29, R10
	XORQ R10, AX
	VZEROUPPER
	MOVQ AX, ret+32(FP)
	RET
