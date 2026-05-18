package linker

import (
	"context"
	"fmt"
	"strings"

	"fz/internal/utils"
)

func linkWindows(ctx context.Context, obj, bin string, verbose bool, mode string, noSymbolCheck bool, sanitize bool, strict bool, libs []string) error {
	if noSymbolCheck {
	}
	switch mode {
	case "raw":
		if err := utils.CheckTool("clang"); err != nil {
			return err
		}
		return linkWithClangWindows(ctx, obj, bin, verbose, false, sanitize, libs)
	case "c", "auto":
		if err := utils.CheckTool("clang"); err != nil {
			return err
		}
		return linkWithClangWindows(ctx, obj, bin, verbose, true, sanitize, libs)
	default:
		return fmt.Errorf("unsupported mode for Windows: %s", mode)
	}
}

func linkMultipleWindows(ctx context.Context, objFiles []string, bin string, verbose bool, mode string, noSymbolCheck bool, sanitize bool, strict bool, libs []string) error {
	if noSymbolCheck {
		// skip for now
	}
	switch mode {
	case "raw":
		if err := utils.CheckTool("clang"); err != nil {
			return err
		}
		return linkMultipleWithClangWindows(ctx, objFiles, bin, verbose, false, sanitize, libs)
	case "c", "auto":
		if err := utils.CheckTool("clang"); err != nil {
			return err
		}
		return linkMultipleWithClangWindows(ctx, objFiles, bin, verbose, true, sanitize, libs)
	default:
		return fmt.Errorf("unsupported mode for Windows: %s", mode)
	}
}

func linkWithClangWindows(ctx context.Context, obj, bin string, verbose bool, allowFallback bool, sanitize bool, libs []string) error {
	args := []string{obj, "-o", bin, "-fuse-ld=lld"}
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if verbose {
		fmt.Printf("Running: clang %s\n", strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, "clang", args...)
	if err == nil {
		return nil
	}
	if !allowFallback {
		if !verbose {
			return fmt.Errorf("clang link failed (use -verbose for details)")
		}
		return fmt.Errorf("clang failed: %w\n%s", err, output)
	}
	argsWithNoPie := append([]string{"-no-pie"}, args...)
	if verbose {
		fmt.Printf("clang failed, retrying with -no-pie\n")
	}
	output2, err2 := runner.Run(ctx, verbose, "clang", argsWithNoPie...)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return fmt.Errorf("clang (with -no-pie) failed (use -verbose for details)")
	}
	return fmt.Errorf("clang -no-pie failed: %w\n%s", err2, output2)
}

func linkMultipleWithClangWindows(ctx context.Context, objFiles []string, bin string, verbose bool, allowFallback bool, sanitize bool, libs []string) error {
	args := append(objFiles, "-o", bin, "-fuse-ld=lld")
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if verbose {
		fmt.Printf("Running: clang %s\n", strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, "clang", args...)
	if err == nil {
		return nil
	}
	if !allowFallback {
		if !verbose {
			return fmt.Errorf("clang link failed (use -verbose for details)")
		}
		return fmt.Errorf("clang failed: %w\n%s", err, output)
	}
	argsWithNoPie := append([]string{"-no-pie"}, args...)
	if verbose {
		fmt.Printf("clang failed, retrying with -no-pie\n")
	}
	output2, err2 := runner.Run(ctx, verbose, "clang", argsWithNoPie...)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return fmt.Errorf("clang (with -no-pie) failed (use -verbose for details)")
	}
	return fmt.Errorf("clang -no-pie failed: %w\n%s", err2, output2)
}
