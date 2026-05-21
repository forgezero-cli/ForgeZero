package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherAddFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "single.txt")
	os.WriteFile(file, []byte("x"), 0o644)
	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.Add(file); err != nil {
		t.Fatal(err)
	}
}

func TestWatcherAddRecursiveNotDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	os.WriteFile(file, []byte("x"), 0o644)
	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.AddRecursive(file); err == nil {
		t.Fatal("expected not directory error")
	}
}

func TestWatcherAddRecursiveMissing(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.AddRecursive("/nonexistent-path-xyz"); err == nil {
		t.Fatal("expected stat error")
	}
}

func TestWatcherHandlerError(t *testing.T) {
	dir := t.TempDir()
	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.AddRecursive(dir)
	done := make(chan struct{})
	go func() {
		w.Watch(20*time.Millisecond, func(string) error {
			close(done)
			return os.ErrPermission
		})
	}()
	time.Sleep(30 * time.Millisecond)
	f, _ := os.Create(filepath.Join(dir, "touch.txt"))
	f.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handler not invoked")
	}
}

func TestShouldIgnoreExtensions(t *testing.T) {
	cases := map[string]bool{
		"/p/.fz_objs":    true,
		"/p/.fz_cache":   true,
		"/p/a.o":         true,
		"/p/a.out":       true,
		"/p/a.exe":       true,
		"/p/main.c":      false,
	}
	for path, want := range cases {
		if shouldIgnore(path) != want {
			t.Fatalf("%s: got %v want %v", path, !want, want)
		}
	}
}
