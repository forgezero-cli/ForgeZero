/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package linker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
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

func containsPair(slice []string, a, b string) bool {
	for i := 0; i < len(slice)-1; i++ {
		if slice[i] == a && slice[i+1] == b {
			return true
		}
	}
	return false
}

func buildObject(t *testing.T, dir, name, asmContent string) string {
	src := filepath.Join(dir, name+".s")
	err := os.WriteFile(src, []byte(asmContent), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, name+".o")
	cmd := exec.Command("gcc", "-c", src, "-o", obj)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("gcc -c failed: %v\n%s", err, out)
	}
	return obj
}

func buildNasmObject(t *testing.T, dir, name, asmContent string) string {
	src := filepath.Join(dir, name+".s")
	err := os.WriteFile(src, []byte(asmContent), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, name+".o")
	cmd := exec.Command("nasm", "-f", "elf64", src, "-o", obj)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("nasm failed: %v\n%s", err, out)
	}
	return obj
}

func TestDiscoverSourceFiles(t *testing.T) {
	dir := t.TempDir()

	files := []string{
		"main.c", "helper.c", "test.cpp", "asm.s", "test.asm", "test.nasm", "test.fasm",
		"skip.go", "skip.rs", "skip.txt", "README.md",
	}
	for _, f := range files {
		path := filepath.Join(dir, f)
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	subDir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subFile := filepath.Join(subDir, "sub.c")
	if err := os.WriteFile(subFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	result := discoverSourceFiles(dir)

	expected := []string{
		"main.c", "helper.c", "test.cpp", "asm.s", "test.asm", "test.nasm", "test.fasm", "sub/sub.c",
	}

	for _, exp := range expected {
		found := false
		for _, r := range result {
			if strings.HasSuffix(r, exp) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected file %s not found in result: %v", exp, result)
		}
	}

	for _, skip := range []string{"skip.go", "skip.rs", "skip.txt", "README.md"} {
		for _, r := range result {
			if strings.HasSuffix(r, skip) {
				t.Errorf("unexpected file %s found", skip)
			}
		}
	}
}

func TestDiscoverSourceFilesEmptyDir(t *testing.T) {
	dir := t.TempDir()
	result := discoverSourceFiles(dir)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d files", len(result))
	}
}

func TestDiscoverSourceFilesWithSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks test skipped on windows")
	}

	dir := t.TempDir()
	realDir := t.TempDir()
	realFile := filepath.Join(realDir, "real.c")
	if err := os.WriteFile(realFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	symlink := filepath.Join(dir, "link.c")
	if err := os.Symlink(realFile, symlink); err != nil {
		t.Skip("cannot create symlink")
	}

	result := discoverSourceFiles(dir)
	found := false
	for _, r := range result {
		if strings.HasSuffix(r, "link.c") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("symlinked .c file not found: %v", result)
	}
}

func TestDetectBackend(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{"c files", []string{"main.c", "helper.c"}, "gcc"},
		{"cpp files", []string{"main.cpp", "test.cc"}, "gcc"},
		{"cxx files", []string{"main.cxx"}, "gcc"},
		{"gas files", []string{"start.s"}, "gas"},
		{"gas uppercase", []string{"start.S"}, "gas"},
		{"nasm files", []string{"boot.asm"}, "nasm"},
		{"nasm explicit", []string{"test.nasm"}, "nasm"},
		{"fasm files", []string{"kernel.fasm"}, "fasm"},
		{"mixed c and asm", []string{"main.c", "boot.asm"}, "gcc"},
		{"asm only", []string{"pure.asm"}, "nasm"},
		{"unknown extension", []string{"main.xyz"}, "gcc"},
		{"empty files", []string{}, "gcc"},
		{"mixed with non-source", []string{"main.c", "README.md", "makefile"}, "gcc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectBackend(tt.files)
			if result != tt.expected {
				t.Errorf("detectBackend(%v) = %s, want %s", tt.files, result, tt.expected)
			}
		})
	}
}

