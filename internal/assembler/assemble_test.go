package assembler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAssembleUnsupported(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "test.unsupported")
	if err := os.WriteFile(tmp, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(t.TempDir(), "out.o")
	err := Assemble(context.Background(), tmp, obj, false, false, "auto")
	if err == nil {
		t.Error("expected error for unsupported extension")
	}
}
