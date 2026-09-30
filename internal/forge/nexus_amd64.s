//go:build amd64

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
#include "utec_amd64.h"

DATA ·nexusBB64MulVector+0(SB)/8, $0x9e3779b97f4a7c15
DATA ·nexusBB64MulVector+8(SB)/8, $0x9e3779b97f4a7c15
DATA ·nexusBB64MulVector+16(SB)/8, $0x9e3779b97f4a7c15
DATA ·nexusBB64MulVector+24(SB)/8, $0x9e3779b97f4a7c15
GLOBL ·nexusBB64MulVector(SB), RODATA|NOPTR, $32
DATA ·nexusBB64MaskVector+0(SB)/8, $0x00000000ffffffff
DATA ·nexusBB64MaskVector+8(SB)/8, $0x00000000ffffffff
DATA ·nexusBB64MaskVector+16(SB)/8, $0x00000000ffffffff
DATA ·nexusBB64MaskVector+24(SB)/8, $0x00000000ffffffff
GLOBL ·nexusBB64MaskVector(SB), RODATA|NOPTR, $32

TEXT ·dispatchBatchAVX2(SB), NOSPLIT, $32-16
	MOVQ batchPointer+0(FP), DI
	MOVQ count+8(FP), AX
	MOVQ AX, 0(SP)
	MOVQ DI, 8(SP)
	MOVQ $0, 24(SP)

nexus_task_loop:
	MOVQ 0(SP), AX
	TESTQ AX, AX
	JZ nexus_done
	MOVQ 8(SP), DI
	CMPQ 24(SP), $0
	JE nexus_dispatch_kind
	MOVQ $-125, AX
	JMP nexus_finish

nexus_dispatch_kind:
	MOVWQZX UTEC_KIND(DI), AX
	CMPQ AX, $1
	JE nexus_hash
	CMPQ AX, $2
	JE nexus_move
	CMPQ AX, $3
	JE nexus_compare
	CMPQ AX, $4
	JE nexus_driver_io
	CMPQ AX, $5
	JE nexus_syscall_proxy
	MOVQ $-1, AX
	JMP nexus_finish

nexus_hash:
	MOVQ UTEC_SRC_PTR(DI), SI
	MOVQ UTEC_LEN(DI), CX
	MOVQ CX, 16(SP)
	MOVQ UTEC_AUX1(DI), R8
	MOVQ $0x9e3779b97f4a7c15, R11
	MOVQ R8, AX
	VMOVQ AX, X1
	VPBROADCASTQ X1, Y4
	VPXOR Y7, Y7, Y7
	VMOVDQU ·nexusBB64MulVector(SB), Y5
	VMOVDQU ·nexusBB64MaskVector(SB), Y6
	MOVQ CX, DX
	SHRQ $7, DX
	JZ nexus_hash_remainder

nexus_hash_four_blocks:
	VMOVDQU (SI), Y0
	VMOVDQU 32(SI), Y1
	VMOVDQU 64(SI), Y2
	VMOVDQU 96(SI), Y3
	VPXOR Y4, Y0, Y0
	VPXOR Y4, Y1, Y1
	VPXOR Y4, Y2, Y2
	VPXOR Y4, Y3, Y3
	VPSRLQ $29, Y0, Y8
	VPSRLQ $29, Y1, Y10
	VPSRLQ $29, Y2, Y12
	VPSRLQ $29, Y3, Y14
	VPSLLQ $35, Y0, Y9
	VPSLLQ $35, Y1, Y11
	VPSLLQ $35, Y2, Y13
	VPSLLQ $35, Y3, Y15
	VPXOR Y8, Y0, Y0
	VPXOR Y10, Y1, Y1
	VPXOR Y12, Y2, Y2
	VPXOR Y14, Y3, Y3
	VPXOR Y9, Y0, Y0
	VPXOR Y11, Y1, Y1
	VPXOR Y13, Y2, Y2
	VPXOR Y15, Y3, Y3
	VPMULUDQ Y5, Y0, Y8
	VPMULUDQ Y5, Y1, Y10
	VPMULUDQ Y5, Y2, Y12
	VPMULUDQ Y5, Y3, Y14
	VPAND Y6, Y8, Y8
	VPAND Y6, Y10, Y10
	VPAND Y6, Y12, Y12
	VPAND Y6, Y14, Y14
	VPXOR Y8, Y0, Y0
	VPXOR Y10, Y1, Y1
	VPXOR Y12, Y2, Y2
	VPXOR Y14, Y3, Y3
	VPXOR Y0, Y7, Y7
	VPXOR Y1, Y7, Y7
	VPXOR Y2, Y7, Y7
	VPXOR Y3, Y7, Y7
	ADDQ $128, SI
	SUBQ $128, CX
	DECQ DX
	JNZ nexus_hash_four_blocks