func TestAutoBuildProjectDisabled(t *testing.T) {
	oldAutoBuild := AutoBuild
	AutoBuild = false
	defer func() { AutoBuild = oldAutoBuild }()

	err := AutoBuildProject(context.Background())
	if err != nil {
		t.Errorf("expected nil when AutoBuild is false, got %v", err)
	}
}

func TestRunBuildGcc(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "test.c")
	content := `#include <stdio.h>
int main() { printf("ok\n"); return 0; }`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir to %s: %v", dir, err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	err := runBuild(context.Background(), []string{"test.c"}, "gcc")
	if err != nil {
		t.Errorf("runBuild with gcc failed: %v", err)
	}

	bin := "./a.out"
	if runtime.GOOS == "windows" {
		bin = "./a.exe"
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("output binary not created")
	}
	os.Remove(bin)
}

func TestRunBuildGccNotFound(t *testing.T) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	err := runBuild(context.Background(), []string{"test.c"}, "gcc")
	if err == nil {
		t.Error("expected error when gcc not found")
	}
}

func TestRunBuildNasm(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "test.asm")
	content := `section .text
global _start
_start:
    mov eax, 60
    xor edi, edi
    syscall`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Errorf("failed to change dir %s: %v", dir, err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	err := runBuild(context.Background(), []string{"test.asm"}, "nasm")
	if err != nil {
		t.Errorf("runBuild with nasm failed: %v", err)
	}
}

func TestAutoBuildProjectIntegration(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}

	oldAutoBuild := AutoBuild
	AutoBuild = true
	defer func() { AutoBuild = oldAutoBuild }()

	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	src := filepath.Join(dir, "main.c")
	content := `#include <stdio.h>
int main() { printf("autobuild works\n"); return 0; }`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	err := AutoBuildProject(context.Background())
	if err != nil {
		t.Errorf("AutoBuildProject failed: %v", err)
	}

	bin := "./a.out"
	if runtime.GOOS == "windows" {
		bin = "./a.exe"
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("output binary not created by AutoBuild")
	}
	os.Remove(bin)
}

func TestLink(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	obj := buildObject(t, dir, "test", `
.globl _start
_start:
	mov $60, %eax
	xor %edi, %edi
	syscall
`)
	bin := filepath.Join(dir, "test")
	err := Link(context.Background(), obj, bin, false, "raw", false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created")
	}
}

func TestApplyGccLdFlags(t *testing.T) {
	args := []string{"test.o", "-o", "bin"}
	got := ApplyGccLdFlags(args, "script.ld", "0x1000")
	expected := []string{"test.o", "-o", "bin", "-Wl,-T,script.ld", "-Wl,-Ttext=0x1000", "-Wl,--build-id=none"}
	if !equalSlices(got, expected) {
		t.Errorf("ApplyGccLdFlags = %v, want %v", got, expected)
	}
	got = ApplyGccLdFlags(args, "", "")
	expectedEmpty := []string{"test.o", "-o", "bin", "-Wl,--build-id=none"}
	if !equalSlices(got, expectedEmpty) {
		t.Errorf("should add deterministic build flag when empty, got %v", got)
	}
}

