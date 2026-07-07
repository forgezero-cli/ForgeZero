/*
 *   Copyright (c) 2026 ForgeZero-cli
 */

package assembler

import "testing"

func TestEmitMovRegImmToZeroAllocs(t *testing.T) {
    allocs := testing.AllocsPerRun(100, func() {
        e := GetEncoder()
        EmitMovRegImmTo(e, 2, 0x11223344)
        _ = e.Bytes()
        PutEncoder(e)
    })
    if allocs != 0 {
        t.Fatalf("expected 0 allocations, got %f", allocs)
    }
}

func TestEmitAddRegRegToZeroAllocs(t *testing.T) {
    allocs := testing.AllocsPerRun(100, func() {
        e := GetEncoder()
        EmitAddRegRegTo(e, 0, 1)
        _ = e.Bytes()
        PutEncoder(e)
    })
    if allocs != 0 {
        t.Fatalf("expected 0 allocations, got %f", allocs)
    }
}

func TestEmitMovRegImmToBytes(t *testing.T) {
    e := GetEncoder()
    EmitMovRegImmTo(e, 3, 0x01020304)
    got := e.Bytes()
    want := []byte{0xBB, 0x04, 0x03, 0x02, 0x01}
    if len(got) != len(want) {
        t.Fatalf("len mismatch got %d want %d", len(got), len(want))
    }
    for i := range got {
        if got[i] != want[i] {
            t.Fatalf("byte %d mismatch got %02x want %02x", i, got[i], want[i])
        }
    }
    PutEncoder(e)
}

func TestEmitAddRegRegToBytes(t *testing.T) {
    e := GetEncoder()
    EmitAddRegRegTo(e, 0, 1)
    got := e.Bytes()
    want := []byte{0x48, 0x01, 0xC8}
    if len(got) != len(want) {
        t.Fatalf("len mismatch got %d want %d", len(got), len(want))
    }
    for i := range got {
        if got[i] != want[i] {
            t.Fatalf("byte %d mismatch got %02x want %02x", i, got[i], want[i])
        }
    }
    PutEncoder(e)
}
