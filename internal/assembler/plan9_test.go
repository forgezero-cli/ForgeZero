package assembler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsGoAsmFilePlan9(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan9.s")
	if err := os.WriteFile(path, []byte("#include \"textflag.h\"\nTEXT ·add(SB),NOSPLIT,$0-8\nRET\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !isGoAsmFile(path) {
		t.Fatal("Plan 9 assembly was not detected")
	}
}
