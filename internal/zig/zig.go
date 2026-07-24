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

package zig

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
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
	switch strings.ToLower(ext) {
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
	compiler := CompilerForSource(ext)
	args := make([]string, 0, 16)
	args = append(args, compiler, "-target", target, "-c", src, "-o", obj)
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
		return errors.New("zig not enabled")
	}
	if !IsAvailable() {
		return errors.New("zig not found in PATH")
	}
	args := CompileArgs(src, obj, debug, target, strings.ToLower(filepath.Ext(src)), extraFlags)
	if verbose {
		_, _ = os.Stdout.WriteString("Running: zig ")
		_, _ = os.Stdout.WriteString(strings.Join(args, " "))
		_, _ = os.Stdout.WriteString("\n")
	}
	output, err := RunCommand(ctx, verbose, args...)
	if err != nil {
		if !verbose {
			return errors.New("zig compile failed (use -verbose for details)")
		}
		return errors.New("zig failed: " + err.Error() + "\n" + output)
	}
	return nil
}

func LinkArgs(objs []string, bin, target string, sanitize, strict bool, libs []string, shared bool, ldScript, textAddr string) []string {
	if target == "" {
		target = "x86_64-linux-gnu"
	}
	args := make([]string, 0, len(objs)+16)
	args = append(args, "c++", "-target", target)
	args = append(args, objs...)
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
		return errors.New("zig not enabled")
	}
	if !IsAvailable() {
		return errors.New("zig not found in PATH")
	}
	args := LinkArgs(objs, bin, target, sanitize, strict, libs, shared, ldScript, textAddr)
	if extraFlags != "" {
		args = append(args, strings.Fields(extraFlags)...)
	}
	if verbose {
		_, _ = os.Stdout.WriteString("Running: zig ")
		_, _ = os.Stdout.WriteString(strings.Join(args, " "))
		_, _ = os.Stdout.WriteString("\n")
	}
	output, err := RunCommand(ctx, verbose, args...)
	if err != nil {
		if !verbose {
			return errors.New("zig link failed (use -verbose for details)")
		}
		return errors.New("zig failed: " + err.Error() + "\n" + output)
	}
	return nil
}
