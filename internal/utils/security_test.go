package utils

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestConstantTimeEqual(t *testing.T) {
	cases := []struct {
		a, b   string
		expect bool
	}{
		{"abc", "abc", true},
		{"abc", "abd", false},
		{"abc", "ab", false},
		{"", "", true},
	}
	for _, tc := range cases {
		got := constantTimeEqual(tc.a, tc.b)
		if got != tc.expect {
			t.Errorf("constantTimeEqual(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.expect)
		}
	}
}

func TestSecureWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")
	data := []byte("classified")
	if err := SecureWriteFile(path, data); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != FilePerm {
		t.Errorf("perm = %o, want %o", info.Mode().Perm(), FilePerm)
	}
	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(read) != string(data) {
		t.Errorf("content = %q, want %q", read, data)
	}
}

func TestSecureMkdirAll(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c", "file")
	if err := SecureMkdirAll(nested); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(dir, "a", "b", "c")
	info, err := os.Stat(parent)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != DirPerm {
		t.Errorf("dir perm = %o, want %o", info.Mode().Perm(), DirPerm)
	}
}

func TestResolveSecurePath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(file, []byte("x"), FilePerm); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveSecurePath(file)
	if err != nil {
		t.Fatal(err)
	}
	if resolved == "" {
		t.Fatal("empty resolved path")
	}
}

func TestRunCommandOutput(t *testing.T) {
	ctx := context.Background()
	out, err := RunCommandOutput(ctx, "echo", "aegis")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestValidateCLIPathTraversal(t *testing.T) {
	cases := []struct {
		path    string
		invalid bool
	}{
		{"safe/path", false},
		{"../etc/passwd", true},
		{"/foo/../bar", true},
		{"normal.txt", false},
	}
	for _, tc := range cases {
		err := ValidateCLIPath(tc.path)
		if tc.invalid && err == nil {
			t.Errorf("ValidateCLIPath(%q) expected error", tc.path)
		}
		if !tc.invalid && err != nil {
			t.Errorf("ValidateCLIPath(%q) unexpected error: %v", tc.path, err)
		}
	}
}

func TestEnsureInsideRoot(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "src", "main.c")
	if err := os.MkdirAll(filepath.Dir(inside), DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inside, []byte("x"), FilePerm); err != nil {
		t.Fatal(err)
	}
	if err := EnsureInsideRoot(root, inside); err != nil {
		t.Errorf("inside path rejected: %v", err)
	}
	outside := filepath.Join(os.TempDir(), "outside-fz-test")
	if err := EnsureInsideRoot(root, outside); err == nil {
		t.Error("expected outside path rejection")
	}
}

func TestValidateCLIArg(t *testing.T) {
	cases := []struct {
		arg     string
		invalid bool
	}{
		{"-O2", false},
		{"$(whoami)", true},
		{"arg;rm", true},
		{"", false},
	}
	for _, tc := range cases {
		err := ValidateCLIArg(tc.arg)
		if tc.invalid && err == nil {
			t.Errorf("ValidateCLIArg(%q) expected error", tc.arg)
		}
		if !tc.invalid && err != nil {
			t.Errorf("ValidateCLIArg(%q): %v", tc.arg, err)
		}
	}
}

func TestValidateFlagTokens(t *testing.T) {
	tokens, err := ValidateFlagTokens([]byte("-O2 -Wall"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 2 {
		t.Fatalf("tokens = %v", tokens)
	}
	_, err = ValidateFlagTokens([]byte("-O2;rm"))
	if err == nil {
		t.Error("expected invalid flag token error")
	}
}

func TestSetExecutionRoot(t *testing.T) {
	dir := t.TempDir()
	prev := GetExecutionRoot()
	SetExecutionRoot(dir)
	t.Cleanup(func() { SetExecutionRoot(prev) })
	if GetExecutionRoot() != dir {
		t.Errorf("execution root = %q, want %q", GetExecutionRoot(), dir)
	}
}

func TestCheckToolWithChecksum(t *testing.T) {
	if _, err := exec.LookPath("echo"); err != nil {
		t.Skip("echo not available")
	}
	path, _ := exec.LookPath("echo")
	hash, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ToolChecksums.Store("echo", hash)
	old := CheckToolFunc
	CheckToolFunc = checkToolInternal
	defer func() {
		CheckToolFunc = old
		ToolChecksums.Delete("echo")
	}()
	if err := CheckTool("echo"); err != nil {
		t.Errorf("valid checksum rejected: %v", err)
	}
	ToolChecksums.Store("echo", "0000000000000000000000000000000000000000000000000000000000000000")
	if err := CheckTool("echo"); err == nil {
		t.Error("expected checksum mismatch")
	}
}

func TestCopyFileSecure(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	if err := os.WriteFile(src, []byte("payload"), FilePerm); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != FilePerm {
		t.Errorf("dst perm = %o", info.Mode().Perm())
	}
}

func TestHashDirWithRoot(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "lib")
	if err := os.MkdirAll(sub, DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.txt"), []byte("a"), FilePerm); err != nil {
		t.Fatal(err)
	}
	h1, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("b"), FilePerm); err != nil {
		t.Fatal(err)
	}
	h2, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Error("hash should change when files added")
	}
}

