package initpkg

import (
	"os"
	"testing"

	fzvfs "fz/internal/fs"
	"fz/internal/utils"
)

func TestRunWriteFzYamlFail(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFailOp("CreateTemp", fzvfs.ErrDiskFull)
	utils.SetFileSystem(m)
	defer utils.SetFileSystem(nil)
	if err := Run(); err == nil {
		t.Fatal("expected write error")
	}
}

func TestRunWriteReadmeFail(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFailOp("CreateTemp", fzvfs.ErrPermission)
	utils.SetFileSystem(m)
	defer utils.SetFileSystem(nil)
	if err := Run(); err == nil {
		t.Fatal("expected write error")
	}
}
