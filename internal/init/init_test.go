package initpkg

import (
	"os"
	"testing"
)

func TestRunCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(dir)

	if err := Run(); err != nil {
		t.Fatal(err)
	}
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

func TestRunFailsIfFilesExist(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(dir)

	os.Create(".fz.yaml")
	if err := Run(); err == nil {
		t.Error("expected error because .fz.yaml exists")
	}
}
