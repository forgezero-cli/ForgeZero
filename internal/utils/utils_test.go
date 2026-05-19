package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type MockRunner struct {
	RunFunc func(ctx context.Context, verbose bool, name string, args ...string) (string, error)
}

func (m *MockRunner) Run(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, verbose, name, args...)
	}
	return "", nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func TestIsWindows(t *testing.T) {
	got := IsWindows()
	expected := runtime.GOOS == "windows"
	if got != expected {
		t.Errorf("IsWindows() = %v, want %v", got, expected)
	}
}

func TestEnsureDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "file.txt")
	if err := EnsureDir(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Error("Directory not created")
	}
	if err := EnsureDir(dir); err != nil {
		t.Error(err)
	}
}

func TestSupportedExtension(t *testing.T) {
	tests := []struct {
		ext  string
		want bool
	}{
		{".asm", true},
		{".s", true},
		{".S", true},
		{".fasm", true},
		{".c", true},
		{".go", false},
	}
	for _, tt := range tests {
		if got := SupportedExtension(tt.ext); got != tt.want {
			t.Errorf("SupportedExtension(%q) = %v, want %v", tt.ext, got, tt.want)
		}
	}
}

func TestDeriveNames(t *testing.T) {
	src := "test.asm"
	bin, obj := DeriveNames(src, "", "")
	expectedBin := "test"
	if runtime.GOOS == "windows" {
		expectedBin = "test.exe"
	}
	if bin != expectedBin {
		t.Errorf("bin = %v, want %v", bin, expectedBin)
	}
	if obj != "test.o" {
		t.Errorf("obj = %v, want test.o", obj)
	}
	bin, obj = DeriveNames(src, "myprog", "myobj.o")
	if bin != "myprog" || obj != "myobj.o" {
		t.Errorf("with flags: bin=%v obj=%v", bin, obj)
	}
}

func TestCheckFileExists(t *testing.T) {
	tmp, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if err := CheckFileExists(tmp.Name()); err != nil {
		t.Errorf("existing file: %v", err)
	}
	if err := CheckFileExists("nonexistent"); err == nil {
		t.Error("nonexistent file should error")
	}
	dir := t.TempDir()
	if err := CheckFileExists(dir); err == nil {
		t.Error("directory should be rejected")
	}
}

func TestRunCommandSilent(t *testing.T) {
	ctx := context.Background()
	out, err := RunCommandSilent(ctx, false, "echo", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello\n" && out != "hello\r\n" {
		t.Errorf("output = %q, want 'hello\\n'", out)
	}
	out, err = RunCommandSilent(ctx, true, "echo", "verbose")
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Errorf("verbose mode returned output %q, want ''", out)
	}
	_, err = RunCommandSilent(ctx, false, "false")
	if err == nil {
		t.Error("false command should fail")
	}
}

func TestCopyFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.txt")
	dst := filepath.Join(t.TempDir(), "dst.txt")
	content := []byte("hello")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(content) {
		t.Error("content mismatch")
	}
	// Non-existent source
	if err := CopyFile("nonexistent", dst); err == nil {
		t.Error("expected error")
	}
}

func TestEnsureDirErrors(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	err := EnsureDir(filepath.Join(file, "sub", "file"))
	if err == nil {
		t.Error("expected error because parent is a file")
	}
}

