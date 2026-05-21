package initpkg

import (
	"os"
	"testing"

	fzvfs "fz/internal/fs"
	"fz/internal/utils"
)

func TestRunWriteFzYamlFail(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)
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
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFailOp("CreateTemp", fzvfs.ErrPermission)
	utils.SetFileSystem(m)
	defer utils.SetFileSystem(nil)
	if err := Run(); err == nil {
		t.Fatal("expected write error")
	}
}
