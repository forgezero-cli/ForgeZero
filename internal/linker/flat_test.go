package linker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"fz/internal/assembler"
)

func TestLinkFlatBinaryCopy(t *testing.T) {
	old := assembler.OutputFormat
	defer func() { assembler.OutputFormat = old }()
	assembler.OutputFormat = "bin"
	dir := t.TempDir()
	obj := filepath.Join(dir, "flat.bin")
	payload := []byte{0x90, 0x90, 0xeb, 0xfe}
	if err := os.WriteFile(obj, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.bin")
	if err := Link(context.Background(), obj, out, false, "raw", true, false, false, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(payload) {
		t.Fatalf("len %d want %d", len(got), len(payload))
	}
	for i := range payload {
		if got[i] != payload[i] {
			t.Fatalf("byte %d mismatch", i)
		}
	}
}

func TestLinkFlatBinaryNoCopy(t *testing.T) {
	old := assembler.OutputFormat
	defer func() { assembler.OutputFormat = old }()
	assembler.OutputFormat = "bin"
	dir := t.TempDir()
	obj := filepath.Join(dir, "same.bin")
	if err := os.WriteFile(obj, []byte{0xcd}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Link(context.Background(), obj, obj, false, "raw", true, false, false, nil); err != nil {
		t.Fatal(err)
	}
}