func TestRunCommandWithStdout(t *testing.T) {
	ctx := context.Background()
	var buf strings.Builder
	_, err := RunCommand(ctx, false, &buf, nil, "echo", "stream")
	if err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Error("expected captured stdout")
	}
}

func TestCheckToolNilFunc(t *testing.T) {
	old := CheckToolFunc
	CheckToolFunc = nil
	defer func() { CheckToolFunc = old }()
	if err := CheckTool("go"); err != nil {
		t.Errorf("CheckTool with nil func: %v", err)
	}
}

func TestGetExecutionRootEmpty(t *testing.T) {
	prev := GetExecutionRoot()
	SetExecutionRoot("")
	t.Cleanup(func() { SetExecutionRoot(prev) })
	if GetExecutionRoot() != "" {
		t.Error("expected empty execution root")
	}
}

func TestBuildCommandRejectsShellMetachar(t *testing.T) {
	_, err := buildCommand(context.Background(), "echo", "bad;cmd")
	if err == nil {
		t.Error("expected invalid arg error")
	}
}

func TestResolveSecurePathInvalid(t *testing.T) {
	_, err := ResolveSecurePath("../etc/passwd")
	if err == nil {
		t.Error("expected traversal rejection")
	}
}

func TestHashDirWithRootOutsideSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	outFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outFile, []byte("x"), FilePerm); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outFile, link); err != nil {
		t.Skip("symlink unsupported")
	}
	rootAbs, _ := filepath.Abs(root)
	_, err := HashDirWithRoot(rootAbs, rootAbs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCLIPathInvalidChars(t *testing.T) {
	cases := []string{"path|pipe", "path`tick", "path\x00null"}
	for _, p := range cases {
		if err := ValidateCLIPath(p); err == nil {
			t.Errorf("ValidateCLIPath(%q) expected error", p)
		}
	}
}

func TestSecureWriteFileNested(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deep", "nested", "out.json")
	payload := []byte(`{"aegis":true}`)
	if err := SecureWriteFile(path, payload); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("got %q", got)
	}
}

func TestCopyFileCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.dat")
	dst := filepath.Join(dir, "sub", "b.dat")
	if err := os.WriteFile(src, []byte("copy-me"), FilePerm); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(src, dst); err != nil {
		t.Fatal(err)
	}
}

func TestCheckFileExistsMissing(t *testing.T) {
	if err := CheckFileExists(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestHashDirWithRootWalkError(t *testing.T) {
	root := t.TempDir()
	bad := filepath.Join(root, "nodir", "file.txt")
	_, err := HashDirWithRoot(root, bad)
	if err == nil {
		t.Error("expected walk error")
	}
}

func TestCheckToolInternalNoChecksumEntry(t *testing.T) {
	if _, err := exec.LookPath("true"); err != nil {
		t.Skip("true not in PATH")
	}
	ToolChecksums.Delete("true")
	if err := checkToolInternal("true"); err != nil {
		t.Errorf("tool without checksum entry: %v", err)
	}
}

func TestHashDirWithRootSymlinkRead(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, DirPerm); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(sub, "t.txt")
	if err := os.WriteFile(target, []byte("1"), FilePerm); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(sub, "l.txt")
	if err := os.Symlink("t.txt", link); err != nil {
		t.Skip("symlink unsupported")
	}
	rootAbs, _ := filepath.Abs(root)
	h, err := HashDirWithRoot(rootAbs, sub)
	if err != nil {
		t.Fatal(err)
	}
	if h == "" {
		t.Fatal("empty hash")
	}
}

func TestBuildCommandEmptyName(t *testing.T) {
	_, err := buildCommand(context.Background(), "")
	if err == nil {
		t.Error("expected empty name error")
	}
}

func TestBuildCommandShDashC(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh missing")
	}
	cmd, err := buildCommand(context.Background(), "sh", "-c", "echo ok")
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil {
		t.Fatal("nil cmd")
	}
}