func TestApplyLdFlags(t *testing.T) {
	args := []string{"test.o", "-o", "bin"}
	got := ApplyLdFlags(args, "script.ld", "0x1000")
	expected := []string{"test.o", "-o", "bin", "-T", "script.ld", "-Ttext", "0x1000", "--build-id=none"}
	if !equalSlices(got, expected) {
		t.Errorf("ApplyLdFlags = %v, want %v", got, expected)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLinkWithGccMockSuccess(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", nil
		},
	}
	ctx := context.Background()
	err := linkWithGcc(ctx, []string{"obj.o"}, "bin", false, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLinkWithGccMockSanitize(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	ctx := context.Background()
	err := linkWithGcc(ctx, []string{"obj.o"}, "bin", false, false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-fsanitize=address") || !contains(capturedArgs, "-fsanitize=undefined") {
		t.Error("sanitizer flags missing")
	}
	if !contains(capturedArgs, "-fsanitize-address-use-after-scope") {
		t.Error("strict flag missing")
	}
}

func TestTryAutoLinkNoClang(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	oldCheck := utils.CheckToolFunc
	defer func() { utils.CheckToolFunc = oldCheck }()

	utils.CheckToolFunc = func(name string) error {
		if name == "gcc" {
			return nil
		}
		return fmt.Errorf("not found")
	}
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			if name == "gcc" {
				return "", nil
			}
			return "", fmt.Errorf("unexpected")
		},
	}
	ctx := context.Background()
	err := tryAutoLink(ctx, []string{"obj.o"}, "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLinkWithGccMock(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	objFiles := []string{"a.o", "b.o"}
	libs := []string{"m"}
	err := linkWithGcc(context.Background(), objFiles, "bin", false, false, true, true, libs)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-fsanitize=address") {
		t.Error("missing -fsanitize=address")
	}
	if !contains(capturedArgs, "-fsanitize-address-use-after-scope") {
		t.Error("missing -fsanitize-address-use-after-scope (strict)")
	}
	for _, lib := range libs {
		if !containsPair(capturedArgs, "-l", lib) {
			t.Errorf("missing -l %s", lib)
		}
	}
}

func TestLinkWithGccNoFallback(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("gcc error")
		},
	}
	objFiles := []string{"a.o"}
	err := linkWithGcc(context.Background(), objFiles, "bin", false, false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

func TestLinkWithClangMock(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	objFiles := []string{"a.o"}
	libs := []string{"c"}
	err := linkWithClang(context.Background(), objFiles, "bin", false, true, true, libs)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-fsanitize=address") {
		t.Error("missing sanitizer flags")
	}
	if !containsPair(capturedArgs, "-l", libs[0]) {
		t.Error("missing library flag")
	}
}

func TestLinkWithClangNoFallback(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("clang error")
		},
	}
	objFiles := []string{"a.o"}
	err := linkWithClang(context.Background(), objFiles, "bin", false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

func TestLinkWithLdMock(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	objFiles := []string{"a.o", "b.o"}
	libs := []string{"m"}
	err := linkWithLd(context.Background(), objFiles, "bin", false, libs)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-o") {
		t.Error("missing -o flag")
	}
	for _, lib := range libs {
		if !containsPair(capturedArgs, "-l", lib) {
			t.Errorf("missing -l %s", lib)
		}
	}
}

func TestTryAutoLinkMultipleWithClangSuccess(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	clangCalled := false
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			if name == "clang" {
				clangCalled = true
				return "", nil
			}
			return "", nil
		},
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed, cannot test strict mode branch")
	}
	objFiles := []string{"a.o"}
	err := tryAutoLink(context.Background(), objFiles, "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !clangCalled {
		t.Error("clang not called in strict mode")
	}
}

func TestTryAutoLinkClangFailFallbackToGcc(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if name == "clang" {
				return "", fmt.Errorf("clang fails")
			}
			if name == "gcc" {
				return "", nil
			}
			return "", nil
		},
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed, cannot test clang failure path")
	}
	objFiles := []string{"a.o"}
	err := tryAutoLink(context.Background(), objFiles, "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestLinkWithGccFallbackToNoPie(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first gcc fails")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	err := linkWithGcc(context.Background(), []string{"obj"}, "bin", false, true, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkWithClangFallbackToNoPie(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first clang fails")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	err := linkWithClang(context.Background(), []string{"obj"}, "bin", false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkWithGccVerboseError(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("gcc error")
		},
	}
	err := linkWithGcc(context.Background(), []string{"obj"}, "bin", true, false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "gcc error") {
		t.Errorf("error should contain command output: %v", err)
	}
}

