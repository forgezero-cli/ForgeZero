package assembler

import "testing"

func TestHotArgAllocations(t *testing.T) {
    var src pathBuf
    if !src.appendString("/tmp/example.asm") {
        t.Fatal("setup failed")
    }
    f := func() {
        var v hotArgVec
        v.reset()
        if !v.pushPath(&src) {
            t.Fatal("pushPath failed")
        }
        if !v.pushLiteral("-o") {
            t.Fatal("pushLiteral failed")
        }
    }
    if testing.AllocsPerRun(100, f) != 0 {
        t.Fatal("hot arg construction allocs > 0")
    }
}

func BenchmarkHotArgConstruction(b *testing.B) {
    var src pathBuf
    _ = src.appendString("/tmp/example.asm")
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        var v hotArgVec
        v.reset()
        _ = v.pushPath(&src)
        _ = v.pushLiteral("-o")
    }
}
