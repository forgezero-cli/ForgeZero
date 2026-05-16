package linker

import (
	"context"
	"fmt"
	"os"
	"os/exec"

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
		return linkWithGcc(ctx, obj, bin, verbose)
	case "auto":
		if err := tryAutoLink(ctx, obj, bin, verbose); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported mode: %s (valid: auto, c, raw)", mode)
	}
}

func tryAutoLink(ctx context.Context, obj, bin string, verbose bool) error {
	if err := utils.CheckTool("gcc"); err == nil {
		err = linkWithGcc(ctx, obj, bin, verbose)
		if err == nil {
			return nil
		}
		if verbose {
			fmt.Println("gcc link failed, retrying with -no-pie")
		}
		err2 := linkWithGccArgs(ctx, obj, bin, verbose, "-no-pie")
		if err2 == nil {
			return nil
		}
		if verbose {
			fmt.Println("gcc -no-pie also failed, falling back to ld")
		}
	}
	if err := utils.CheckTool("ld"); err == nil {
		return linkWithLd(ctx, obj, bin, verbose)
	}
	return fmt.Errorf("auto linking failed: neither gcc nor ld available or all attempts failed")
}

func linkWithGcc(ctx context.Context, obj, bin string, verbose bool) error {
	return linkWithGccArgs(ctx, obj, bin, verbose)
}

func linkWithGccArgs(ctx context.Context, obj, bin string, verbose bool, extraArgs ...string) error {
	args := append([]string{obj, "-o", bin}, extraArgs...)
	if verbose {
		fmt.Println("Running: gcc", args)
	}
	cmd := exec.CommandContext(ctx, "gcc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func linkWithLd(ctx context.Context, obj, bin string, verbose bool) error {
	args := []string{obj, "-o", bin}
	if verbose {
		fmt.Println("Running: ld", args)
	}
	cmd := exec.CommandContext(ctx, "ld", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
