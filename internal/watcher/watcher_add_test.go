package watcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWatcherAddRecursiveWalkError(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "blocked")
	os.Mkdir(sub, 0o000)
	defer os.Chmod(sub, 0o755)
	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.AddRecursive(dir); err == nil {
		t.Fatal("expected walk error")
	}
}
