package linker

import (
	"context"
	"fmt"
	"strings"

	"fz/internal/utils"
)

func Link(ctx context.Context, obj, bin string, verbose bool, mode string) error {
	if err := utils.CheckFileExists(obj); err != nil {
		return err
	}
	if err := utils.EnsureDir(bin); err != nil {
		return err
	}

	switch mode {
	case "raw":
		if err := utils.CheckTool("ld"); err != nil {
			return err
		}
		return linkWithLd(ctx, obj, bin, verbose)
	case "c":
		if err := utils.CheckTool("gcc"); err != nil {
			return err
		}
		return linkWithGcc(ctx, obj, bin, verbose, false)
	case "auto":
		return tryAutoLink(ctx, obj, bin, verbose)
	default:
		return fmt.Errorf("unsupported mode: %s (valid: auto, c, raw)", mode)
	}
}

func tryAutoLink(ctx context.Context, obj, bin string, verbose bool) error {
	if err := utils.CheckTool("gcc"); err == nil {
		err = linkWithGcc(ctx, obj, bin, verbose, true)
		if err == nil {
			return nil
		}
	}
	if err := utils.CheckTool("ld"); err == nil {
		return linkWithLd(ctx, obj, bin, verbose)
	}
	return fmt.Errorf("auto linking failed: neither gcc nor ld available or all attempts failed")
}

func linkWithGcc(ctx context.Context, obj, bin string, verbose bool, allowNoPieFallback bool) error {
	if verbose {
		fmt.Printf("Running: gcc %s -o %s\n", obj, bin)
	}
	output, err := utils.RunCommandSilent(ctx, verbose, "gcc", obj, "-o", bin)
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
		fmt.Printf("Running: gcc %s -o %s -no-pie\n", obj, bin)
	}
	output2, err2 := utils.RunCommandSilent(ctx, verbose, "gcc", obj, "-o", bin, "-no-pie")
	if err2 == nil {
		return nil
	}
	if !verbose {
		return fmt.Errorf("gcc (with -no-pie) failed (use -verbose for details)")
	}
	return fmt.Errorf("gcc -no-pie failed: %w\n%s", err2, output2)
}

func linkWithLd(ctx context.Context, obj, bin string, verbose bool) error {
	if verbose {
		fmt.Printf("Running: ld %s -o %s\n", obj, bin)
	}
	output, err := utils.RunCommandSilent(ctx, verbose, "ld", obj, "-o", bin)
	if err != nil {
		if !verbose {
			return fmt.Errorf("ld link failed (use -verbose for details)")
		}
		return fmt.Errorf("ld failed: %w\n%s", err, output)
	}
	return nil
}

func LinkMultiple(ctx context.Context, objFiles []string, bin string, verbose bool, mode string) error {
	if len(objFiles) == 0 {
		return fmt.Errorf("no object files to link")
	}
	if err := utils.EnsureDir(bin); err != nil {
		return err
	}
	switch mode {
	case "raw":
		if err := utils.CheckTool("ld"); err != nil {
			return err
		}
		return linkMultipleWithLd(ctx, objFiles, bin, verbose)
	case "c":
		if err := utils.CheckTool("gcc"); err != nil {
			return err
		}
		return linkMultipleWithGcc(ctx, objFiles, bin, verbose, false)
	case "auto":
		return tryAutoLinkMultiple(ctx, objFiles, bin, verbose)
	default:
		return fmt.Errorf("unsupported mode: %s (valid: auto, c, raw)", mode)
	}
}

func tryAutoLinkMultiple(ctx context.Context, objFiles []string, bin string, verbose bool) error {
	if err := utils.CheckTool("gcc"); err == nil {
		err = linkMultipleWithGcc(ctx, objFiles, bin, verbose, true)
		if err == nil {
			return nil
		}
	}
	if err := utils.CheckTool("ld"); err == nil {
		return linkMultipleWithLd(ctx, objFiles, bin, verbose)
	}
	return fmt.Errorf("auto linking failed: neither gcc nor ld available or all attempts failed")
}

func linkMultipleWithGcc(ctx context.Context, objFiles []string, bin string, verbose bool, allowNoPieFallback bool) error {
	args := append(objFiles, "-o", bin)
	if verbose {
		fmt.Printf("Running: gcc %s\n", strings.Join(args, " "))
	}
	output, err := utils.RunCommandSilent(ctx, verbose, "gcc", args...)
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
		fmt.Printf("Running: gcc -no-pie %s\n", strings.Join(args, " "))
	}
	argsWithNoPie := append([]string{"-no-pie"}, args...)
	output2, err2 := utils.RunCommandSilent(ctx, verbose, "gcc", argsWithNoPie...)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return fmt.Errorf("gcc (with -no-pie) failed (use -verbose for details)")
	}
	return fmt.Errorf("gcc -no-pie failed: %w\n%s", err2, output2)
}

func linkMultipleWithLd(ctx context.Context, objFiles []string, bin string, verbose bool) error {
	args := append(objFiles, "-o", bin)
	if verbose {
		fmt.Printf("Running: ld %s\n", strings.Join(args, " "))
	}
	output, err := utils.RunCommandSilent(ctx, verbose, "ld", args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("ld link failed (use -verbose for details)")
		}
		return fmt.Errorf("ld failed: %w\n%s", err, output)
	}
	return nil
}
