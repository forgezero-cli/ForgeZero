package main

import (
	"os"
	"testing"
)

func TestMainInit(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

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
