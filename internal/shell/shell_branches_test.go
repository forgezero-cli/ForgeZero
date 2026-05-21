package shell

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c-bata/go-prompt"
)

func docWithText(t *testing.T, text string) prompt.Document {
	t.Helper()
	b := prompt.NewBuffer()
	b.InsertText(text, false, true)
	return *b.Document()
}

func TestCompleterEmptyWord(t *testing.T) {
	if len(Completer(docWithText(t, ""))) != 0 {
		t.Fatal("expected empty suggestions")
	}
}

func TestCompleterCommandPrefix(t *testing.T) {
	sugs := Completer(docWithText(t, "bu"))
	if len(sugs) == 0 {
		t.Fatal("expected build suggestion")
	}
}

func TestCompleterSetCommand(t *testing.T) {
	sugs := Completer(docWithText(t, "set"))
	if len(sugs) == 0 {
		t.Fatal("expected set suggestion")
	}
}

func TestRunExecutorBranches(t *testing.T) {
	oldPromptNew := promptNew
	defer func() { promptNew = oldPromptNew }()
	var executor func(string)
	promptNew = func(exec func(string), completer func(prompt.Document) []prompt.Suggest, opts ...prompt.Option) interface{ Run() } {
		executor = exec
		return fakePrompt{}
	}
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	Run()
	if executor == nil {
		w.Close()
		os.Stdout = oldStdout
		t.Fatal("executor not captured")
	}
	executor("")
	executor("help")
	executor("show")
	executor("watch")
	executor("unknown-cmd")
	executor("set badformat")
	executor("set mode=raw")
	state := DefaultState()
	state.SourcePath = ""
	executor("build")
	state.SourceType = "dir"
	state.SourcePath = t.TempDir()
	executor("build")
	state.SourceType = "file"
	state.SourcePath = filepath.Join(t.TempDir(), "x.txt")
	executor("build")
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "fz interactive shell") {
		t.Fatal("missing banner")
	}
}

func TestCmdBuildDirPath(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	state := DefaultState()
	dir := t.TempDir()
	writeASM := func(name, body string) {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte(body), 0o644)
	}
	writeASM("main.asm", `
section .text
global _start
_start:
	mov eax, 60
	xor edi, edi
	syscall
`)
	state.SourceType = "dir"
	state.SourcePath = dir
	state.Out = filepath.Join(t.TempDir(), "out")
	state.Mode = "raw"
	state.Verbose = true
	state.NoCache = true
	state.NoSymbolCheck = true
	if err := cmdBuild(state); err != nil {
		t.Fatal(err)
	}
}

func TestCmdBuildUnsupportedExt(t *testing.T) {
	state := DefaultState()
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	os.WriteFile(p, []byte("x"), 0o644)
	state.SourcePath = p
	state.SourceType = "file"
	err := cmdBuild(state)
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("got %v", err)
	}
}

func TestCmdBuildVerboseC(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	state := DefaultState()
	dir := t.TempDir()
	src := filepath.Join(dir, "main.c")
	os.WriteFile(src, []byte("int main(){return 0;}"), 0o644)
	state.SourcePath = src
	state.SourceType = "file"
	state.Out = filepath.Join(dir, "app")
	state.Mode = "c"
	state.Verbose = true
	if err := cmdBuild(state); err != nil {
		t.Fatal(err)
	}
}

func TestCompleterForSetCommand(t *testing.T) {
	sugs := Completer(docWithText(t, "set"))
	if len(sugs) == 0 {
		t.Fatal("expected suggestions")
	}
}

func TestCmdSetAllKeys(t *testing.T) {
	state := DefaultState()
	keys := []string{
		"format=elf64", "sanitize=true", "verbose=true", "debug=true",
		"no-cache=true", "no-symbol-check=true", "keep-obj=true",
		"ld-script=ld.ld", "text-addr=0x1000", "out=/tmp/x",
	}
	for _, kv := range keys {
		parts := strings.SplitN(kv, "=", 2)
		if err := cmdSet(state, []string{"set", kv}); err != nil {
			t.Fatalf("%s: %v", kv, err)
		}
		_ = parts
	}
}

func TestCmdCleanWrongSourceType(t *testing.T) {
	state := DefaultState()
	state.SourceType = "file"
	if err := cmdClean(state); err == nil {
		t.Fatal("expected error")
	}
}

func writeASMFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
