package linker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"fz/internal/utils"
)

var (
	runner   CmdRunner = &RealCmdRunner{}
	LdScript string
	TextAddr string
	Target   = "x86_64-linux-gnu"
	LdFlags  string
	Shared   bool
)

func ldForTarget() string {
	switch {
	case strings.Contains(Target, "arm"):
		return "arm-linux-gnueabihf-ld"
	case strings.Contains(Target, "riscv"):
		return "riscv64-unknown-elf-ld"
	default:
		return "ld"
	}
}

func gccForTarget() string {
	switch {
	case strings.Contains(Target, "arm"):
		return "arm-linux-gnueabihf-gcc"
	case strings.Contains(Target, "riscv"):
		return "riscv64-unknown-elf-gcc"
	default:
		return "gcc"
	}
}

func clangForTarget() string {
	return "clang"
}

func Link(ctx context.Context, obj, bin string, verbose bool, mode string, noSymbolCheck bool, sanitize bool, strict bool, libs []string) error {
	if err := utils.CheckFileExists(obj); err != nil {
		return err
	}
	info, err := os.Stat(obj)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fmt.Errorf("object file %s is empty", obj)
	}
	if err := utils.EnsureDir(bin); err != nil {
		return err
	}
	if !noSymbolCheck {
		if err := CheckDuplicateSymbols([]string{obj}, verbose); err != nil {
			return err
		}
	}

	if runtime.GOOS == "windows" {
		return linkWindowsImpl(ctx, obj, bin, verbose, mode, sanitize, libs)
	}

	switch mode {
	case "raw":
		if err := utils.CheckTool(ldForTarget()); err != nil {
			return err
		}
		return linkWithLd(ctx, obj, bin, verbose, libs)
	case "c":
		if err := utils.CheckTool(gccForTarget()); err != nil {
			return err
		}
		return linkWithGcc(ctx, obj, bin, verbose, false, sanitize, strict, libs)
	case "auto":
		return tryAutoLink(ctx, obj, bin, verbose, sanitize, strict, libs)
	default:
		return fmt.Errorf("unsupported mode: %s (valid: auto, c, raw)", mode)
	}
}

func LinkMultiple(ctx context.Context, objFiles []string, bin string, verbose bool, mode string, noSymbolCheck bool, sanitize bool, strict bool, libs []string) error {
	if len(objFiles) == 0 {
		return fmt.Errorf("no object files to link")
	}
	for _, obj := range objFiles {
		info, err := os.Stat(obj)
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			return fmt.Errorf("object file %s is empty", obj)
		}
	}
	if err := utils.EnsureDir(bin); err != nil {
		return err
	}
	if !noSymbolCheck {
		if err := CheckDuplicateSymbols(objFiles, verbose); err != nil {
			return err
		}
	}

	if runtime.GOOS == "windows" {
		return linkMultipleWindowsImpl(ctx, objFiles, bin, verbose, mode, sanitize, libs)
	}

	switch mode {
	case "raw":
		if err := utils.CheckTool(ldForTarget()); err != nil {
			return err
		}
		return linkMultipleWithLd(ctx, objFiles, bin, verbose, libs)
	case "c":
		if err := utils.CheckTool(gccForTarget()); err != nil {
			return err
		}
		return linkMultipleWithGcc(ctx, objFiles, bin, verbose, false, sanitize, strict, libs)
	case "auto":
		return tryAutoLinkMultiple(ctx, objFiles, bin, verbose, sanitize, strict, libs)
	default:
		return fmt.Errorf("unsupported mode: %s (valid: auto, c, raw)", mode)
	}
}