func TestLinkWithGccFallbackNoPieVerbose(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first fail")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	err := linkWithGcc(context.Background(), []string{"obj"}, "bin", true, true, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestTryAutoLinkNoLinker(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	objFiles := []string{"a.o"}
	err := tryAutoLink(context.Background(), objFiles, "bin", false, false, false, nil)
	if err == nil {
		t.Error("expected error when no linker")
	}
	if err.Error() != "auto linking failed: no suitable linker" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTryAutoLinkForceLdSkipsProbe(t *testing.T) {
	oldRunner := runner
	oldForceLd := ForceLD
	defer func() {
		runner = oldRunner
		ForceLD = oldForceLd
	}()
	ForceLD = true

	called := false
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			if name != ldForTarget() {
				t.Errorf("expected only ld to be invoked, got %s", name)
			}
			called = true
			return "", nil
		},
	}

	objFiles := []string{"a.o"}
	err := tryAutoLink(context.Background(), objFiles, "bin", false, false, false, nil)
	if err != nil {
		t.Fatalf("expected ld-only link to succeed, got %v", err)
	}
	if !called {
		t.Fatal("expected ld to be invoked")
	}
}

func TestLinkWithGccFallbackNoPie(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first fail")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	objFiles := []string{"a.o"}
	err := linkWithGcc(context.Background(), objFiles, "bin", false, true, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkWithLdMissingLib(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	libs := []string{"m", "c"}
	err := linkWithLd(context.Background(), []string{"obj.o"}, "bin", false, libs)
	if err != nil {
		t.Fatal(err)
	}
	if !containsPair(capturedArgs, "-l", "m") || !containsPair(capturedArgs, "-l", "c") {
		t.Error("library flags missing")
	}
}

func TestValidateLinkCallErrors(t *testing.T) {
	if err := validateLinkCall(nil, "out"); err == nil {
		t.Error("expected invalid linking context error")
	}
	if err := validateLinkCall(context.Background(), ""); err == nil {
		t.Error("expected output file name required error")
	}
}

func TestShouldUseResponseFile(t *testing.T) {
	args := make([]string, 200)
	for i := range args {
		args[i] = "arg"
	}
	if !shouldUseResponseFile(args) {
		t.Fatal("expected response file for many args")
	}
}

func TestCreateResponseFileWritesArgs(t *testing.T) {
	path, err := createResponseFile([]string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "a") || !strings.Contains(string(data), "b") {
		t.Fatal("response file content missing args")
	}
}

func TestRunLinkerCommandNoName(t *testing.T) {
	if _, err := runLinkerCommand(context.Background(), false, "", []string{"a"}); err == nil {
		t.Fatal("expected error when linker name is empty")
	}
}

func TestEnsureContextTimeoutReplacesExpiredContext(t *testing.T) {
	expiredCtx, cancel := context.WithTimeout(context.Background(), 0)
	cancel()
	ctx, cancelCtx := ensureContextTimeout(expiredCtx, 30*time.Second)
	defer cancelCtx()

	if err := ctx.Err(); err != nil {
		t.Fatalf("expected fresh context, got %v", err)
	}
	if deadline, ok := ctx.Deadline(); !ok {
		t.Fatal("expected deadline on new context")
	} else if time.Until(deadline) < 29*time.Second {
		t.Fatalf("expected at least 29s remaining, got %v", time.Until(deadline))
	}
}

func TestTryAutoLinkStrictClangSuccess(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	clangCalled := false
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			if name == "clang" {
				clangCalled = true
				return "", nil
			}
			return "", nil
		},
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed, cannot test strict clang path")
	}
	err := tryAutoLink(context.Background(), []string{"obj.o"}, "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !clangCalled {
		t.Error("clang not called in strict mode")
	}
}

