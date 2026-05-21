//go:build amd64
// +build amd64

#include "textflag.h"

TEXT ·callRaw0(SB), NOSPLIT, $0-8
    MOVQ code+0(FP), AX
    JMP AX

TEXT ·callRaw2(SB), NOSPLIT, $0-32
    MOVQ code+0(FP), AX
    MOVQ p+8(FP), DI
    MOVQ n+16(FP), SI
    JMP AX