func tryAutoLink(ctx context.Context, obj, bin string, verbose bool, sanitize bool, strict bool, libs []string) error {
	if strict {
		if _, err := exec.LookPath(clangForTarget()); err == nil {
			if verbose {
				fmt.Println("Strict mode: using clang for better sanitizers")
			}
			err = linkWithClang(ctx, obj, bin, verbose, true, sanitize, libs)
			if err == nil {
				return nil
			}
		} else if verbose {
			fmt.Println("clang not found, falling back to gcc (limited strict mode)")
		}
	}
	if err := utils.CheckTool(gccForTarget()); err == nil {
		err = linkWithGcc(ctx, obj, bin, verbose, true, sanitize, strict, libs)
		if err == nil {
			return nil
		}
	}
	if err := utils.CheckTool(ldForTarget()); err == nil {
		return linkWithLd(ctx, obj, bin, verbose, libs)
	}
	return fmt.Errorf("auto linking failed: no suitable linker")
}

func linkWithClang(ctx context.Context, obj, bin string, verbose bool, allowNoPieFallback bool, sanitize bool, libs []string) error {
	args := []string{obj, "-o", bin}
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
		args = append(args, "-fsanitize-address-use-after-return=always")
		args = append(args, "-fsanitize-address-use-after-scope")
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", clangForTarget(), strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, clangForTarget(), args...)
	if err == nil {
		return nil
	}
	if !allowNoPieFallback {
		if !verbose {
			return fmt.Errorf("clang link failed (use -verbose for details)")
		}
		return fmt.Errorf("clang failed: %w\n%s", err, output)
	}
	argsWithNoPie := append([]string{"-no-pie"}, args...)
	if verbose {
		fmt.Printf("clang failed, retrying with -no-pie\n")
	}
	output2, err2 := runner.Run(ctx, verbose, clangForTarget(), argsWithNoPie...)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return fmt.Errorf("clang (with -no-pie) failed (use -verbose for details)")
	}
	return fmt.Errorf("clang -no-pie failed: %w\n%s", err2, output2)
}

func linkWithGcc(ctx context.Context, obj, bin string, verbose bool, allowNoPieFallback bool, sanitize bool, strict bool, libs []string) error {
	args := []string{obj, "-o", bin}
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
		if strict {
			args = append(args, "-fsanitize-address-use-after-scope")
		}
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", gccForTarget(), strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, gccForTarget(), args...)
	if err == nil {
		return nil
	}
	if !allowNoPieFallback {
		if !verbose {
			return fmt.Errorf("gcc link failed (use -verbose for details)")
		}
		return fmt.Errorf("gcc failed: %w\n%s", err, output)
	}
	if verbose {
		fmt.Printf("gcc failed, retrying with -no-pie\n")
	}
	argsWithNoPie := append([]string{"-no-pie"}, args...)
	if verbose {
		fmt.Printf("Running: %s %s\n", gccForTarget(), strings.Join(argsWithNoPie, " "))
	}
	output2, err2 := runner.Run(ctx, verbose, gccForTarget(), argsWithNoPie...)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return fmt.Errorf("gcc (with -no-pie) failed (use -verbose for details)")
	}
	return fmt.Errorf("gcc -no-pie failed: %w\n%s", err2, output2)
}

func linkWithLd(ctx context.Context, obj, bin string, verbose bool, libs []string) error {
	args := []string{obj, "-o", bin}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", ldForTarget(), strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, ldForTarget(), args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("ld link failed (use -verbose for details)")
		}
		return fmt.Errorf("ld failed: %w\n%s", err, output)
	}
	return nil
}

func tryAutoLinkMultiple(ctx context.Context, objFiles []string, bin string, verbose bool, sanitize bool, strict bool, libs []string) error {
	if strict {
		if _, err := exec.LookPath(clangForTarget()); err == nil {
			if verbose {
				fmt.Println("Strict mode: using clang for better sanitizers")
			}
			err = linkMultipleWithClang(ctx, objFiles, bin, verbose, true, sanitize, libs)
			if err == nil {
				return nil
			}
		} else if verbose {
			fmt.Println("clang not found, falling back to gcc (limited strict mode)")
		}
	}
	if err := utils.CheckTool(gccForTarget()); err == nil {
		err = linkMultipleWithGcc(ctx, objFiles, bin, verbose, true, sanitize, strict, libs)
		if err == nil {
			return nil
		}
	}
	if err := utils.CheckTool(ldForTarget()); err == nil {
		return linkMultipleWithLd(ctx, objFiles, bin, verbose, libs)
	}
	return fmt.Errorf("auto linking failed: no suitable linker")
}

