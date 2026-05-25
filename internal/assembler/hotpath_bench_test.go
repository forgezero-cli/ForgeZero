package assembler

import "testing"

func TestHotArgAllocations(t *testing.T) {
	var src pathBuf
	src.appendString("/tmp/example.asm")
	allocs := testing.AllocsPerRun(100, func() {
		var v hotArgVec
		v.reset()

		v.pushPath(&src)
		v.pushLiteral("-o")
	})

	if allocs > 0 {
		t.Fatalf("hot arg constuction allocs = %v, want 0", allocs)
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
