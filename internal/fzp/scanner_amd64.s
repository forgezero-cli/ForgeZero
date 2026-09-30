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

TEXT ·scanFZPBlocksAVX2(SB), NOSPLIT, $0-48
	MOVQ data+0(FP), SI
	MOVQ blocks+8(FP), R8
	MOVQ out+16(FP), DI
	MOVQ offsets+24(FP), R10
	XORQ R11, R11
	XORQ R12, R12
	MOVQ $10, AX
	VMOVD AX, X1
	VPBROADCASTB X1, Y1
	MOVQ $35, AX
	VMOVD AX, X2
	VPBROADCASTB X2, Y2
	MOVQ $32, AX
	VMOVD AX, X3
	VPBROADCASTB X3, Y3
	MOVQ $9, AX
	VMOVD AX, X4
	VPBROADCASTB X4, Y4
	MOVQ $13, AX
	VMOVD AX, X5
	VPBROADCASTB X5, Y5

scan_blocks_loop:
	TESTQ R8, R8
	JZ scan_blocks_done
	VMOVDQU (SI), Y0
	VPCMPEQB Y1, Y0, Y6
	VPMOVMSKB Y6, AX
	MOVL AX, R9

scan_newline_loop:
	TESTL R9, R9
	JZ scan_newline_done
	TZCNTL R9, CX
	LEAQ (R11)(CX*1), BX
	MOVW BX, 0(R10)
	ADDQ $2, R10
	INCQ R12
	BLSRL R9, R9
	JMP scan_newline_loop

scan_newline_done:
	VPCMPEQB Y2, Y0, Y6
	VPMOVMSKB Y6, BX
	VPCMPEQB Y3, Y0, Y6
	VPMOVMSKB Y6, CX
	VPCMPEQB Y4, Y0, Y6
	VPMOVMSKB Y6, DX
	VPCMPEQB Y5, Y0, Y6
	VPMOVMSKB Y6, R9
	ORL CX, DX
	ORL R9, DX
	MOVL AX, 0(DI)
	MOVL BX, 4(DI)
	MOVL DX, 8(DI)
	ADDQ $32, SI
	ADDQ $32, R11
	ADDQ $12, DI
	DECQ R8
	JMP scan_blocks_loop

scan_blocks_done:
	VZEROUPPER
	MOVQ blocks+8(FP), AX
	MOVQ AX, ret+32(FP)
	MOVQ R12, ret1+40(FP)
	RET
