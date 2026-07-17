/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package assembler

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/forgezero-cli/ForgeZero/internal/seal"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

var (
	OutputFormat          = "elf64"
	Target                = "x86_64-linux-gnu"
	AsmFlags              []string
	ForceFASM             bool
	CcFlags               string
	ZigRequested          bool
	ZigEnabled            bool
	CcFLagsParsed         []string
	AdditionalIncludeDirs []string
	ForceInternalAsm      bool = true
	UseNasm               bool
	CcFlagsOnce           sync.Once
	runCommand            = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return utils.RunCommandSilent(ctx, verbose, name, args...)
	}
	initFlagsOnce sync.Once
)

func initAssemblerFlags() {
	initFlagsOnce.Do(func() {
		if muslFlag := flag.Lookup("musl"); muslFlag != nil && muslFlag.Value.String() != "" {
			muslVal := muslFlag.Value.String()
			switch {
			case muslVal == "riscv64":
				Target = "riscv64-linux-musl"
			case muslVal != "true" && muslVal != "false":
				Target = muslVal + "-linux-musl"
			case muslVal == "true":
				Target = "x86_64-linux-musl"
			}
		}
		if useNasmFlag := flag.Lookup("use-nasm"); useNasmFlag != nil {
			UseNasm = useNasmFlag.Value.String() == "true"
		}
		ForceInternalAsm = !UseNasm
	})
}

func SetRunCommand(fn func(ctx context.Context, verbose bool, name string, args ...string) (string, error)) {
	if fn == nil {
		runCommand = func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
			return utils.RunCommandSilent(ctx, verbose, name, args...)
		}
		return
	}
	runCommand = fn
}

func SetAdditionalIncludeDirs(dirs []string) {
	if len(dirs) == 0 {
		AdditionalIncludeDirs = nil
		return
	}
	seen := make(map[string]struct{}, len(dirs))
	out := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}
		resolved := trimmed
		if abs, err := filepath.Abs(trimmed); err == nil {
			resolved = abs
		}
		if _, ok := seen[resolved]; ok {
			continue
		}
		seen[resolved] = struct{}{}
		out = append(out, resolved)
	}
	AdditionalIncludeDirs = out
}

func assembleGoAsm(ctx context.Context, src, obj string, verbose bool) error {
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		goroot = runtime.GOROOT()
	}
	includeDir := goroot + "/src/runtime"

	if verbose {
		writeStderr("Running: go tool asm -I " + includeDir + src + "-o " + obj + "\n")
	}

	cmd := exec.CommandContext(ctx, "go", "tool", "asm", "-I", includeDir, src, "-o", obj)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func WriteFlatAssembledNotice(path string) {
	writeStderr("Assembled flat binary: " + path + "\n")
}

func validateArgs(args []string) error {
	for _, arg := range args {
		if err := utils.ValidateCLIArg(arg); err != nil {
			return err
		}
	}
	return nil
}

func CCForTarget() string         { return ccForTarget() }
func CXXForTarget() string        { return cxxForTarget() }
func GasCmdForTarget() string     { return gasCmdForTarget() }
func FormatFlagForTarget() string { return formatFlagForTarget() }

func ccForTarget() string {
	if strings.Contains(Target, "riscv") {
		if err := utils.CheckTool("zig"); err == nil {
			return "zig"
		}
		return "riscv64-unknown-elf-gcc"
	}
	switch {
	case isWasmTarget():
		if err := utils.CheckTool("emcc"); err == nil {
			return "emcc"
		}
		return "clang"
	case strings.Contains(Target, "arm"):
		return "arm-linux-gnueabihf-gcc"
	default:
		return "gcc"
	}
}

func isGoAsmFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte("TEXT ·")) ||
		bytes.Contains(data, []byte("#include \"textflag.h\""))
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

func getCompiler(src string) string {
	if strings.HasSuffix(src, ".m") {
		return "clang"
	}
	return ccForTarget()
}

