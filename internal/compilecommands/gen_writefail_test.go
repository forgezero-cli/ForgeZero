package compilecommands

import (
	"os"
	"path/filepath"
	"testing"

	"fz/internal/config"
	fzvfs "fz/internal/fs"
	"fz/internal/utils"
)

func TestGenerateSecureWriteFail(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	c := filepath.Join(dir, "m.c")
	if err := os.WriteFile(c, []byte("int main(){}"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFailOp("Rename", fzvfs.ErrDiskFull)
	utils.SetFileSystem(m)
	defer utils.SetFileSystem(nil)
	cfg := &config.Config{SourceFiles: []string{c}}
	if err := Generate(cfg, dir); err == nil {
		t.Fatal("expected write error")
	}
}
