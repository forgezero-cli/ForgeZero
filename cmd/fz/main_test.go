package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fz/internal/assembler"
	"fz/internal/linker"
	"fz/internal/utils"
)

type fakeCmdRunner struct{}

func (fakeCmdRunner) Run(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-o" {
			out := args[i+1]
			data := []byte("BINARY")
			if strings.Contains(filepath.Base(name), "nasm") || strings.Contains(filepath.Base(name), "as") || strings.Contains(filepath.Base(name), "clang") || strings.Contains(filepath.Base(name), "gcc") {
				data = []byte("OBJ")
			}
			if err := os.WriteFile(out, data, 0o755); err != nil {
				return "", err
			}
			return "", nil
		}
	}
	return "", nil
}

func captureOutput(t *testing.T, f func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	f()
	_ = w.Close()
	return <-outC
}

func runFzArgs(t *testing.T, args []string) string {
	oldArgs := os.Args
	oldFlags := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlags
	}()
	os.Args = args
	flag.CommandLine = flag.NewFlagSet(args[0], flag.ExitOnError)
	return captureOutput(t, func() {
		main()
	})
}

func TestFullCliFlowInitBuildSeal(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	oldCheck := utils.CheckToolFunc
	utils.CheckToolFunc = func(string) error { return nil }
	defer func() {
		utils.CheckToolFunc = oldCheck
	}()

	assembler.SetRunCommand(func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-o" {
				out := args[i+1]
				if err := os.WriteFile(out, []byte("OBJ"), 0o755); err != nil {
					return "", err
				}
				return "", nil
			}
		}
		return "", nil
	})
	defer assembler.SetRunCommand(nil)

	linker.SetRunner(fakeCmdRunner{})
	defer linker.ResetRunner()

	initOutput := runFzArgs(t, []string{"fz", "-init"})
	if !strings.Contains(initOutput, "project initialized") {
		t.Fatalf("unexpected init output: %s", initOutput)
	}

	mainAsm := filepath.Join(tmpDir, "main.asm")
	if err := os.WriteFile(mainAsm, []byte("section .text\nglobal main\nmain:\n    mov rax, 60\n    xor rdi, rdi\n    syscall\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	buildOutput := runFzArgs(t, []string{"fz", "-dir", ".", "-out", "app", "-mode", "raw", "-no-sanitize", "-keep-obj"})
	if !strings.Contains(buildOutput, "Built: app") {
		t.Fatalf("unexpected build output: %s", buildOutput)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "app")); err != nil {
		t.Fatalf("binary not created: %v", err)
	}

	versionOutput := runFzArgs(t, []string{"fz", "version"})
	if !strings.Contains(versionOutput, "[MIL-SPEC]") {
		t.Fatalf("unexpected version banner: %s", versionOutput)
	}

	sealOutput := runFzArgs(t, []string{"fz", "--seal"})
	if !strings.Contains(sealOutput, "seal written") {
		t.Fatalf("unexpected seal output: %s", sealOutput)
	}
}
