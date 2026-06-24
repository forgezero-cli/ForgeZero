package zig

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func TestLinkSuccessMocked(t *testing.T) {
	oldRun := RunCommand
	oldFunc := utils.CheckToolFunc
	defer func() { RunCommand = oldRun; utils.CheckToolFunc = oldFunc }()
	utils.CheckToolFunc = func(string) error { return nil }
	ZigRequested = true
	defer func() { ZigRequested = false }()
	RunCommand = func(ctx context.Context, verbose bool, args ...string) (string, error) {
		return "", nil
	}
	if err := Link(context.Background(), []string{"a.o"}, "out", false, "x86_64-linux-gnu", false, false, nil, false, "", "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestLinkVerboseFail(t *testing.T) {
	oldRun := RunCommand
	oldFunc := utils.CheckToolFunc
	defer func() { RunCommand = oldRun; utils.CheckToolFunc = oldFunc }()
	utils.CheckToolFunc = func(string) error { return nil }
	ZigEnabled = true
	defer func() { ZigEnabled = false }()
	RunCommand = func(ctx context.Context, verbose bool, args ...string) (string, error) {
		return "detail", errors.New("link fail")
	}
	err := Link(context.Background(), []string{"a.o"}, "out", true, "x86_64-linux-gnu", true, true, []string{"m"}, false, "s.ld", "0x1000", "-O2")
	if err == nil || !strings.Contains(err.Error(), "zig failed") {
		t.Fatalf("got %v", err)
	}
}

func TestLinkNonVerboseFail(t *testing.T) {
	oldRun := RunCommand
	oldFunc := utils.CheckToolFunc
	defer func() { RunCommand = oldRun; utils.CheckToolFunc = oldFunc }()
	utils.CheckToolFunc = func(string) error { return nil }
	ZigRequested = true
	defer func() { ZigRequested = false }()
	RunCommand = func(ctx context.Context, verbose bool, args ...string) (string, error) {
		return "", errors.New("fail")
	}
	err := Link(context.Background(), []string{"a.o"}, "out", false, "x86_64-linux-gnu", false, false, nil, false, "", "", "")
	if err == nil || !strings.Contains(err.Error(), "verbose") {
		t.Fatalf("got %v", err)
	}
}

func TestCompileVerboseFail(t *testing.T) {
	oldRun := RunCommand
	oldFunc := utils.CheckToolFunc
	defer func() { RunCommand = oldRun; utils.CheckToolFunc = oldFunc }()
	utils.CheckToolFunc = func(string) error { return nil }
	ZigEnabled = true
	defer func() { ZigEnabled = false }()
	RunCommand = func(ctx context.Context, verbose bool, args ...string) (string, error) {
		return "out", errors.New("compile fail")
	}
	err := Compile(context.Background(), "main.c", "main.o", true, true, "wasm32-unknown-unknown", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLinkArgsSharedAndSanitize(t *testing.T) {
	args := LinkArgs([]string{"a.o"}, "bin", "x86_64-linux-gnu", true, true, []string{"pthread"}, true, "", "")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-shared") || !strings.Contains(joined, "-fsanitize=address") {
		t.Fatal(joined)
	}
}

func TestCompileArgsEmptyTarget(t *testing.T) {
	args := CompileArgs("a.c", "a.o", false, "", ".c", "")
	if !strings.Contains(strings.Join(args, " "), "x86_64-linux-gnu") {
		t.Fatal(args)
	}
}

func TestShouldUseZigEnabled(t *testing.T) {
	old := ZigEnabled
	ZigEnabled = true
	defer func() { ZigEnabled = old }()
	if !shouldUseZig() {
		t.Fatal()
	}
}