func assembleWithNasm(ctx context.Context, src, obj string, debug, verbose bool) error {
	nasmPath := "nasm"
	if path, err := exec.LookPath("nasm"); err == nil {
		nasmPath = path
	}

	args := make([]string, 0, 8)
	args = append(args, "-f", "elf64", "-o", obj)
	if debug {
		args = append(args, "-g", "-F", "dwarf")
	}
	if len(AsmFlags) > 0 {
		args = append(args, AsmFlags...)
	}
	args = append(args, src)

	if verbose {
		writeStderr("Running: " + nasmPath + " " + strings.Join(args, " ") + "\n")
	}
	_, err := runCommand(ctx, verbose, nasmPath, args...)
	return err
}

func Assemble(ctx context.Context, src, obj string, debug, verbose bool, mode string) error {
	initAssemblerFlags()

	target := Target
	if ctxTarget, ok := ctx.Value(utils.TargetCtxKey).(string); ok && ctxTarget != "" {
		target = ctxTarget
	}

	if verbose {
		if UseNasm {
			writeStderr("Using NASM for .asm files\n")
		} else {
			writeStderr("Using internal assembler for .asm files\n")
		}
	}

	src = filepath.Clean(src)
	obj = filepath.Clean(obj)
	if seal.IsDecoyMode() {
		return emitDecoyObject(obj)
	}
	if err := utils.ValidateCLIPath(src); err != nil {
		return err
	}
	if err := utils.ValidateCLIPath(obj); err != nil {
		return err
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
			return errors.New("unsupported source mode: " + mode + " (supported: raw, auto)")
		}
		if isWasmTargetWithTarget(target) {
			return errors.New("cannot assemble .asm files for wasm target")
		}
		if strings.HasSuffix(src, ".asm") && UseNasm {
			return assembleWithNasm(ctx, src, obj, debug, verbose)
		}
		return assembleRawASM(ctx, src, obj, target)
	case ".s":
		if isGoAsmFile(src) {
			return assembleGoAsm(ctx, src, obj, verbose)
		}
		return assembleS(ctx, src, obj, verbose, target)
	case ".S":
		return compileCWithTarget(ctx, src, obj, verbose, ccForTargetWithTarget(target), target)
	case ".m", ".mm":
		return compileCWithTarget(ctx, src, obj, verbose, getCompilerWithTarget(src, target), target)
	case ".c":
		return compileCWithTarget(ctx, src, obj, verbose, ccForTargetWithTarget(target), target)
	case ".cpp", ".cc", ".cxx":
		return compileCWithTarget(ctx, src, obj, verbose, cxxForTargetWithTarget(target), target)
	default:
		return errors.New("unsupported source extension: " + ext + " (supported: .asm, .s, .S, .m, .c, .cpp, .cc, .cxx)")
	}
}

func assembleS(ctx context.Context, src, obj string, verbose bool, target string) error {
	_, err := runCommand(ctx, verbose, gasCmdForTargetWithTarget(target), "-o", obj, src)
	return err
}

func writeStderr(s string) {
	os.Stderr.WriteString(s)
}

func getGccIncludePath() string {
	cmd := exec.Command("gcc", "-print-file-name=include")
	out, err := cmd.Output()
	if err == nil {
		path := strings.TrimSpace(string(out))
		if filepath.IsAbs(path) {
			return path
		}
	}
	return ""
}

func compileC(ctx context.Context, src, obj string, verbose bool, compiler string) error {
	target := Target
	if ctxTarget, ok := ctx.Value(utils.TargetCtxKey).(string); ok && ctxTarget != "" {
		target = ctxTarget
	}
	return compileCWithTarget(ctx, src, obj, verbose, compiler, target)
}

