package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherAddRecursive(t *testing.T) {
	dir := t.TempDir()
	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	err = w.AddRecursive(dir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatcherWatchEvent(t *testing.T) {
	dir := t.TempDir()
	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	err = w.AddRecursive(dir)
	if err != nil {
		t.Fatal(err)
	}
	eventReceived := make(chan bool, 1)
	go w.Watch(100*time.Millisecond, func(string) error {
		eventReceived <- true
		return nil
	})
	time.Sleep(50 * time.Millisecond)
	f, err := os.Create(filepath.Join(dir, "test.txt"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	select {
	case <-eventReceived:
	case <-time.After(2 * time.Second):
		t.Error("event not received")
	}
}

func TestWatcherIgnoreArtifacts(t *testing.T) {
	dir := t.TempDir()
	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	err = w.AddRecursive(dir)
	if err != nil {
		t.Fatal(err)
	}
	eventReceived := make(chan bool, 1)
	go w.Watch(100*time.Millisecond, func(string) error {
		eventReceived <- true
		return nil
	})
	time.Sleep(50 * time.Millisecond)
	objDir := filepath.Join(dir, ".fz_objs")
	err = os.MkdirAll(objDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(objDir, "test.o"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	select {
	case <-eventReceived:
		t.Error("should ignore .fz_objs")
	case <-time.After(500 * time.Millisecond):
	}
}
