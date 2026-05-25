package assembler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fz/internal/seal"
	"fz/internal/utils"
)

var (
	OutputFormat   = "elf64"
	Target         = "x86_64-linux-gnu"
	AsmFlags       []string
	ForceFASM      bool
	CcFlags        string
	ZigRequested   bool
	ZigEnabled     bool
	runCommand     = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return utils.RunCommandSilent(ctx, verbose, name, args...)
	}
)

func SetRunCommand(fn func(ctx context.Context, verbose bool, name string, args ...string) (string, error)) {
	if fn == nil {
		runCommand = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return utils.RunCommandSilent(ctx, verbose, name, args...)
		}
		return
	}
	runCommand = fn
}

func WriteFlatAssembledNotice(path string) {
	fmt.Printf("Assembled flat binary: %s\n", path)
}

func validateArgs(args []string) error {
	for _, arg := range args {
		if err := utils.ValidateCLIArg(arg); err != nil {
			return err
		}
	}
	return nil
}

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

func CCForTarget() string {
	return ccForTarget()
}

func CXXForTarget() string {
	return cxxForTarget()
}

func GasCmdForTarget() string {
	return gasCmdForTarget()
}

func FormatFlagForTarget() string {
	return formatFlagForTarget()
}

func ccForTarget() string {
	switch {
	case isWasmTarget():
		if err := utils.CheckTool("emcc"); err == nil {
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
		if err := utils.CheckTool("em++"); err == nil {
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

func gasCmdForTarget() string {
	if isWasmTarget() {
		return "clang"
	}
	if strings.Contains(Target, "arm") {
		return "arm-linux-gnueabihf-as"
	}
	if strings.Contains(Target, "riscv") {
		return "riscv64-unknown-elf-as"
	}
	return "as"
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
	case ".asm", ".fasm":
		if mode != "raw" && mode != "auto" {
			return fmt.Errorf("unsupported source mode: %s (supported: raw, auto)", mode)
		}
		if isWasmTarget() {
			return fmt.Errorf("cannot assemble .asm files for wasm target")
		}
		return assembleRawASM(ctx, src, obj)
	case ".s":
		return assembleS(ctx, src, obj, verbose)
	case ".S":
		return compileC(ctx, src, obj, verbose, ccForTarget())
	case ".c":
		return compileC(ctx, src, obj, verbose, ccForTarget())
	case ".cpp", ".cc", ".cxx":
		return compileC(ctx, src, obj, verbose, cxxForTarget())
	default:
		return fmt.Errorf("unsupported source extension: %s (supported: .asm, .s, .S, .c, .cpp, .cc, .cxx)", ext)
	}
}

func assembleS(ctx context.Context, src, obj string, verbose bool) error {
	_, err := runCommand(ctx, verbose, gasCmdForTarget(), "-o", obj, src)
	return err
}

func compileC(ctx context.Context, src, obj string, verbose bool, compiler string) error {
	args := []string{"-c", src, "-o", obj}
	if CcFlags != "" {
		args = append(args, strings.Fields(CcFlags)...)
	}
	_, err := runCommand(ctx, verbose, compiler, args...)
	return err
}

func assembleRawASM(ctx context.Context, src, obj string) error {
	if IsBinFormat() {
		return assembleRawBinary(src, obj)
	}
	return assembleBareMetalObject(ctx, src, obj)
}

func assembleRawBinary(src, obj string) error {
	srcBuf, release, err := loadSourcePooled(src)
	if err != nil {
		return err
	}
	defer release()

	out, err := emitSourceRaw(srcBuf, selfTargetProfile(Target))
	if err != nil {
		return err
	}
	return os.WriteFile(obj, out, 0o644)
}

func isWasmTarget() bool {
	return strings.Contains(Target, "wasm") || strings.Contains(Target, "wasm32")
}
