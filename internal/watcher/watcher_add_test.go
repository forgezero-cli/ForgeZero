package watcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWatcherAddRecursiveWalkError(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "blocked")

	if err := os.Mkdir(sub, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chmod(sub, 0o755); err != nil {
			t.Errorf("failed to restore permissions: %v", err)
		}
	}()

	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.AddRecursive(dir); err == nil {
		t.Fatal("expected walk error")
	}
}
