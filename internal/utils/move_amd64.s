/*
 *   Copyright (c) 2026 forgezero-cli
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

#include "textflag.h"

TEXT ·copyBytesAVX2(SB), NOSPLIT, $0-24
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ size+16(FP), CX

move_align:
	TESTQ $31, DI
	JE move_vector
	MOVB (SI), AL
	MOVB AL, (DI)
	INCQ SI
	INCQ DI
	DECQ CX
	JMP move_align

move_vector:
	CMPQ CX, $128
	JB move_remainder
	CMPQ CX, $512
	JB move_four_vectors
	PREFETCHT0 256(SI)

move_four_vectors:
	VMOVDQU (SI), Y0
	VMOVDQU 32(SI), Y1
	VMOVDQU 64(SI), Y2
	VMOVDQU 96(SI), Y3
	VMOVNTDQ Y0, (DI)
	VMOVNTDQ Y1, 32(DI)
	VMOVNTDQ Y2, 64(DI)
	VMOVNTDQ Y3, 96(DI)
	ADDQ $128, SI
	ADDQ $128, DI
	SUBQ $128, CX
	JMP move_vector

move_remainder:
	CMPQ CX, $32
	JB move_tail
	VMOVDQU (SI), Y0
	VMOVNTDQ Y0, (DI)
	ADDQ $32, SI
	ADDQ $32, DI
	SUBQ $32, CX
	JMP move_remainder

move_tail:
	TESTQ CX, CX
	JZ move_done
	MOVB (SI), AL
	MOVB AL, (DI)
	INCQ SI
	INCQ DI
	DECQ CX
	JMP move_tail

move_done:
	SFENCE
	VZEROUPPER
	RET