nexus_hash_remainder:
	CMPQ CX, $32
	JB nexus_hash_tail
nexus_hash_block:
	VMOVDQU (SI), Y0
	VPXOR Y4, Y0, Y0
	VPSRLQ $29, Y0, Y8
	VPSLLQ $35, Y0, Y9
	VPXOR Y8, Y0, Y0
	VPXOR Y9, Y0, Y0
	VPMULUDQ Y5, Y0, Y8
	VPAND Y6, Y8, Y8
	VPXOR Y8, Y0, Y0
	VPXOR Y0, Y7, Y7
	ADDQ $32, SI
	SUBQ $32, CX
	CMPQ CX, $32
	JAE nexus_hash_block

nexus_hash_tail:
	MOVQ R8, R9
	TESTQ CX, CX
	JZ nexus_hash_fold
nexus_hash_tail_loop:
	MOVBQZX (SI), AX
	XORQ AX, R9
	ROLQ $13, R9
	IMULQ R11, R9
	INCQ SI
	DECQ CX
	JNZ nexus_hash_tail_loop

nexus_hash_fold:
	VEXTRACTI128 $0, Y7, X0
	VEXTRACTI128 $1, Y7, X1
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
	JMP nexus_finish

nexus_move:
	MOVQ UTEC_SRC_PTR(DI), SI
	MOVQ UTEC_DST_PTR(DI), DX
	MOVQ UTEC_LEN(DI), CX
	XORQ R9, R9
	MOVQ CX, 16(SP)
	TESTQ CX, CX
	JZ nexus_move_done
	CMPQ SI, DX
	JE nexus_move_done
	JAE nexus_move_forward
	LEAQ (SI)(CX*1), AX
	CMPQ DX, AX
	JAE nexus_move_forward
	LEAQ -1(SI)(CX*1), SI
	LEAQ -1(DX)(CX*1), DX
nexus_move_backward:
	MOVB (SI), AX
	MOVB AX, (DX)
	DECQ SI
	DECQ DX
	DECQ CX
	JNZ nexus_move_backward
	JMP nexus_move_done

nexus_move_forward:
	MOVWQZX UTEC_FLAGS(DI), AX
	TESTQ $1, AX
	JZ nexus_move_temporal
	CMPQ CX, $4096
	JB nexus_move_temporal
nexus_move_align:
	TESTQ $31, DX
	JE nexus_move_stream
	MOVB (SI), AX
	MOVB AX, (DX)
	INCQ SI
	INCQ DX
	DECQ CX
	JMP nexus_move_align
nexus_move_stream:
	CMPQ CX, $128
	JB nexus_move_stream_tail
	PREFETCHT0 256(SI)
	MOVQ $1, R9
	VMOVDQU (SI), Y0
	VMOVDQU 32(SI), Y1
	VMOVDQU 64(SI), Y2
	VMOVDQU 96(SI), Y3
	VMOVNTDQ Y0, (DX)
	VMOVNTDQ Y1, 32(DX)
	VMOVNTDQ Y2, 64(DX)
	VMOVNTDQ Y3, 96(DX)
	ADDQ $128, SI
	ADDQ $128, DX
	SUBQ $128, CX
	JMP nexus_move_stream
nexus_move_stream_tail:
	CMPQ CX, $32
	JB nexus_move_byte_tail
	MOVQ $1, R9
	VMOVDQU (SI), Y0
	VMOVNTDQ Y0, (DX)
	ADDQ $32, SI
	ADDQ $32, DX
	SUBQ $32, CX
	JMP nexus_move_stream_tail
