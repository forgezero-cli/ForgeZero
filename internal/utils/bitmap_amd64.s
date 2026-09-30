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

TEXT ·bitmapSetAtomicAsm(SB), NOSPLIT, $0-16
	MOVQ word+0(FP), AX
	MOVQ bit+8(FP), CX
	MOVQ $1, DX
	SHLQ CX, DX
	LOCK
	ORQ DX, 0(AX)
	RET

TEXT ·bitmapTestAndSetAsm(SB), NOSPLIT, $0-17
	MOVQ word+0(FP), AX
	MOVQ bit+8(FP), CX
	LOCK
	BTSQ CX, 0(AX)
	SETCS ret+16(FP)
	RET

TEXT ·bitmapClearAtomicAsm(SB), NOSPLIT, $0-16
	MOVQ word+0(FP), AX
	MOVQ bit+8(FP), CX
	MOVQ $1, DX
	SHLQ CX, DX
	NOTQ DX
	LOCK
	ANDQ DX, 0(AX)
	RET
