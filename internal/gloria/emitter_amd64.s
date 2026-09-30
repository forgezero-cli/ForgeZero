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

TEXT ·writeBareMetalPrintHeaderAVX2(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ template+8(FP), SI
	MOVQ mask+16(FP), DX
	MOVQ patch+24(FP), AX
	VMOVDQU (SI), Y0
	VMOVDQU (AX), Y1
	VMOVDQU (DX), Y2
	VPBLENDVB Y2, Y1, Y0, Y3
	VMOVDQU Y3, (DI)
	VZEROUPPER
	RET