nexus_move_temporal:
	CMPQ CX, $128
	JB nexus_move_byte_tail
	VMOVDQU (SI), Y0
	VMOVDQU 32(SI), Y1
	VMOVDQU 64(SI), Y2
	VMOVDQU 96(SI), Y3
	VMOVDQU Y0, (DX)
	VMOVDQU Y1, 32(DX)
	VMOVDQU Y2, 64(DX)
	VMOVDQU Y3, 96(DX)
	ADDQ $128, SI
	ADDQ $128, DX
	SUBQ $128, CX
	JMP nexus_move_temporal
nexus_move_byte_tail:
	CMPQ CX, $32
	JB nexus_move_byte_tail_loop
	VMOVDQU (SI), Y0
	VMOVDQU Y0, (DX)
	ADDQ $32, SI
	ADDQ $32, DX
	SUBQ $32, CX
	JMP nexus_move_byte_tail
nexus_move_byte_tail_loop:
	TESTQ CX, CX
	JZ nexus_move_done
	MOVB (SI), AX
	MOVB AX, (DX)
	INCQ SI
	INCQ DX
	DECQ CX
	JMP nexus_move_byte_tail
nexus_move_done:
	TESTQ R9, R9
	JZ nexus_move_no_fence
	SFENCE
nexus_move_no_fence:
	VZEROUPPER
	MOVQ 16(SP), AX
	JMP nexus_finish

nexus_compare:
	MOVQ UTEC_SRC_PTR(DI), SI
	MOVQ UTEC_DST_PTR(DI), DX
	MOVQ UTEC_LEN(DI), CX
	CMPQ CX, $32
	JB nexus_compare_tail
nexus_compare_vector:
	VMOVDQU (SI), Y0
	VMOVDQU (DX), Y1
	VPCMPEQB Y1, Y0, Y2
	VPMOVMSKB Y2, AX
	CMPL AX, $0xffffffff
	JNE nexus_compare_not_equal
	ADDQ $32, SI
	ADDQ $32, DX
	SUBQ $32, CX
	CMPQ CX, $32
	JAE nexus_compare_vector
nexus_compare_tail:
	TESTQ CX, CX
	JZ nexus_compare_equal
	MOVBQZX (SI), AX
	MOVBQZX (DX), BX
	CMPB AL, BL
	JNE nexus_compare_not_equal
	INCQ SI
	INCQ DX
	DECQ CX
	JMP nexus_compare_tail
nexus_compare_equal:
	MOVQ $1, AX
	VZEROUPPER
	JMP nexus_finish
nexus_compare_not_equal:
	XORQ AX, AX
	VZEROUPPER
	JMP nexus_finish

nexus_driver_io:
	MOVQ UTEC_LEN(DI), CX
	CMPQ CX, $64
	JNE nexus_driver_io_invalid
	MOVQ UTEC_SRC_PTR(DI), SI
	MOVQ UTEC_DST_PTR(DI), DX
	TESTQ SI, SI
	JZ nexus_driver_io_invalid
	TESTQ DX, DX
	JZ nexus_driver_io_invalid
	MOVQ DRIVER_IO_SQE(DX), R8
	MOVQ DRIVER_IO_ARRAY_SLOT(DX), R9
	TESTQ R8, R8
	JZ nexus_driver_io_invalid
	TESTQ R9, R9
	JZ nexus_driver_io_invalid
	VMOVDQU (SI), Y0
	VMOVDQU 32(SI), Y1
	VMOVDQU Y0, (R8)
	VMOVDQU Y1, 32(R8)
	MOVL UTEC_ID(DI), CX
	MOVL CX, (R9)
	SFENCE
	MOVQ $64, AX
	VZEROUPPER
	JMP nexus_finish
nexus_driver_io_invalid:
	MOVQ $-1, AX
	JMP nexus_finish

nexus_syscall_proxy:
	MOVQ DI, 16(SP)
	MOVQ 16(SP), BX
	MOVQ UTEC_AUX1(BX), AX
	CMPQ AX, $SYS_UNSHARE
	JE nexus_syscall_unshare
	CMPQ AX, $SYS_MOUNT
	JE nexus_syscall_mount
	CMPQ AX, $SYS_PIVOT_ROOT
	JE nexus_syscall_pivot_root
	CMPQ AX, $SYS_CHROOT
	JE nexus_syscall_chroot
	CMPQ AX, $SYS_CHDIR
	JE nexus_syscall_chroot
	CMPQ AX, $SYS_UMOUNT2
	JE nexus_syscall_umount2
	JMP nexus_syscall_denied

