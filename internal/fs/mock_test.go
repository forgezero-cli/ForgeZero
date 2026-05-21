package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMockInjectedErrors(t *testing.T) {
	dir := t.TempDir()
	m := NewMock(Unix{})
	m.SetFailOp("MkdirAll", ErrDiskFull)
	if err := m.MkdirAll(filepath.Join(dir, "x"), 0o700); err != ErrDiskFull {
		t.Fatalf("got %v", err)
	}
}

func TestOpenVerifiedSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "t")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "l")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("no symlink")
	}
	_, err := Unix{}.OpenVerified(link)
	if err != ErrSymlink {
		t.Fatalf("got %v", err)
	}
}
