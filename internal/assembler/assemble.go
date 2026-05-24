package assembler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fz/internal/seal"
	"fz/internal/utils"
	"fz/internal/zig"
)

var (
	OutputFormat = "elf64"
	Target       = "x86_64-linux-gnu"
	AsmFlags     []string
	ZigRequested bool
	ZigEnabled   bool
	CcFlags      string
)

var (
	runCommand = utils.RunCommandSilent
)

func SetRunCommand(fn func(ctx context.Context, verbose bool, name string, args ...string) (string, error)) {
	if fn == nil {
		runCommand = utils.RunCommandSilent
		return
	}
	runCommand = fn
}

func validateArgs(args []string) error {
	for _, arg := range args {
		if err := utils.ValidateCLIArg(arg); err != nil {
			return err
		}
	}
	return nil
}

func isWasmTarget() bool {
	return strings.Contains(Target, "wasm") || strings.Contains(Target, "wasm32")
}

var (
	ForceFASM bool
)

func asmCmdForTarget() string {
	switch {
	case isWasmTarget():
		return "clang"
	case strings.Contains(Target, "arm"):
		return "arm-linux-gnueabihf-as"
	case strings.Contains(Target, "riscv"):
		return "riscv64-unknown-elf-as"
	default:
		return "nasm"
	}
}

func ccForTarget() string {
	switch {
	case isWasmTarget():
		if _, err := exec.LookPath("emcc"); err == nil {
			return "emcc"
		}
		return "clang"
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
	case isWasmTarget():
		if _, err := exec.LookPath("em++"); err == nil {
			return "em++"
		}
		return "clang++"
	case strings.Contains(Target, "arm"):
		return "arm-linux-gnueabihf-g++"
	case strings.Contains(Target, "riscv"):
		return "riscv64-unknown-elf-g++"
	default:
		return "g++"
	}
}

func Assemble(ctx context.Context, src, obj string, debug, verbose bool, mode string) error {
	src = filepath.Clean(src)
	obj = filepath.Clean(obj)
	if seal.IsDecoyMode() {
		return emitDecoyObject(obj)
	}
	if err := utils.ValidateCLIPath(src); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	if err := utils.ValidateCLIPath(obj); err != nil {
		return fmt.Errorf("invalid object path: %w", err)
	}
	if err := utils.CheckFileExists(src); err != nil {
		return err
	}
	if err := utils.EnsureDir(obj); err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(src))
	switch ext {
	case ".asm":
		if ForceFASM {
			if err := CheckAssemblerTool("fasm"); err != nil {
				return err
			}
			return assembleFASM(ctx, src, obj, debug, verbose)
		}
		if mode == "raw" {
			return assembleRawASM(ctx, src, obj, debug, verbose)
		}
		if IsBinFormat() {
			if err := CheckAssemblerTool("nasm"); err != nil {
				return err
			}
			return assembleNASMHot(ctx, src, obj, debug, verbose)
		}
		if err := ensureAsmTool(asmCmdForTarget()); err != nil {
			return err
		}
		return assembleNASM(ctx, src, obj, debug, verbose)

	case ".s", ".S":
		if err := ensureAsmTool(asmCmdForTarget()); err != nil {
			return err
		}
		return assembleGAS(ctx, src, obj, debug, verbose)
	case ".fasm":
		if err := CheckAssemblerTool("fasm"); err != nil {
			return err
		}
		return assembleFASM(ctx, src, obj, debug, verbose)
	case ".c":
		if zig.ZigRequested || zig.ZigEnabled {
			return zig.Compile(ctx, src, obj, debug, verbose, Target, CcFlags)
		}
		if err := utils.CheckTool(ccForTarget()); err != nil {
			return err
		}
		return assembleC(ctx, src, obj, debug, verbose)
	case ".cpp", ".cc", ".cxx", ".c++":
		if zig.ZigRequested || zig.ZigEnabled {
			return zig.Compile(ctx, src, obj, debug, verbose, Target, CcFlags)
		}
		if err := utils.CheckTool(cxxForTarget()); err != nil {
			return err
		}
		return assembleCpp(ctx, src, obj, debug, verbose)
	default:
		return fmt.Errorf("unsupported source extension: %s (supported: .asm, .s, .S, .fasm, .c, .cpp, .cc, .cxx)", ext)
	}
}

func assembleRawASM(ctx context.Context, src, obj string, debug, verbose bool) error {
	if isWasmTarget() {
		return fmt.Errorf("cannot assemble .asm files for wasm target")
	}
	if IsBinFormat() {
		if verbose || debug {
			return assembleNASMSlow(ctx, src, obj, debug, verbose)
		}
		return assembleNASMHot(ctx, src, obj, debug, verbose)
	}
	return assembleNASMSlow(ctx, src, obj, debug, verbose)
}

func assembleNASM(ctx context.Context, src, obj string, debug, verbose bool) error {
	if isWasmTarget() {
		return fmt.Errorf("cannot assemble .asm files for wasm target")
	}
	if IsBinFormat() {
		return assembleNASMHot(ctx, src, obj, debug, verbose)
	}
	return assembleNASMSlow(ctx, src, obj, debug, verbose)
}

func assembleNASMSlow(ctx context.Context, src, obj string, debug, verbose bool) error {
	cmd, err := assembleNASMSlowCmd()
	if err != nil {
		return err
	}
	format := formatFlagForTarget()
	args := []string{format, src, "-o", obj}
	if debug && cmd == "nasm" {
		args = append([]string{"-g"}, args...)
	}
	if len(AsmFlags) > 0 {
		if err := validateArgs(AsmFlags); err != nil {
			return err
		}
		args = append(args, AsmFlags...)
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", cmd, strings.Join(args, " "))
	}
	output, err := runCommand(ctx, verbose, cmd, args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("%s failed (use -verbose for details)", cmd)
		}
		return fmt.Errorf("%s failed: %w\n%s", cmd, err, output)
	}
	return nil
}

