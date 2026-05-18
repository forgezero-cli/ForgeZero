//go:build !windows

package linker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"fz/internal/utils"
)

func tryAutoLink(ctx context.Context, obj, bin string, verbose bool, sanitize bool, strict bool, libs []string) error {
	if strict {
		if _, err := exec.LookPath("clang"); err == nil {
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
	if err := utils.CheckTool("gcc"); err == nil {
		err = linkWithGcc(ctx, obj, bin, verbose, true, sanitize, strict, libs)
		if err == nil {
			return nil
		}
	}
	if err := utils.CheckTool("ld"); err == nil {
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
	if verbose {
		fmt.Printf("Running: clang %s\n", strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, "clang", args...)
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
	output2, err2 := runner.Run(ctx, verbose, "clang", argsWithNoPie...)
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
	if verbose {
		fmt.Printf("Running: gcc %s\n", strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, "gcc", args...)
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
		fmt.Printf("Running: gcc %s\n", strings.Join(argsWithNoPie, " "))
	}
	output2, err2 := runner.Run(ctx, verbose, "gcc", argsWithNoPie...)
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
	if verbose {
		fmt.Printf("Running: ld %s\n", strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, "ld", args...)
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
		if _, err := exec.LookPath("clang"); err == nil {
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
	if err := utils.CheckTool("gcc"); err == nil {
		err = linkMultipleWithGcc(ctx, objFiles, bin, verbose, true, sanitize, strict, libs)
		if err == nil {
			return nil
		}
	}
	if err := utils.CheckTool("ld"); err == nil {
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
	if verbose {
		fmt.Printf("Running: clang %s\n", strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, "clang", args...)
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
	output2, err2 := runner.Run(ctx, verbose, "clang", argsWithNoPie...)
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
	if verbose {
		fmt.Printf("Running: gcc %s\n", strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, "gcc", args...)
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
		fmt.Printf("Running: gcc %s\n", strings.Join(argsWithNoPie, " "))
	}
	output2, err2 := runner.Run(ctx, verbose, "gcc", argsWithNoPie...)
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
	if verbose {
		fmt.Printf("Running: ld %s\n", strings.Join(args, " "))
	}
	output, err := runner.Run(ctx, verbose, "ld", args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("ld link failed (use -verbose for details)")
		}
		return fmt.Errorf("ld failed: %w\n%s", err, output)
	}
	return nil
}
