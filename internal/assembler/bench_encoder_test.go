/*
 *   Copyright (c) 2026 ForgeZero-cli
 */

package assembler

import "testing"

func BenchmarkEmitMovRegImmTo(b *testing.B) {
    for i := 0; i < b.N; i++ {
        e := GetEncoder()
        EmitMovRegImmTo(e, byte(i&7), 0x11223344)
        _ = e.Bytes()
        PutEncoder(e)
    }
}

func BenchmarkEmitAddRegRegTo(b *testing.B) {
    for i := 0; i < b.N; i++ {
        e := GetEncoder()
        EmitAddRegRegTo(e, byte(i&7), byte((i+1)&7))
        _ = e.Bytes()
        PutEncoder(e)
    }
}
