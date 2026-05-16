package assembler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fz/internal/utils"
)

func Assemble(ctx context.Context, src, obj string, debug, verbose bool, mode string) error {
	if err := utils.CheckFileExists(src); err != nil {
		return err
	}
	if err := utils.EnsureDir(obj); err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(src))
	switch ext {
	case ".asm":
		if err := utils.CheckTool("nasm"); err != nil {
			return err
		}
		return assembleNASM(ctx, src, obj, debug, verbose)
	case ".s", ".S":
		if err := utils.CheckTool("gcc"); err != nil {
			return err
		}
		return assembleGAS(ctx, src, obj, debug, verbose)
	case ".fasm":
		if err := utils.CheckTool("fasm"); err != nil {
			return err
		}
		return assembleFASM(ctx, src, obj, verbose)
	default:
		return fmt.Errorf("unsupported source extension: %s (supported: .asm, .s, .S, .fasm)", ext)
	}
}

func assembleNASM(ctx context.Context, src, obj string, debug, verbose bool) error {
	args := []string{"-felf64", src, "-o", obj}
	if debug {
		args = append([]string{"-g"}, args...)
	}
	if verbose {
		fmt.Println("Running: nasm", strings.Join(args, " "))
	}
	cmd := exec.CommandContext(ctx, "nasm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func assembleGAS(ctx context.Context, src, obj string, debug, verbose bool) error {
	args := []string{"-c", src, "-o", obj}
	if debug {
		args = append([]string{"-g"}, args...)
	}
	if verbose {
		fmt.Println("Running: gcc", strings.Join(args, " "))
	}
	cmd := exec.CommandContext(ctx, "gcc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func assembleFASM(ctx context.Context, src, obj string, verbose bool) error {
	args := []string{src, obj}
	if verbose {
		fmt.Println("Running: fasm", strings.Join(args, " "))
	}
	cmd := exec.CommandContext(ctx, "fasm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
