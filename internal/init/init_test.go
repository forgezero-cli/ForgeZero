package initpkg

import (
	"os"
	"testing"
)

func TestRunCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change to test dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
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

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change to test directory: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	f, err := os.Create(".fz.yaml")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := Run(); err == nil {
		t.Error("expected error because .fz.yaml exists")
	}
}
