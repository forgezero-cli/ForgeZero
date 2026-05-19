package shell

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{`build`, []string{"build"}},
		{`set mode=raw`, []string{"set", "mode=raw"}},
		{`set "ld-script=linker.ld"`, []string{"set", "ld-script=linker.ld"}},
		{`show`, []string{"show"}},
		{`exit`, []string{"exit"}},
	}
	for _, tt := range tests {
		got := splitCommand(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitCommand(%q) = %v, want %v", tt.input, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitCommand(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestCmdHelp(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	cmdHelp()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Commands:") {
		t.Error("help output missing 'Commands:'")
	}
}

func TestCmdShow(t *testing.T) {
	state := DefaultState()
	state.Mode = "raw"
	state.Format = "bin"
	state.Strict = true
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	cmdShow(state)
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Mode: raw") {
		t.Error("show output missing Mode")
	}
	if !strings.Contains(out, "Format: bin") {
		t.Error("show output missing Format")
	}
}

func TestCmdSet(t *testing.T) {
	state := DefaultState()
	args := []string{"set", "mode=raw"}
	if err := cmdSet(state, args); err != nil {
		t.Fatal(err)
	}
	if state.Mode != "raw" {
		t.Errorf("expected mode raw, got %s", state.Mode)
	}
	args = []string{"set", "strict=true"}
	if err := cmdSet(state, args); err != nil {
		t.Fatal(err)
	}
	if !state.Strict {
		t.Error("expected strict true")
	}
	args = []string{"set", "invalid"}
	if err := cmdSet(state, args); err == nil {
		t.Error("expected error for invalid set syntax")
	}
}

func TestCmdBuild(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	if _, err := exec.LookPath("ld"); err != nil {
		t.Skip("ld not installed")
	}
	state := DefaultState()
	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "test.asm")
	err := os.WriteFile(src, []byte(`
section .text
global _start
_start:
	mov eax, 60
	xor edi, edi
	syscall
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	state.SourcePath = src
	state.SourceType = "asm"
	state.Out = filepath.Join(tempDir, "myapp")
	state.Format = "elf64"
	state.Mode = "raw"
	state.Verbose = true
	state.Debug = false
	state.NoSymbolCheck = false
	state.Sanitize = false
	state.Strict = false
	state.KeepObj = false
	state.NoCache = false
	err = cmdBuild(state)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if _, err := os.Stat(state.Out); err != nil {
		t.Error("output binary not created")
	}
	info, err := os.Stat(state.Out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Error("output binary is not executable")
	}
}

func TestCmdClean(t *testing.T) {
	state := DefaultState()
	dir := t.TempDir()
	state.SourcePath = dir
	state.SourceType = "dir"
	objDir := filepath.Join(dir, ".fz_objs")
	cacheDir := filepath.Join(dir, ".fz_cache")
	if err := os.MkdirAll(objDir, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", objDir, err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("failed to create cache dir %s: %v", cacheDir, err)
	}
	err := cmdClean(state)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(objDir); !os.IsNotExist(err) {
		t.Error(".fz_objs not removed")
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Error(".fz_cache not removed")
	}
}
