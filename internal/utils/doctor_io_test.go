package utils

import (
	"os"
	"path/filepath"
	"testing"

	fzvfs "fz/internal/fs"
)

func TestRemovePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gone.txt")
	if err := os.WriteFile(path, []byte("x"), FilePerm); err != nil {
		t.Fatal(err)
	}
	if err := RemovePath(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file should be gone")
	}
}

func TestOpenVerifiedRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "r.txt")
	if err := os.WriteFile(path, []byte("data"), FilePerm); err != nil {
		t.Fatal(err)
	}
	f, err := OpenVerifiedRead(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestLookExecutableExported(t *testing.T) {
	_, err := LookExecutable("go")
	if err != nil {
		t.Skip("go not in path")
	}
}

func TestRemovePathMockFail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x")
	m := fzvfs.NewMock(fzvfs.Default)
	resolved, _ := ResolveSecurePath(path)
	m.SetFail("Remove", resolved, fzvfs.ErrPermission)
	SetFileSystem(m)
	defer ResetFileSystem()
	if err := RemovePath(path); err == nil {
		t.Fatal("expected error")
	}
}
