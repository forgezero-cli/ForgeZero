//go:build arm64
// +build arm64

#include "textflag.h"

TEXT ·callRaw0(SB), NOSPLIT, $0-8
    MOVD code+0(FP), R0
    BL R0
    RET

TEXT ·callRaw2(SB), NOSPLIT, $0-32
    MOVD code+0(FP), R0
    MOVD p+8(FP), R1
    MOVD n+16(FP), R2
    BL R0
    RET

TEXT ·callRawRet(SB), NOSPLIT, $0-8
    MOVD code+0(FP), R0
    BL R0
    RET