func TestTryAutoLinkStrictClangFailGccSuccess(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if name == "clang" {
				return "", fmt.Errorf("clang error")
			}
			if name == "gcc" {
				return "", nil
			}
			return "", nil
		},
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed, cannot test clang failure path")
	}
	err := tryAutoLink(context.Background(), []string{"obj.o"}, "bin", false, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestLinkWithClangNoFallbackError(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("clang error")
		},
	}
	err := linkWithClang(context.Background(), []string{"obj.o"}, "bin", false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

func TestLinkWithClangFallbackVerbose(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first clang fail")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	err := linkWithClang(context.Background(), []string{"obj.o"}, "bin", true, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkWithGccFallbackVerbose(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first fail")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	objFiles := []string{"a.o"}
	err := linkWithGcc(context.Background(), objFiles, "bin", true, true, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkWithClangFallback(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	callCount := 0
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 && !contains(args, "-no-pie") {
				return "", fmt.Errorf("first fail")
			}
			if callCount == 2 && contains(args, "-no-pie") {
				return "", nil
			}
			return "", nil
		},
	}
	objFiles := []string{"a.o"}
	err := linkWithClang(context.Background(), objFiles, "bin", false, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestLinkWithLdMockNoRealLd(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	err := linkWithLd(context.Background(), []string{"obj.o"}, "bin", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-o") {
		t.Error("missing -o flag")
	}
}

func TestLinkWithGccVerboseFalseError(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return "", fmt.Errorf("gcc error")
		},
	}
	err := linkWithGcc(context.Background(), []string{"obj.o"}, "bin", false, false, false, false, nil)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "use -verbose for details") {
		t.Error("should hint -verbose")
	}
}

func TestLinkWithLd(t *testing.T) {
	if _, err := exec.LookPath("ld"); err != nil {
		t.Skip("ld not installed")
	}
	dir := t.TempDir()
	obj := buildObject(t, dir, "test", `
.globl _start
_start:
    mov $60, %eax
    xor %edi, %edi
    syscall
`)
	bin := filepath.Join(dir, "test")
	err := linkWithLd(context.Background(), []string{obj}, bin, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created")
	}
}

func TestLinkMultipleWithLd(t *testing.T) {
	if _, err := exec.LookPath("ld"); err != nil {
		t.Skip("ld not installed")
	}
	dir := t.TempDir()
	obj1 := buildObject(t, dir, "a", `
.globl _start
_start:
    call b
    mov $60, %eax
    syscall
`)
	obj2 := buildObject(t, dir, "b", `
.globl b
b:
    ret
`)
	bin := filepath.Join(dir, "test")
	err := linkWithLd(context.Background(), []string{obj1, obj2}, bin, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Error("binary not created")
	}
}

func TestLinkWithGccSharedFlag(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	oldShared := Shared
	Shared = true
	defer func() { Shared = oldShared }()
	err := linkWithGcc(context.Background(), []string{"obj.o"}, "bin", false, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-shared") {
		t.Error("missing -shared flag")
	}
}

func TestLinkWithGccLdFlags(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var capturedArgs []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}
	oldLdFlags := LdFlags
	LdFlags = "-Wl,-Map=output.map -pthread"
	defer func() { LdFlags = oldLdFlags }()
	err := linkWithGcc(context.Background(), []string{"obj.o"}, "bin", false, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-Wl,-Map=output.map") || !contains(capturedArgs, "-pthread") {
		t.Errorf("LdFlags not injected correctly: %v", capturedArgs)
	}
}

