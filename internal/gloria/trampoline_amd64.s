// trampoline_amd64.s
#include "textflag.h"

// func callUint64(ptr unsafe.Pointer) uint64
TEXT ·callUint64(SB), NOSPLIT, $0-16
    MOVQ ptr+0(FP), AX
    TESTQ AX, AX
    JZ    ret_zero
    CALL AX
    MOVQ AX, ret+8(FP)
    RET
ret_zero:
    MOVQ $0, ret+8(FP)
    RET