func TestHashDir(t *testing.T) {
	dir := t.TempDir()
	// Create files in subdirectories
	sub1 := filepath.Join(dir, "a")
	sub2 := filepath.Join(dir, "b")
	os.MkdirAll(sub1, 0o755)
	os.MkdirAll(sub2, 0o755)
	file1 := filepath.Join(sub1, "f1.txt")
	file2 := filepath.Join(sub2, "f2.txt")
	if err := os.WriteFile(file1, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash1, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	hash2, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if hash1 != hash2 {
		t.Error("hashes differ for same directory")
	}
	if err := os.WriteFile(file1, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash3, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if hash1 == hash3 {
		t.Error("hash didn't change after file modification")
	}
}

func TestHashDirEmpty(t *testing.T) {
	dir := t.TempDir()
	hash, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}
	hash2, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if hash != hash2 {
		t.Error("hashes differ for same empty dir")
	}
}

func TestCheckFileExistsDirectory(t *testing.T) {
	dir := t.TempDir()
	err := CheckFileExists(dir)
	if err == nil {
		t.Error("expected error for directory")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestEnsureDirAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureDir(filepath.Join(dir, "somefile")); err != nil {
		t.Fatal(err)
	}
}

func TestRunCommandSilentWithStderr(t *testing.T) {
	ctx := context.Background()
	output, err := RunCommandSilent(ctx, false, "sh", "-c", "echo error >&2; exit 1")
	if err == nil {
		t.Error("expected error")
	}
	if output == "" {
		t.Error("expected stderr output")
	}
}

func TestCheckTool(t *testing.T) {
	oldFunc := CheckToolFunc
	defer func() { CheckToolFunc = oldFunc }()

	CheckToolFunc = func(name string) error {
		return fmt.Errorf("mock error")
	}
	err := CheckTool("any")
	if err == nil {
		t.Error("expected error")
	}

	CheckToolFunc = oldFunc
	if err := CheckTool("go"); err != nil {
		t.Error("go should be found in PATH")
	}
}

func TestHashDirPermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission test not reliable on Windows")
	}
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o000); err != nil {
		t.Skip("cannot set permission")
	}
	defer os.Chmod(sub, 0o755)
	_, err := HashDir(sub)
	if err == nil {
		t.Error("expected error due to permission denied")
	}
}

func TestCopyFileSrcNotExist(t *testing.T) {
	err := CopyFile("nonexistent", "dst")
	if err == nil {
		t.Error("expected error")
	}
}

func TestRunCommandSilentTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := RunCommandSilent(ctx, false, "sleep", "1")
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestHashDirSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink not supported")
	}
	_, err := HashDir(dir)
	if err != nil {
		t.Error(err)
	}
}

func TestRunCommandSilentContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := RunCommandSilent(ctx, false, "sleep", "1")
	if err == nil {
		t.Error("expected error due to cancelled context")
	}
}

func TestCheckToolNotFound(t *testing.T) {
	err := CheckTool("_this_tool_should_not_exist_xyz_")
	if err == nil {
		t.Error("expected error")
	}
}

func TestDeriveNamesWithFlags(t *testing.T) {
	src := "test.asm"
	bin, obj := DeriveNames(src, "myprog", "myobj.o")
	if bin != "myprog" {
		t.Errorf("bin = %v, want myprog", bin)
	}
	if obj != "myobj.o" {
		t.Errorf("obj = %v, want myobj.o", obj)
	}
}

func TestSupportedExtensionAll(t *testing.T) {
	exts := []string{".asm", ".s", ".S", ".fasm", ".c", ".cpp", ".cc", ".cxx"}
	for _, ext := range exts {
		if !SupportedExtension(ext) {
			t.Errorf("SupportedExtension(%q) = false, want true", ext)
		}
	}
	if SupportedExtension(".go") {
		t.Error("SupportedExtension(.go) should be false")
	}
}

func TestRunCommandSilentErrorVerbose(t *testing.T) {
	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	ctx := context.Background()
	_, err := RunCommandSilent(ctx, true, "sh", "-c", "echo error >&2 && exit 1")

	wErr.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error")
	}

	var bufErr bytes.Buffer
	bufErr.ReadFrom(rErr)
	if !strings.Contains(bufErr.String(), "error") {
		t.Errorf("stderr output missing, got: %q", bufErr.String())
	}
}

// ----- Test linkWithGcc with no fallback and error (already exists, but ensure verbose branch) -----
