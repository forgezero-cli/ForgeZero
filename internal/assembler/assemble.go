package assembler

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"fz/internal/utils"
)

var (
	OutputFormat = "elf64"
	Target       = "x86_64-linux-gnu"
)

func formatForTarget() string {
	switch {
	case strings.Contains(Target, "x86_64"):
		return "-felf64"
	case strings.Contains(Target, "i386") || strings.Contains(Target, "i686"):
		return "-felf32"
	case strings.Contains(Target, "arm"):
		return "-march=armv7-a"
	default:
		return "-felf64"
	}
}

func gasCmdForTarget() string {
	switch {
	case strings.Contains(Target, "arm"):
		return "arm-linux-gnueabihf-as"
	case strings.Contains(Target, "riscv"):
		return "riscv64-unknown-elf-as"
	default:
		return "as"
	}
}

func ccForTarget() string {
	switch {
	case strings.Contains(Target, "arm"):
		return "arm-linux-gnueabihf-gcc"
	case strings.Contains(Target, "riscv"):
		return "riscv64-unknown-elf-gcc"
	default:
		return "gcc"
	}
}

func cxxForTarget() string {
	switch {
	case strings.Contains(Target, "arm"):
		return "arm-linux-gnueabihf-g++"
	case strings.Contains(Target, "riscv"):
		return "riscv64-unknown-elf-g++"
	default:
		return "g++"
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
		if err := utils.CheckTool(gasCmdForTarget()); err != nil {
			return err
		}
		return assembleGAS(ctx, src, obj, debug, verbose)
	case ".fasm":
		if err := utils.CheckTool("fasm"); err != nil {
			return err
		}
		return assembleFASM(ctx, src, obj, verbose)
	case ".c":
		if err := utils.CheckTool(ccForTarget()); err != nil {
			return err
		}
		return assembleC(ctx, src, obj, debug, verbose)
	case ".cpp", ".cc", ".cxx", ".c++":
		if err := utils.CheckTool(cxxForTarget()); err != nil {
			return err
		}
		return assembleCpp(ctx, src, obj, debug, verbose)
	default:
		return fmt.Errorf("unsupported source extension: %s (supported: .asm, .s, .S, .fasm, .c, .cpp, .cc, .cxx)", ext)
	}
}

func assembleNASM(ctx context.Context, src, obj string, debug, verbose bool) error {
	format := formatForTarget()
	args := []string{format, src, "-o", obj}
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
	cmd := gasCmdForTarget()
	args := []string{"-c", src, "-o", obj}
	if debug {
		args = append([]string{"-g"}, args...)
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", cmd, strings.Join(args, " "))
	}
	output, err := utils.RunCommandSilent(ctx, verbose, cmd, args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("%s failed (use -verbose for details)", cmd)
		}
		return fmt.Errorf("%s failed: %w\n%s", cmd, err, output)
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
	compiler := ccForTarget()
	args := []string{"-c", src, "-o", obj}
	strictFlags := []string{"-Wall", "-Wextra", "-Werror", "-Wpedantic", "-Wshadow", "-Wconversion"}
	args = append(args, strictFlags...)
	if debug {
		args = append(args, "-g")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", compiler, strings.Join(args, " "))
	}
	output, err := utils.RunCommandSilent(ctx, verbose, compiler, args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("%s compilation failed (use -verbose for details)", compiler)
		}
		return fmt.Errorf("%s failed: %w\n%s", compiler, err, output)
	}
	return nil
}

func assembleCpp(ctx context.Context, src, obj string, debug, verbose bool) error {
	compiler := cxxForTarget()
	args := []string{"-c", src, "-o", obj}
	strictFlags := []string{"-Wall", "-Wextra", "-Werror", "-Wpedantic", "-Wshadow", "-Wconversion"}
	args = append(args, strictFlags...)
	if debug {
		args = append(args, "-g")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", compiler, strings.Join(args, " "))
	}
	output, err := utils.RunCommandSilent(ctx, verbose, compiler, args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("%s compilation failed (use -verbose for details)", compiler)
		}
		return fmt.Errorf("%s failed: %w\n%s", compiler, err, output)
	}
	return nil
}
