package utils

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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

func TestExecRawNoop(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("ExecRaw is amd64-only")
	}
	ExecRaw([]byte{0xc3})
}

func TestExecRawXor(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("ExecRawXor is amd64-only")
	}
	data := []byte{0x1, 0x2, 0x3, 0x4}
	if got := ExecRawXor(data); got != 0x1^0x2^0x3^0x4 {
		t.Fatalf("ExecRawXor = %d, want %d", got, 0x1^0x2^0x3^0x4)
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
	if out != "verbose\n" && out != "verbose\r\n" {
		t.Errorf("verbose mode returned output %q, want 'verbose\n'", out)
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
		t.Fatalf("expected no error for safe symlink, got %v", err)
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

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	h1, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Error("hashes differ")
	}
	if len(h1) != 64 {
		t.Errorf("unexpected hash length %d", len(h1))
	}
}

func TestHashFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 64 {
		t.Errorf("unexpected empty hash length %d", len(h))
	}
}

func TestHashDirRejectsSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := t.TempDir()
	targetFile := filepath.Join(targetDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(tmpDir, "link.txt")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Skip("symlink not supported")
	}
	_, err := HashDir(tmpDir)
	if err != nil {
		t.Fatalf("expected no error for unsafe symlink (should be skipped), got %v", err)
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
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = wErr
	defer func() {
		os.Stderr = oldStderr
		wErr.Close()
		rErr.Close()
	}()

	ctx := context.Background()
	_, err = RunCommandSilent(ctx, true, "sh", "-c", "echo error >&2 && exit 1")
	if err == nil {
		t.Error("expected error")
	}
	if err := wErr.Close(); err != nil {
		t.Fatal(err)
	}

	var bufErr bytes.Buffer
	if _, err := bufErr.ReadFrom(rErr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bufErr.String(), "error") {
		t.Errorf("stderr output missing, got: %q", bufErr.String())
	}
}

func TestHashDataDigestZeroAllocs(t *testing.T) {
	data := []byte("aegis zero allocations")
	if _, err := HashDataDigest(data); err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(100, func() {
		_, _ = HashDataDigest(data)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations, got %f", allocs)
	}
}

func TestValidateCLIFuzz(t *testing.T) {
	randSrc := rand.New(rand.NewSource(1))
	for i := 0; i < 500; i++ {
		n := randSrc.Intn(64)
		buf := make([]byte, n)
		for j := 0; j < n; j++ {
			buf[j] = byte(randSrc.Intn(256))
		}
		_ = ValidateCLIArg(string(buf))
		_ = ValidateCLIPath(string(buf))
	}
	if err := ValidateCLIPath("../etc/passwd"); err == nil {
		t.Fatal("expected path traversal detection")
	}
}

func TestHashFileTamperDetected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tamper.bin")
	data := []byte("initial integrity data")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[0] ^= 0xff
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("hash did not change after tampering")
	}
}

func TestHashDirTamperDetected(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("two modified"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("directory hash did not change after tampering")
	}
}

func TestHashFileConcurrentResilience(t *testing.T) {
	path := filepath.Join(t.TempDir(), "parallel.bin")
	if err := os.WriteFile(path, []byte("resilience"), 0o600); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errCh := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_ = os.WriteFile(path, []byte(fmt.Sprintf("resilience-%d", i)), 0o600)
			}
			_, err := HashFile(path)
			if err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err == nil {
			continue
		}
		if err == ErrHashOpen || err == ErrHashRead {
			continue
		}
		t.Fatal(err)
	}
}

func TestEnsureInsideRootBlocksTraversal(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureInsideRoot(root, outside); err == nil {
		t.Fatal("expected outside root rejection")
	}
}