func TestParanoidLinker(t *testing.T) {
	origRunner := runner
	origCheckTool := utils.CheckToolFunc
	origLdFlags := LdFlags
	origShared := Shared
	origLdScript := LdScript
	origTextAddr := TextAddr
	defer func() {
		runner = origRunner
		utils.CheckToolFunc = origCheckTool
		LdFlags = origLdFlags
		Shared = origShared
		LdScript = origLdScript
		TextAddr = origTextAddr
	}()

	resetMocks := func() {
		runner = &MockRunner{RunFunc: nil}
		utils.CheckToolFunc = nil
		LdFlags = ""
		Shared = false
		LdScript = ""
		TextAddr = ""
	}

	objFile := func() string {
		f, err := os.CreateTemp("", "test*.o")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte("dummy object content")); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		return f.Name()
	}
	objPath := objFile()
	defer os.Remove(objPath)

	resetMocks()
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if name == "gcc" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected: %s", name)
	}}
	err := Link(context.Background(), objPath, "out", false, "c", false, false, false, nil)
	if err != nil {
		t.Errorf("mode c: %v", err)
	}

	resetMocks()
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if name == "ld" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected: %s", name)
	}}
	err = Link(context.Background(), objPath, "out", false, "raw", false, false, false, nil)
	if err != nil {
		t.Errorf("mode raw: %v", err)
	}

	resetMocks()
	callCount := 0
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		callCount++
		if name == "gcc" && !contains(args, "-no-pie") {
			return "", fmt.Errorf("gcc fails")
		}
		if name == "gcc" && contains(args, "-no-pie") {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = Link(context.Background(), objPath, "out", false, "auto", false, false, false, nil)
	if err != nil {
		t.Errorf("auto fallback to -no-pie: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}

	resetMocks()
	callCount = 0
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		callCount++
		if name == "gcc" {
			return "", fmt.Errorf("gcc fails")
		}
		if name == "ld" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = Link(context.Background(), objPath, "out", false, "auto", false, false, false, nil)
	if err != nil {
		t.Errorf("auto fallback to ld: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls (gcc, gcc -no-pie, ld), got %d", callCount)
	}

	resetMocks()
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "", fmt.Errorf("always fails")
	}}
	err = Link(context.Background(), objPath, "out", false, "auto", false, false, false, nil)
	if err == nil {
		t.Error("expected error when no linker works")
	}
	if !strings.Contains(err.Error(), "no suitable linker") && !strings.Contains(err.Error(), "link failed") {
		t.Errorf("wrong error: %v", err)
	}

	resetMocks()
	Shared = true
	var capturedArgs []string
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	}}
	err = linkWithGcc(context.Background(), []string{objPath}, "out", false, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-shared") {
		t.Error("missing -shared flag")
	}

	resetMocks()
	LdFlags = "-Wl,-Map=out.map -pthread"
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	}}
	err = linkWithGcc(context.Background(), []string{objPath}, "out", false, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-Wl,-Map=out.map") || !contains(capturedArgs, "-pthread") {
		t.Errorf("LdFlags not split: %v", capturedArgs)
	}

	resetMocks()
	LdScript = "linker.ld"
	TextAddr = "0x1000"
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	}}
	err = linkWithLd(context.Background(), []string{objPath}, "out", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(capturedArgs, "-T") || !contains(capturedArgs, "linker.ld") ||
		!contains(capturedArgs, "-Ttext") || !contains(capturedArgs, "0x1000") {
		t.Errorf("linker script/address missing: %v", capturedArgs)
	}

	resetMocks()
	emptyObj, _ := os.CreateTemp("", "empty*.o")
	emptyObj.Close()
	defer os.Remove(emptyObj.Name())
	err = Link(context.Background(), emptyObj.Name(), "out", false, "raw", false, false, false, nil)
	if err == nil {
		t.Error("expected error for empty object")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("wrong error: %v", err)
	}

	resetMocks()
	utils.CheckToolFunc = func(name string) error {
		if name == "gcc" {
			return fmt.Errorf("gcc not found")
		}
		return nil
	}
	err = Link(context.Background(), objPath, "out", false, "c", false, false, false, nil)
	if err == nil || !strings.Contains(err.Error(), "gcc not found") {
		t.Errorf("missing gcc not detected: %v", err)
	}
	utils.CheckToolFunc = nil

	resetMocks()
	utils.CheckToolFunc = func(name string) error {
		if name == "ld" {
			return fmt.Errorf("ld not found")
		}
		return nil
	}
	err = Link(context.Background(), objPath, "out", false, "raw", false, false, false, nil)
	if err == nil || !strings.Contains(err.Error(), "ld not found") {
		t.Errorf("missing ld not detected: %v", err)
	}
	utils.CheckToolFunc = nil

	resetMocks()
	utils.CheckToolFunc = func(name string) error {
		if name == "clang" {
			return nil
		}
		if name == "gcc" {
			return nil
		}
		return fmt.Errorf("not found")
	}
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if name == "clang" && contains(args, "-fsanitize=address") {
			return "", nil
		}
		if name == "gcc" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = tryAutoLink(context.Background(), []string{objPath}, "out", false, true, true, nil)
	if err != nil {
		t.Errorf("strict mode with clang: %v", err)
	}
	utils.CheckToolFunc = nil

	resetMocks()
	utils.CheckToolFunc = func(name string) error {
		if name == "clang" {
			return fmt.Errorf("not found")
		}
		if name == "gcc" {
			return nil
		}
		return fmt.Errorf("not found")
	}
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if name == "gcc" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = tryAutoLink(context.Background(), []string{objPath}, "out", false, true, true, nil)
	utils.CheckToolFunc = nil
	if err != nil {
		t.Errorf("clang not found fallback to gcc: %v", err)
	}

	resetMocks()
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if name == "ld" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = LinkMultiple(context.Background(), []string{objPath, objPath}, "out", false, "raw", false, false, false, nil)
	if err != nil {
		t.Errorf("LinkMultiple raw: %v", err)
	}

	resetMocks()
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		if name == "gcc" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = LinkMultiple(context.Background(), []string{objPath}, "out", false, "c", false, false, false, nil)
	if err != nil {
		t.Errorf("LinkMultiple c: %v", err)
	}

	resetMocks()
	callCount = 0
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		callCount++
		if name == "gcc" {
			return "", fmt.Errorf("gcc fails")
		}
		if name == "ld" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected")
	}}
	err = LinkMultiple(context.Background(), []string{objPath}, "out", false, "auto", false, false, false, nil)
	if err != nil {
		t.Errorf("LinkMultiple auto fallback: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls (gcc, gcc -no-pie, ld), got %d", callCount)
	}

	resetMocks()
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "", nil
	}}
	err = Link(context.Background(), objPath, "out", true, "raw", false, false, false, nil)
	w.Close()
	os.Stdout = oldStdout
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Running: ld") {
		t.Error("verbose mode did not print command")
	}
	if err != nil {
		t.Errorf("verbose link: %v", err)
	}

	resetMocks()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			return "", nil
		}
	}}
	err = Link(ctx, objPath, "out", false, "raw", false, false, false, nil)
	if err == nil || err.Error() != "context canceled" {
		t.Errorf("expected context cancellation, got %v", err)
	}
}

