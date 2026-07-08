// Copyright (c) 2026 ForgeZero-cli
//go:build amd64
// +build amd64

#include "textflag.h"

TEXT ·prefetch(SB), NOSPLIT, $0-8
    MOVQ ptr+0(FP), DI
    PREFETCHT0 (DI)
    RET
