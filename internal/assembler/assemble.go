package assembler

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"fz/internal/utils"
)

var OutputFormat = "elf64"

func formatToNasmFlag(format string) string {
	switch format {
	case "elf32":
		return "-felf32"
	case "elf64":
		return "-felf64"
	case "bin":
		return "-fbin"
	default:
		return "-felf64"
	}
}

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
	case ".c":
		if err := utils.CheckTool("gcc"); err != nil {
			return err
		}
		return assembleC(ctx, src, obj, debug, verbose)
	case ".cpp", ".cc", ".cxx", ".c++":
		if err := utils.CheckTool("g++"); err != nil {
			return err
		}
		return assembleCpp(ctx, src, obj, debug, verbose)
	default:
		return fmt.Errorf("unsupported source extension: %s (supported: .asm, .s, .S, .fasm, .c, .cpp, .cc, .cxx)", ext)
	}
}

func assembleNASM(ctx context.Context, src, obj string, debug, verbose bool) error {
	formatFlag := formatToNasmFlag(OutputFormat)
	args := []string{formatFlag, src, "-o", obj}
	if debug {
		args = append([]string{"-g"}, args...)
	}
	if verbose {
		fmt.Println("Running: nasm", strings.Join(args, " "))
	}
	output, err := utils.RunCommandSilent(ctx, verbose, "nasm", args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("nasm failed (use -verbose for details)")
		}
		return fmt.Errorf("nasm failed: %w\n%s", err, output)
	}
	return nil
}

func assembleGAS(ctx context.Context, src, obj string, debug, verbose bool) error {
	args := []string{"-c", src, "-o", obj}
	if debug {
		args = append([]string{"-g"}, args...)
	}
	if verbose {
		fmt.Println("Running: gcc", strings.Join(args, " "))
	}
	output, err := utils.RunCommandSilent(ctx, verbose, "gcc", args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("gcc assembly failed (use -verbose for details)")
		}
		return fmt.Errorf("gcc failed: %w\n%s", err, output)
	}
	return nil
}

func assembleFASM(ctx context.Context, src, obj string, verbose bool) error {
	args := []string{src, obj}
	if verbose {
		fmt.Println("Running: fasm", strings.Join(args, " "))
	}
	output, err := utils.RunCommandSilent(ctx, verbose, "fasm", args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("fasm failed (use -verbose for details)")
		}
		return fmt.Errorf("fasm failed: %w\n%s", err, output)
	}
	return nil
}

func assembleC(ctx context.Context, src, obj string, debug, verbose bool) error {
	args := []string{"-c", src, "-o", obj}
	strictFlags := []string{"-Wall", "-Wextra", "-Werror", "-Wpedantic", "-Wshadow", "-Wconversion"}
	args = append(args, strictFlags...)
	if debug {
		args = append(args, "-g")
	}
	if verbose {
		fmt.Println("Running: gcc", strings.Join(args, " "))
	}
	output, err := utils.RunCommandSilent(ctx, verbose, "gcc", args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("gcc compilation failed (use -verbose for details)")
		}
		return fmt.Errorf("gcc failed: %w\n%s", err, output)
	}
	return nil
}

func assembleCpp(ctx context.Context, src, obj string, debug, verbose bool) error {
	args := []string{"-c", src, "-o", obj}
	strictFlags := []string{"-Wall", "-Wextra", "-Werror", "-Wpedantic", "-Wshadow", "-Wconversion"}
	args = append(args, strictFlags...)
	if debug {
		args = append(args, "-g")
	}
	if verbose {
		fmt.Println("Running: g++", strings.Join(args, " "))
	}
	output, err := utils.RunCommandSilent(ctx, verbose, "g++", args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("g++ compilation failed (use -verbose for details)")
		}
		return fmt.Errorf("g++ failed: %w\n%s", err, output)
	}
	return nil
}