func compileCWithTarget(ctx context.Context, src, obj string, verbose bool, compiler string, target string) error {
	compilerParts := strings.Fields(compiler)
	compilerBin := compilerParts[0]

	args := make([]string, 0, 8)
	args = append(args, "-c", src, "-o", obj)

	if strings.HasSuffix(src, ".m") {
		args = append(args, "-x", "objective-c")
		if gccInc := getGccIncludePath(); gccInc != "" {
			args = append(args, "-I"+gccInc)
		}
	}

	if compilerBin == "zig" && target != "" {
		args = append(args, "cc", "-target", target)
	}
	if len(compilerParts) > 1 {
		if compilerBin != "zig" || compilerParts[1] != "cc" {
			args = append(args, compilerParts[1:]...)
		}
	}

	CcFlagsOnce.Do(func() {
		if CcFlags != "" {
			CcFLagsParsed = strings.Fields(CcFlags)
		}
	})

	if len(AdditionalIncludeDirs) > 0 {
		for _, dir := range AdditionalIncludeDirs {
			args = append(args, "-I"+dir)
		}
	}
	if len(CcFLagsParsed) > 0 {
		args = append(args, CcFLagsParsed...)
	}

	if len(PCHIncludeArgs) > 0 {
		var headerPath string
		var newArgs []string
		for i := 0; i < len(PCHIncludeArgs); i++ {
			if PCHIncludeArgs[i] == "-include" && i+1 < len(PCHIncludeArgs) {
				headerPath = PCHIncludeArgs[i+1]
				i++
				continue
			}
			newArgs = append(newArgs, PCHIncludeArgs[i])
		}
		if headerPath != "" {
			pchPath, err := EnsurePCH(ctx, headerPath, compilerBin, args, target, verbose)
			if err == nil && pchPath != "" {
				args = append(args, "-include-pch", pchPath)
			} else {
				args = append(args, "-include", headerPath)
			}
		}
		if len(newArgs) > 0 {
			args = append(args, newArgs...)
		}
	}

	_, err := runCommand(ctx, verbose, compilerBin, args...)
	return err
}

func assembleRawASM(ctx context.Context, src, obj string, target string) error {
	if IsBinFormat() {
		return assembleRawBinary(src, obj, target)
	}
	return assembleBareMetalObject(ctx, src, obj)
}

func assembleRawBinary(src, obj string, target string) error {
	srcBuf, release, err := loadSourcePooled(src)
	if err != nil {
		return err
	}
	defer release()

	out, err := emitSourceRaw(srcBuf, selfTargetProfile(target))
	if err != nil {
		return err
	}
	return os.WriteFile(obj, out, 0o644)
}

func isWasmTarget() bool {
	return strings.Contains(Target, "wasm") || strings.Contains(Target, "wasm32")
}

func isWasmTargetWithTarget(target string) bool {
	return strings.Contains(target, "wasm") || strings.Contains(target, "wasm32")
}

func ccForTargetWithTarget(target string) string {
	if strings.Contains(target, "riscv") {
		if err := utils.CheckTool("zig"); err == nil {
			return "zig"
		}
		return "riscv64-unknown-elf-gcc"
	}
	switch {
	case isWasmTargetWithTarget(target):
		if err := utils.CheckTool("emcc"); err == nil {
			return "emcc"
		}
		return "clang"
	case strings.Contains(target, "arm"):
		return "arm-linux-gnueabihf-gcc"
	default:
		return "gcc"
	}
}

func cxxForTargetWithTarget(target string) string {
	switch {
	case isWasmTargetWithTarget(target):
		if err := utils.CheckTool("em++"); err == nil {
			return "em++"
		}
		return "clang++"
	case strings.Contains(target, "arm"):
		return "arm-linux-gnueabihf-g++"
	case strings.Contains(target, "riscv"):
		return "riscv64-unknown-elf-g++"
	default:
		return "g++"
	}
}

func gasCmdForTargetWithTarget(target string) string {
	if isWasmTargetWithTarget(target) {
		return "clang"
	}
	if strings.Contains(target, "arm") {
		return "arm-linux-gnueabihf-as"
	}
	if strings.Contains(target, "riscv") {
		return "riscv64-unknown-elf-as"
	}
	return "as"
}

func getCompilerWithTarget(src string, target string) string {
	if strings.HasSuffix(src, ".m") {
		return "clang"
	}
	return ccForTargetWithTarget(target)
}