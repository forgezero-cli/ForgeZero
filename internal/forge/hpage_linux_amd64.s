//go:build linux && amd64

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

TEXT ·mmapSyscall(SB), NOSPLIT, $0-24
	MOVQ size+0(FP), SI
	MOVQ flags+8(FP), R10
	XORQ DI, DI
	MOVQ $3, DX
	MOVQ $-1, R8
	XORQ R9, R9
	MOVQ $9, AX
	SYSCALL
	MOVQ AX, ret+16(FP)
	RET

TEXT ·munmapSyscall(SB), NOSPLIT, $0-16
	MOVQ pointer+0(FP), DI
	MOVQ size+8(FP), SI
	MOVQ $11, AX
	SYSCALL
	RET
