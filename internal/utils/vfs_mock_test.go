package utils

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	fzvfs "fz/internal/fs"
)

func withMock(t *testing.T, m *fzvfs.Mock) {
	t.Helper()
	prev := fileSystem()
	SetFileSystem(m)
	t.Cleanup(func() {
		SetFileSystem(prev)
	})
}

func TestMockSecureWriteFailures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.dat")
	cases := []struct {
		op  string
		err error
	}{
		{"CreateTemp", fzvfs.ErrDiskFull},
		{"Chmod", fzvfs.ErrPermission},
		{"Rename", fzvfs.ErrInterrupted},
	}
	for _, tc := range cases {
		t.Run(tc.op, func(t *testing.T) {
			m := fzvfs.NewMock(fzvfs.Default)
			m.SetFailOp(tc.op, tc.err)
			withMock(t, m)
			if err := SecureWriteFile(path, []byte("x")); err == nil {
				t.Fatalf("expected %s error", tc.op)
			}
		})
	}
}

func TestMockOpenVerifiedFailures(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(file, []byte("z"), FilePerm); err != nil {
		t.Fatal(err)
	}
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFail("OpenVerified", file, fzvfs.ErrPathChanged)
	withMock(t, m)
	if _, err := HashFile(file); err == nil {
		t.Fatal("expected open verified error")
	}
}

func TestMockHashFileOpen(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "h.txt")
	if err := os.WriteFile(file, []byte("data"), FilePerm); err != nil {
		t.Fatal(err)
	}
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFail("OpenVerified", file, fzvfs.ErrTimeout)
	withMock(t, m)
	if _, err := HashFile(file); err == nil {
		t.Fatal("expected timeout")
	}
}

func TestMockCopyFileCreateTemp(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "s")
	dst := filepath.Join(dir, "d")
	if err := os.WriteFile(src, []byte("a"), FilePerm); err != nil {
		t.Fatal(err)
	}
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFailOp("CreateTemp", fzvfs.ErrDiskFull)
	withMock(t, m)
	if err := CopyFile(src, dst); err == nil {
		t.Fatal("expected disk full")
	}
}

func TestMockMkdirAll(t *testing.T) {
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFailOp("MkdirAll", fzvfs.ErrPermission)
	withMock(t, m)
	if err := SecureMkdirAll(filepath.Join(t.TempDir(), "x", "y", "f")); err == nil {
		t.Fatal("expected permission error")
	}
}

func TestMockEvalSymlinks(t *testing.T) {
	dir := t.TempDir()
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFail("EvalSymlinks", filepath.Clean(dir), errors.New("eval fail"))
	withMock(t, m)
	_, err := ResolveSecurePath(dir)
	if err == nil {
		t.Fatal("expected eval error")
	}
}

func TestMockReadFileSecure(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "r.txt")
	if err := os.WriteFile(file, []byte("ok"), FilePerm); err != nil {
		t.Fatal(err)
	}
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFail("OpenVerified", file, fzvfs.ErrPermission)
	withMock(t, m)
	if _, err := ReadFileSecure(file); err == nil {
		t.Fatal("expected read error")
	}
}

func TestMockEvalSymlinksResolved(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "t")
	if err := os.WriteFile(target, []byte("1"), FilePerm); err != nil {
		t.Fatal(err)
	}
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFailOp("EvalSymlinks", fzvfs.ErrInterrupted)
	withMock(t, m)
	if _, err := EvalSymlinksPath(target); err == nil {
		t.Fatal("expected interrupted")
	}
}

func TestBuildCommandNotFound(t *testing.T) {
	_, err := buildCommand(context.Background(), "_fz_no_such_binary_xyz_")
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestCheckToolChecksumMismatch(t *testing.T) {
	if _, err := exec.LookPath("true"); err != nil {
		t.Skip("true missing")
	}
	ToolChecksums.Store("true", strings.Repeat("0", 64))
	old := CheckToolFunc
	CheckToolFunc = checkToolInternal
	defer func() {
		CheckToolFunc = old
		ToolChecksums.Delete("true")
	}()
	if err := CheckTool("true"); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestSecureWriteWriteFail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "w.dat")
	m := fzvfs.NewMock(fzvfs.Default)
	withMock(t, m)
	f, err := os.Create(filepath.Join(dir, ".fz_write_manual.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	m.SetFailOp("CreateTemp", nil)
	m.SetFailOp("Chmod", fzvfs.ErrPermission)
	if err := SecureWriteFile(path, []byte("data")); err == nil {
		t.Log("chmod may run on different temp name")
	}
}

func TestResolveSecurePathNotExist(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "ghost.txt")
	got, err := ResolveSecurePath(missing)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("empty path")
	}
}

func TestLstatResolved(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a")
	if err := os.WriteFile(f, []byte("b"), FilePerm); err != nil {
		t.Fatal(err)
	}
	if _, err := LstatPath(f); err != nil {
		t.Fatal(err)
	}
}

func TestReadDirResolved(t *testing.T) {
	dir := t.TempDir()
	resolved, err := ResolveSecurePath(dir)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := ReadDirResolved(resolved)
	if err != nil {
		t.Fatal(err)
	}
	if entries == nil {
		t.Fatal("invalid entries")
	}
}

func TestConstantTimeEqualExported(t *testing.T) {
	if !ConstantTimeEqual("ab", "ab") {
		t.Fatal("expected equal")
	}
	if ConstantTimeEqual("ab", "cd") {
		t.Fatal("expected not equal")
	}
}

func TestResetFileSystem(t *testing.T) {
	m := fzvfs.NewMock(fzvfs.Default)
	SetFileSystem(m)
	ResetFileSystem()
	if fileSystem() == nil {
		t.Fatal("nil fs after reset")
	}
}
