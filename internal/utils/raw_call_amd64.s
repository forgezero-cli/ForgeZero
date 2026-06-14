//go:build amd64
// +build amd64

#include "textflag.h"

TEXT ·callRaw0(SB), NOSPLIT, $0-8
    MOVQ code+0(FP), AX
    CALL AX
    RET

TEXT ·callRaw2(SB), NOSPLIT, $0-24
    MOVQ code+0(FP), AX
    MOVQ p+8(FP), DI
    MOVQ n+16(FP), SI
    CALL AX
    RET


TEXT ·callRawRet(SB), NOSPLIT, $0-16 
    MOVQ code+0(FP), AX 
    CALL AX 
    MOVQ AX, ret+8(FP)
    RET 
