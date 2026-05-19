package main

import (
	"os"
	"testing"
)

func TestMainInit(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to test directory: %v\n", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"fz", "-init"}

	main()

	if _, err := os.Stat(".fz.yaml"); err != nil {
		t.Error(".fz.yaml not created")
	}
	if _, err := os.Stat(".fzignore"); err != nil {
		t.Error(".fzignore not created")
	}
	if _, err := os.Stat("README.md"); err != nil {
		t.Error("README.md not created")
	}
}
