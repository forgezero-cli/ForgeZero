package zig

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"fz/internal/utils"
)

var (
	ZigRequested bool
	ZigEnabled   bool
	RunCommand   = func(ctx context.Context, verbose bool, args ...string) (string, error) {
		return utils.RunCommandSilent(ctx, verbose, "zig", args...)
	}
)

func IsAvailable() bool {
	return utils.CheckTool("zig") == nil
}

func shouldUseZig() bool {
	if ZigRequested {
		return true
	}
	return ZigEnabled || IsAvailable()
}

func CompilerForSource(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".m", ".mm":
		return "clang"
	case ".cpp", ".cc", ".cxx", ".c++":
		return "c++"
	default:
		return "cc"
	}
}

func CompileArgs(src, obj string, debug bool, target, ext, extraFlags string) []string {
	if target == "" {
		target = "x86_64-linux-gnu"
	}
	args := []string{CompilerForSource(ext), "-target", target, "-c", src, "-o", obj}
	args = append(args, "-fno-ident", "-fno-diagnostics-color", "-Wall", "-Wextra", "-Werror", "-Wpedantic", "-Wshadow", "-Wconversion")
	if debug {
		args = append(args, "-g")
		if dir := utils.GetExecutionRoot(); dir != "" {
			args = append(args, "-fdebug-prefix-map="+filepath.Clean(dir)+"=.")
		}
	}
	if extraFlags != "" {
		args = append(args, strings.Fields(extraFlags)...)
	}
	return args
}

func Compile(ctx context.Context, src, obj string, debug, verbose bool, target, extraFlags string) error {
	if !shouldUseZig() {
		return fmt.Errorf("zig not enabled")
	}
	if !IsAvailable() {
		return fmt.Errorf("zig not found in PATH")
	}
	args := CompileArgs(src, obj, debug, target, strings.ToLower(filepath.Ext(src)), extraFlags)
	if verbose {
		fmt.Printf("Running: zig %s\n", strings.Join(args, " "))
	}
	output, err := RunCommand(ctx, verbose, args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("zig compile failed (use -verbose for details)")
		}
		return fmt.Errorf("zig failed: %w\n%s", err, output)
	}
	return nil
}

func LinkArgs(objs []string, bin, target string, sanitize, strict bool, libs []string, shared bool, ldScript, textAddr string) []string {
	if target == "" {
		target = "x86_64-linux-gnu"
	}
	args := append([]string{"c++", "-target", target}, objs...)
	args = append(args, "-o", bin)
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
		if strict {
			args = append(args, "-fsanitize-address-use-after-return=always", "-fsanitize-address-use-after-scope")
		}
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	if ldScript != "" {
		args = append(args, "-Wl,-T,"+ldScript)
	}
	if textAddr != "" {
		args = append(args, "-Wl,-Ttext="+textAddr)
	}
	if !strings.Contains(target, "wasm") && !strings.Contains(target, "wasm32") {
		args = append(args, "-Wl,--build-id=none")
	}
	if shared {
		args = append(args, "-shared")
	}
	return args
}

func Link(ctx context.Context, objs []string, bin string, verbose bool, target string, sanitize bool, strict bool, libs []string, shared bool, ldScript string, textAddr string, extraFlags string) error {
	if !shouldUseZig() {
		return fmt.Errorf("zig not enabled")
	}
	if !IsAvailable() {
		return fmt.Errorf("zig not found in PATH")
	}
	args := LinkArgs(objs, bin, target, sanitize, strict, libs, shared, ldScript, textAddr)
	if extraFlags != "" {
		args = append(args, strings.Fields(extraFlags)...)
	}
	if verbose {
		fmt.Printf("Running: zig %s\n", strings.Join(args, " "))
	}
	output, err := RunCommand(ctx, verbose, args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("zig link failed (use -verbose for details)")
		}
		return fmt.Errorf("zig failed: %w\n%s", err, output)
	}
	return nil
}
