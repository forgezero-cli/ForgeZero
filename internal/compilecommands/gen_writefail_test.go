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
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)
	c := filepath.Join(dir, "m.c")
	os.WriteFile(c, []byte("int main(){}"), 0o644)
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFailOp("Rename", fzvfs.ErrDiskFull)
	utils.SetFileSystem(m)
	defer utils.SetFileSystem(nil)
	cfg := &config.Config{SourceFiles: []string{c}}
	if err := Generate(cfg, dir); err == nil {
		t.Fatal("expected write error")
	}
}