nexus_syscall_unshare:
	MOVQ UTEC_AUX2(BX), DI
	MOVQ DI, R12
	ANDQ $0x6c020000, R12
	CMPQ R12, DI
	JNE nexus_syscall_denied
	TESTQ $0x00020000, DI
	JZ nexus_syscall_denied
	JMP nexus_syscall_execute

nexus_syscall_mount:
	MOVQ UTEC_LEN(BX), R10
	CMPQ R10, $0x44000
	JE nexus_syscall_mount_private
	CMPQ R10, $0x5000
	JNE nexus_syscall_denied
	MOVQ UTEC_AUX2(BX), DI
	TESTQ DI, DI
	JZ nexus_syscall_denied
	MOVQ UTEC_SRC_PTR(BX), SI
	TESTQ SI, SI
	JZ nexus_syscall_denied
	JMP nexus_syscall_mount_common
nexus_syscall_mount_private:
	MOVQ UTEC_AUX2(BX), DI
	TESTQ DI, DI
	JNZ nexus_syscall_denied
	MOVQ UTEC_SRC_PTR(BX), SI
	TESTQ SI, SI
	JZ nexus_syscall_denied
nexus_syscall_mount_common:
	MOVQ UTEC_DST_PTR(BX), DX
	TESTQ DX, DX
	JNZ nexus_syscall_denied
	MOVQ UTEC_AUX3(BX), R8
	TESTQ R8, R8
	JNZ nexus_syscall_denied
	MOVQ UTEC_RET_VAL(BX), R9
	TESTQ R9, R9
	JNZ nexus_syscall_denied
	JMP nexus_syscall_execute

nexus_syscall_pivot_root:
	MOVQ UTEC_AUX2(BX), DI
	MOVQ UTEC_SRC_PTR(BX), SI
	TESTQ DI, DI
	JZ nexus_syscall_denied
	TESTQ SI, SI
	JZ nexus_syscall_denied
	JMP nexus_syscall_zero_tail_args

nexus_syscall_chroot:
	MOVQ UTEC_AUX2(BX), DI
	TESTQ DI, DI
	JZ nexus_syscall_denied
	XORQ SI, SI
	JMP nexus_syscall_zero_tail_args

nexus_syscall_umount2:
	MOVQ UTEC_AUX2(BX), DI
	MOVQ UTEC_SRC_PTR(BX), SI
	TESTQ DI, DI
	JZ nexus_syscall_denied
	CMPQ SI, $2
	JNE nexus_syscall_denied
	XORQ DX, DX
	nexus_syscall_zero_tail_args:
		MOVQ UTEC_DST_PTR(BX), DX
		MOVQ UTEC_LEN(BX), R10
		MOVQ UTEC_AUX3(BX), R8
		MOVQ UTEC_RET_VAL(BX), R9
	TESTQ DX, DX
	JNZ nexus_syscall_denied
	TESTQ R10, R10
	JNZ nexus_syscall_denied
	TESTQ R8, R8
	JNZ nexus_syscall_denied
	TESTQ R9, R9
	JNZ nexus_syscall_denied

nexus_syscall_execute:
	MOVQ UTEC_AUX1(BX), AX
	SYSCALL
	TESTQ AX, AX
	JNS nexus_finish
	MOVWQZX UTEC_FLAGS(BX), CX
	TESTQ $4, CX
	JZ nexus_finish
	MOVQ $1, 24(SP)
	JMP nexus_finish

nexus_syscall_denied:
	MOVQ $-1, AX
	MOVWQZX UTEC_FLAGS(BX), CX
	TESTQ $4, CX
	JZ nexus_finish
	MOVQ $1, 24(SP)
	JMP nexus_finish

nexus_finish:
	MOVQ 8(SP), DI
	MOVQ AX, UTEC_RET_VAL(DI)
	ADDQ $64, DI
	MOVQ DI, 8(SP)
	MOVQ 0(SP), CX
	DECQ CX
	MOVQ CX, 0(SP)
	JMP nexus_task_loop

nexus_done:
	VZEROUPPER
	RET
