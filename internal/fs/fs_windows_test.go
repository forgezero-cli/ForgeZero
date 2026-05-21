//go:build windows

package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsRenameAtomic(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.tmp")
	newPath := filepath.Join(dir, "new.dat")
	if err := os.WriteFile(old, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := Windows{}
	if err := w.Rename(old, newPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "payload" {
		t.Fatalf("got %q", data)
	}
}

func TestWindowsOpenVerified(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := Windows{}
	f, err := w.OpenVerified(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestWindowsChmodNoPanic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.txt")
	if err := os.WriteFile(path, []byte("z"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (Windows{}).Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestWindowsCleanPathBackslash(t *testing.T) {
	got := CleanPath(`C:\a\b\c`)
	if got == "" {
		t.Fatal("empty")
	}
}

func TestDefaultIsWindows(t *testing.T) {
	if _, ok := Default.(Windows); !ok {
		t.Fatalf("Default type %T", Default)
	}
}
