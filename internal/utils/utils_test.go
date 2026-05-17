package utils

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestIsWindows(t *testing.T) {
	got := IsWindows()
	expected := runtime.GOOS == "windows"
	if got != expected {
		t.Errorf("IsWindows() = %v, want %v", got, expected)
	}
}

func TestCheckTool(t *testing.T) {
	if err := CheckTool("go"); err != nil {
		t.Errorf("CheckTool(go) failed: %v", err)
	}
	if err := CheckTool("nonexistent_tool_xyz"); err == nil {
		t.Error("CheckTool(nonexistent) should fail")
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

func TestRunCommandSilentTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := RunCommandSilent(ctx, false, "sleep", "1")
	if err == nil {
		t.Error("expected timeout error")
	}
}