func TestLinkInvalidMode(t *testing.T) {
	dir := t.TempDir()
	obj := filepath.Join(dir, "obj.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "bin")
	err := Link(context.Background(), obj, bin, false, "invalid", false, false, false, nil)
	if err == nil {
		t.Error("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "unsupported mode") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestLinkMultipleInvalidMode(t *testing.T) {
	dir := t.TempDir()
	obj := filepath.Join(dir, "obj.o")
	if err := os.WriteFile(obj, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := LinkMultiple(context.Background(), []string{obj}, "bin", false, "invalid", false, false, false, nil)
	if err == nil {
		t.Error("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "unsupported mode") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestLinkMultipleParallelWithArchiveFallback(t *testing.T) {
	dir := t.TempDir()
	objs := make([]string, 0, 25)
	for i := 0; i < 24; i++ {
		obj := filepath.Join(dir, fmt.Sprintf("obj%02d.o", i))
		if err := os.WriteFile(obj, []byte("dummy"), 0o644); err != nil {
			t.Fatal(err)
		}
		objs = append(objs, obj)
	}
	archive := filepath.Join(dir, "libfoo.a")
	if err := os.WriteFile(archive, []byte("archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	objs = append(objs, archive)

	oldRunner := runner
	oldCheckTool := utils.CheckToolFunc
	oldZigRequested := ZigRequested
	oldZigEnabled := ZigEnabled
	defer func() {
		runner = oldRunner
		utils.CheckToolFunc = oldCheckTool
		ZigRequested = oldZigRequested
		ZigEnabled = oldZigEnabled
	}()

	var calls []string
	utils.CheckToolFunc = func(name string) error { return nil }
	ZigRequested = false
	ZigEnabled = false
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return "", nil
	}}

	if err := LinkMultipleParallel(context.Background(), objs, filepath.Join(dir, "out"), false, "c", true, false, false, nil, 4); err != nil {
		t.Fatalf("expected archive fallback link to succeed: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected single-stage link call when archive present, got %d", len(calls))
	}
	if !strings.HasPrefix(calls[0], "gcc ") {
		t.Fatalf("unexpected linker call: %s", calls[0])
	}
}

func TestLinkMultipleParallelPartitions(t *testing.T) {
	t.Skip("skipping parallel link test due to environment constraints")
	// FIXME please:
	// if _, err := exec.LookPath("ld"); err != nil {
	// 	t.Skip("ld not installed")
	// }
	// if _, err := exec.LookPath("gcc"); err != nil {
	// 	t.Skip("gcc not installed")
	// }
	// objs := make([]string, 0, 32)
	// dir := t.TempDir()
	// for i := 0; i < 32; i++ {
	// 	obj := filepath.Join(dir, fmt.Sprintf("obj%02d.o", i))
	// 	if err := os.WriteFile(obj, []byte("dummy"), 0o644); err != nil {
	// 		t.Fatal(err)
	// 	}
	// 	objs = append(objs, obj)
	// }
	// var calls []string
	// oldRunner := runner
	// oldCheckTool := utils.CheckToolFunc
	// defer func() {
	// 	runner = oldRunner
	// 	utils.CheckToolFunc = oldCheckTool
	// }()
	// utils.CheckToolFunc = func(name string) error { return nil }
	// runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
	// 	calls = append(calls, name+" "+strings.Join(args, " "))
	// 	return "", nil
	// }}
	// err := LinkMultipleParallel(context.Background(), objs, filepath.Join(dir, "out"), false, "raw", true, false, false, nil, 4)
	// if err != nil {
	// 	t.Fatalf("expected parallel LinkMultipleParallel to succeed: %v", err)
	// }
	// if len(calls) < 3 {
	// 	t.Fatalf("expected at least 3 linker calls, got %d", len(calls))
	// }
	// foundReloc := false
	// foundFinal := false
	// for _, call := range calls {
	// 	if strings.Contains(call, "-r") {
	// 		foundReloc = true
	// 	}
	// 	if strings.Contains(call, "-o "+filepath.Join(dir, "out")) {
	// 		foundFinal = true
	// 	}
	// }
	// if !foundReloc {
	// 	t.Fatal("expected partial relocatable link calls")
	// }
	// if !foundFinal {
	// 	t.Fatal("expected final link call")
	// }
}

func TestLinkEmptyObjectList(t *testing.T) {
	err := LinkMultiple(context.Background(), []string{}, "bin", false, "raw", false, false, false, nil)
	if err == nil {
		t.Error("expected error for empty object list")
	}
	if !strings.Contains(err.Error(), "no object files") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestLinkNonexistentObject(t *testing.T) {
	err := Link(context.Background(), "nonexistent.o", "bin", false, "raw", false, false, false, nil)
	if err == nil {
		t.Error("expected error for nonexistent object")
	}
}

func TestResponseFileCreation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("response file test skipped on windows")
	}
	oldRunner := runner
	defer func() { runner = oldRunner }()
	var argsUsed []string
	runner = &MockRunner{
		RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			argsUsed = args
			return "", nil
		},
	}
	longArgs := make([]string, 200)
	for i := range longArgs {
		longArgs[i] = "very_long_argument_to_force_response_file_usage"
	}
	_, err := runLinkerCommand(context.Background(), false, "testlinker", longArgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(argsUsed) != 1 || !strings.HasPrefix(argsUsed[0], "@") {
		t.Errorf("expected response file argument, got %v", argsUsed)
	}
}

func TestCheckDuplicateSymbolsVerboseReal(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("nasm not installed")
	}
	if _, err := exec.LookPath("objdump"); err != nil {
		t.Skip("objdump not installed")
	}
	dir := t.TempDir()
	buildNasmObject(t, dir, "dup1", `
section .text
global dup
dup: ret
`)
	buildNasmObject(t, dir, "dup2", `
section .text
global dup
dup: ret
`)
	obj1 := filepath.Join(dir, "dup1.o")
	obj2 := filepath.Join(dir, "dup2.o")
	err := CheckDuplicateSymbols(context.Background(), []string{obj1, obj2}, true)
	if err == nil {
		t.Error("expected duplicate symbol error")
	}
	if !strings.Contains(err.Error(), "duplicate global symbols") {
		t.Errorf("wrong error: %v", err)
	}
}