func linkMultipleWithClang(ctx context.Context, objFiles []string, bin string, verbose bool, allowNoPieFallback bool, sanitize bool, libs []string) error {
	args := append(objFiles, "-o", bin)
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
		args = append(args, "-fsanitize-address-use-after-return=always")
		args = append(args, "-fsanitize-address-use-after-scope")
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", clangForTarget(), strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, clangForTarget(), args...)
	if err == nil {
		return nil
	}
	if !allowNoPieFallback {
		if !verbose {
			return fmt.Errorf("clang link failed (use -verbose for details)")
		}
		return fmt.Errorf("clang failed: %w\n%s", err, output)
	}
	argsWithNoPie := append([]string{"-no-pie"}, args...)
	if verbose {
		fmt.Printf("clang failed, retrying with -no-pie\n")
	}
	output2, err2 := runner.Run(ctx, verbose, clangForTarget(), argsWithNoPie...)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return fmt.Errorf("clang (with -no-pie) failed (use -verbose for details)")
	}
	return fmt.Errorf("clang -no-pie failed: %w\n%s", err2, output2)
}

func linkMultipleWithGcc(ctx context.Context, objFiles []string, bin string, verbose bool, allowNoPieFallback bool, sanitize bool, strict bool, libs []string) error {
	args := append(objFiles, "-o", bin)
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
		if strict {
			args = append(args, "-fsanitize-address-use-after-scope")
		}
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", gccForTarget(), strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, gccForTarget(), args...)
	if err == nil {
		return nil
	}
	if !allowNoPieFallback {
		if !verbose {
			return fmt.Errorf("gcc link failed (use -verbose for details)")
		}
		return fmt.Errorf("gcc failed: %w\n%s", err, output)
	}
	if verbose {
		fmt.Printf("gcc failed, retrying with -no-pie\n")
	}
	argsWithNoPie := append([]string{"-no-pie"}, args...)
	if verbose {
		fmt.Printf("Running: %s %s\n", gccForTarget(), strings.Join(argsWithNoPie, " "))
	}
	output2, err2 := runner.Run(ctx, verbose, gccForTarget(), argsWithNoPie...)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return fmt.Errorf("gcc (with -no-pie) failed (use -verbose for details)")
	}
	return fmt.Errorf("gcc -no-pie failed: %w\n%s", err2, output2)
}

func linkMultipleWithLd(ctx context.Context, objFiles []string, bin string, verbose bool, libs []string) error {
	args := append(objFiles, "-o", bin)
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", ldForTarget(), strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, ldForTarget(), args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("ld link failed (use -verbose for details)")
		}
		return fmt.Errorf("ld failed: %w\n%s", err, output)
	}
	return nil
}

func linkWindowsImpl(ctx context.Context, obj, bin string, verbose bool, mode string, sanitize bool, libs []string) error {
	if err := utils.CheckTool("clang"); err != nil {
		return err
	}
	args := []string{obj, "-o", bin, "-fuse-ld=lld"}
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: clang %s\n", strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, "clang", args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("clang link failed (use -verbose for details)")
		}
		return fmt.Errorf("clang failed: %w\n%s", err, output)
	}
	return nil
}

func linkMultipleWindowsImpl(ctx context.Context, objFiles []string, bin string, verbose bool, mode string, sanitize bool, libs []string) error {
	if err := utils.CheckTool("clang"); err != nil {
		return err
	}
	args := append(objFiles, "-o", bin, "-fuse-ld=lld")
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: clang %s\n", strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, "clang", args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("clang link failed (use -verbose for details)")
		}
		return fmt.Errorf("clang failed: %w\n%s", err, output)
	}
	return nil
}
