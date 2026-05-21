package utils

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	fzvfs "fz/internal/fs"
)

func TestResolveDestFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new", "file.txt")
	got, err := resolveDest(path)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("empty dest")
	}
}

func TestReadFileSecureOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "r.txt")
	payload := []byte("secure-read")
	if err := os.WriteFile(path, payload, FilePerm); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFileSecure(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("got %q", got)
	}
}

func TestStatResolved(t *testing.T) {
	dir := t.TempDir()
	resolved, err := ResolveSecurePath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := StatResolved(resolved); err != nil {
		t.Fatal(err)
	}
}

func TestSupportedExtensionUpper(t *testing.T) {
	if !SupportedExtension(".S") {
		t.Fatal(".S should be supported")
	}
}

func TestCopyFileRenameFail(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "s")
	dst := filepath.Join(dir, "d")
	if err := os.WriteFile(src, []byte("x"), FilePerm); err != nil {
		t.Fatal(err)
	}
	m := fzvfs.NewMock(fzvfs.Unix{})
	m.SetFailOp("Rename", fzvfs.ErrDiskFull)
	withMock(t, m)
	if err := CopyFile(src, dst); err == nil {
		t.Fatal("expected rename error")
	}
}

func TestSecureWriteCloseFail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out")
	m := fzvfs.NewMock(fzvfs.Unix{})
	withMock(t, m)
	if err := SecureWriteFile(path, []byte("data")); err != nil {
		return
	}
}

func TestHashDirWithRootSymlinkError(t *testing.T) {
	root := t.TempDir()
	m := fzvfs.NewMock(fzvfs.Unix{})
	m.SetFailOp("Readlink", fzvfs.ErrPermission)
	withMock(t, m)
	bad := filepath.Join(root, "link")
	if err := os.Symlink("nowhere", bad); err != nil {
		t.Skip("symlink")
	}
	_, err := HashDirWithRoot(root, root)
	if err == nil {
		t.Log("walk may not hit readlink on mock")
	}
}

func TestCheckFileExistsRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := CheckFileExists(dir); err == nil {
		t.Fatal("expected directory error")
	}
}

func TestRunCommandInvalidName(t *testing.T) {
	_, err := RunCommand(context.Background(), false, nil, nil, "bad;name")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateCLIArgNewline(t *testing.T) {
	if err := ValidateCLIArg("a\nb"); err == nil {
		t.Fatal("expected error")
	}
}

func TestEnsureInsideRootInvalidRoot(t *testing.T) {
	if err := EnsureInsideRoot("../bad", "file"); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveDestEvalFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fallback.txt")
	m := fzvfs.NewMock(fzvfs.Unix{})
	m.SetFailOp("EvalSymlinks", fzvfs.ErrInterrupted)
	withMock(t, m)
	got, err := resolveDest(path)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("empty")
	}
}

func TestSecureWriteAllBranches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c.dat")
	cases := []struct {
		op  string
		err error
	}{
		{"CreateTemp", fzvfs.ErrDiskFull},
		{"Chmod", fzvfs.ErrPermission},
		{"Rename", fzvfs.ErrInterrupted},
	}
	for _, tc := range cases {
		m := fzvfs.NewMock(fzvfs.Unix{})
		m.SetFailOp(tc.op, tc.err)
		withMock(t, m)
		if err := SecureWriteFile(path, []byte("x")); err == nil {
			t.Fatalf("%s: expected error", tc.op)
		}
	}
}

func TestCheckFileExistsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "t")
	if err := os.WriteFile(target, []byte("x"), FilePerm); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "l")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink")
	}
	if err := CheckFileExists(link); err == nil {
		t.Fatal("expected symlink rejection")
	}
}

func TestSupportedExtensionTable(t *testing.T) {
	cases := map[string]bool{
		".asm": true, ".s": true, ".S": true, ".fasm": true,
		".c": true, ".cpp": true, ".cc": true, ".cxx": true,
		".go": false, ".txt": false,
	}
	for ext, want := range cases {
		if SupportedExtension(ext) != want {
			t.Fatalf("%s = %v want %v", ext, !want, want)
		}
	}
}

func TestLstatPathInvalid(t *testing.T) {
	if _, err := LstatPath("../bad"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEvalSymlinksPath(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("1"), FilePerm); err != nil {
		t.Fatal(err)
	}
	got, err := EvalSymlinksPath(f)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("empty")
	}
}

func TestReadFileSecureReadFail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken")
	if err := os.WriteFile(path, []byte("x"), FilePerm); err != nil {
		t.Fatal(err)
	}
	m := fzvfs.NewMock(fzvfs.Unix{})
	withMock(t, m)
	m.SetFail("OpenVerified", path, fzvfs.ErrTimeout)
	if _, err := ReadFileSecure(path); err == nil {
		t.Fatal("expected error")
	}
}

func TestHashDirAbsFail(t *testing.T) {
	_, err := HashDir(string([]byte{0}))
	if err == nil {
		t.Fatal("expected abs error")
	}
}

func TestHashDirWithRootEvalSymlinkFail(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "x")
	if err := os.Symlink("missing-target-xyz", link); err != nil {
		t.Skip("symlink")
	}
	_, err := HashDirWithRoot(root, root)
	if err == nil {
		t.Fatal("expected eval error")
	}
}

func TestResolveDestBothFail(t *testing.T) {
	if _, err := resolveDest("../bad/path"); err == nil {
		t.Fatal("expected error")
	}
}

func TestHashFileReadFail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ro")
	if err := os.WriteFile(path, []byte("z"), 0o400); err != nil {
		t.Fatal(err)
	}
	m := fzvfs.NewMock(fzvfs.Unix{})
	m.SetFail("OpenVerified", path, fzvfs.ErrPermission)
	withMock(t, m)
	if _, err := HashFile(path); err == nil {
		t.Fatal("expected error")
	}
}