func assembleGAS(ctx context.Context, src, obj string, debug, verbose bool) error {
	cmd := gasCmdForTarget()
	args := []string{"-c", src, "-o", obj}
	if isWasmTarget() {
		args = append([]string{"--target=wasm32-unknown-unknown"}, args...)
	}
	if debug {
		args = append([]string{"-g"}, args...)
	}
	if len(AsmFlags) > 0 {
		args = append(args, AsmFlags...)
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", cmd, strings.Join(args, " "))
	}
	output, err := runCommand(ctx, verbose, cmd, args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("%s failed (use -verbose for details)", cmd)
		}
		return fmt.Errorf("%s failed: %w\n%s", cmd, err, output)
	}
	return nil
}

func assembleFASM(ctx context.Context, src, obj string, debug, verbose bool) error {
	if isWasmTarget() {
		return fmt.Errorf("cannot assemble .fasm files for wasm target")
	}
	srcFile := src
	var tmpFile *os.File
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("cannot read source: %w", err)
	}
	inject := "format ELF64\n"
	if IsBinFormat() {
		inject = "format binary\n"
	}
	if !bytes.Contains(data, []byte("format ELF64")) && !bytes.Contains(data, []byte("format binary")) {
		tmpFile, err = os.CreateTemp(filepath.Dir(src), "fz_fasm_*.asm")
		if err != nil {
			return fmt.Errorf("cannot create temp file: %w", err)
		}
		_ = tmpFile.Chmod(utils.FilePerm)
		tmpName := tmpFile.Name()
		defer func() {
			if tmpFile != nil {
				tmpFile.Close()
			}
			os.Remove(tmpName)
		}()
		if _, err := tmpFile.WriteString(inject); err != nil {
			return err
		}
		if _, err := tmpFile.Write(data); err != nil {
			return err
		}
		if err := tmpFile.Close(); err != nil {
			return err
		}
		tmpFile = nil
		srcFile = tmpName
		if verbose {
			fmt.Println("FASM: injected 'format ELF64' directive (object file mode)")
		}
	} else if verbose {
		fmt.Println("FASM: source already contains 'format ELF64'")
	}

	args := []string{srcFile, obj}
	if debug {
		args = append([]string{"-dDEBUG=1"}, args...)
	}
	if len(AsmFlags) > 0 {
		args = append(args, AsmFlags...)
	}
	if verbose {
		fmt.Println("Running: fasm", strings.Join(args, " "))
		if debug {
			fmt.Fprintln(os.Stderr, "note: FASM debug flag")
		}
	}
	output, err := runCommand(ctx, verbose, "fasm", args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("fasm failed (use -verbose for details)")
		}
		lines := strings.Split(output, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			lower := strings.ToLower(line)
			if strings.Contains(lower, "error") || strings.Contains(lower, "fatal") || strings.Contains(lower, "line") {
				return fmt.Errorf("fasm error: %s", line)
			}
		}
		return fmt.Errorf("fasm failed: %w\n%s", err, output)
	}
	return nil
}

func assembleC(ctx context.Context, src, obj string, debug, verbose bool) error {
	compiler := ccForTarget()
	args := []string{"-c", src, "-o", obj}
	if isWasmTarget() && compiler == "clang" {
		args = append([]string{"--target=wasm32-unknown-unknown"}, args...)
	}
	strictFlags := []string{"-Wall", "-Wextra", "-Werror", "-Wpedantic", "-Wshadow", "-Wconversion"}
	args = append(args, strictFlags...)
	if debug {
		args = append(args, "-g")
		if dir := utils.GetExecutionRoot(); dir != "" {
			args = append(args, "-fdebug-prefix-map="+filepath.Clean(dir)+"=.")
		}
	}
	if len(CcFlags) > 0 {
		extraFlags := strings.Fields(CcFlags)
		if err := validateArgs(extraFlags); err != nil {
			return err
		}
		args = append(args, extraFlags...)
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", compiler, strings.Join(args, " "))
	}
	output, err := runCommand(ctx, verbose, compiler, args...)
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
	if isWasmTarget() && compiler == "clang++" {
		args = append([]string{"--target=wasm32-unknown-unknown"}, args...)
	}
	strictFlags := []string{"-Wall", "-Wextra", "-Werror", "-Wpedantic", "-Wshadow", "-Wconversion"}
	args = append(args, strictFlags...)
	if debug {
		args = append(args, "-g")
		if dir := utils.GetExecutionRoot(); dir != "" {
			args = append(args, "-fdebug-prefix-map="+filepath.Clean(dir)+"=.")
		}
	}
	if len(CcFlags) > 0 {
		extraFlags := strings.Fields(CcFlags)
		if err := validateArgs(extraFlags); err != nil {
			return err
		}
		args = append(args, extraFlags...)
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", compiler, strings.Join(args, " "))
	}
	output, err := runCommand(ctx, verbose, compiler, args...)
	if err != nil {
		if !verbose {
			return fmt.Errorf("%s compilation failed (use -verbose for details)", compiler)
		}
		return fmt.Errorf("%s failed: %w\n%s", compiler, err, output)
	}
	return nil
}

func CCForTarget() string {
	return ccForTarget()
}

func gasCmdForTarget() string {
	switch {
	case isWasmTarget():
		return "clang"
	case strings.Contains(Target, "arm"):
		return "arm-linux-gnueabihf-as"
	case strings.Contains(Target, "riscv"):
		return "riscv64-unknown-elf-as"
	default:
		return "as"
	}
}
