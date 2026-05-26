package linker

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"fz/internal/config"
	"fz/internal/utils"
	"fz/internal/zig"
)

var (
	runner       CmdRunner = &RealCmdRunner{}
	LdScript     string
	TextAddr     string
	Target       = "x86_64-linux-gnu"
	LdFlags      string
	Shared       bool
	ZigRequested bool
	ZigEnabled   bool
	ForceLD      bool
)

func SetRunner(r CmdRunner) {
	runner = r
}

func ResetRunner() {
	runner = &RealCmdRunner{}
}

var lookPathFunc = exec.LookPath

func useZig() bool {
	if ZigRequested {
		return true
	}
	return ZigEnabled
}

func isWasmTarget() bool {
	return strings.Contains(Target, "wasm") || strings.Contains(Target, "wasm32")
}

func shouldUseResponseFile(args []string) bool {
	if len(args) > 128 {
		return true
	}
	total := 0
	for _, arg := range args {
		total += len(arg) + 1
	}
	return total > 8192
}

func createResponseFile(args []string) (string, error) {
	f, err := os.CreateTemp("", "fz_link_args_*.rsp")
	if err != nil {
		return "", err
	}
	name := f.Name()
	_ = f.Chmod(utils.FilePerm)

	writer := bufio.NewWriterSize(f, 64*1024)

	for _, arg := range args {
		if strings.ContainsAny(arg, "\n\r\x00") {
			_ = f.Close()
			_ = os.Remove(name)
			return "", fmt.Errorf("invalid argument for response file")
		}
		if err := utils.ValidateCLIArg(arg); err != nil {
			_ = f.Close()
			_ = os.Remove(name)
			return "", fmt.Errorf("invalid argument for response file: %w", err)
		}
		_, _ = writer.WriteString(arg)
		_ = writer.WriteByte('\n')
	}

	if err := writer.Flush(); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return "", err
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}

	return name, nil
}

func runLinkerCommand(ctx context.Context, verbose bool, name string, args []string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("linker executable is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, isReal := runner.(*RealCmdRunner); isReal {
		if err := utils.ValidateCLIArg(name); err != nil {
			return "", fmt.Errorf("invalid linker name: %w", err)
		}
		resolved, err := utils.FindExecutable(ctx, name)
		if err != nil {
			return "", fmt.Errorf("linker not found: %w", err)
		}
		name = resolved
	}
	if shouldUseResponseFile(args) {
		path, err := createResponseFile(args)
		if err != nil {
			return "", err
		}
		defer os.Remove(path)
		args = []string{"@" + path}
	}
	if _, isReal := runner.(*RealCmdRunner); isReal && isLdExecutable(name) {
		ctx, cancel := ensureContextTimeout(ctx, 30*time.Second)
		defer cancel()
		return runLinkerCombinedOutput(ctx, verbose, name, args)
	}
	return runner.Run(ctx, verbose, name, args...)
}

func ensureContextTimeout(ctx context.Context, min time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if deadline, ok := ctx.Deadline(); ok {
		if time.Until(deadline) >= min {
			return ctx, func() {}
		}
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, min)
}

func isLdExecutable(name string) bool {
	base := filepath.Base(name)
	if base == "ld" || base == "ld.lld" || base == "wasm-ld" {
		return true
	}
	return strings.HasSuffix(base, "-ld")
}

func runLinkerCombinedOutput(ctx context.Context, verbose bool, name string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir := utils.GetExecutionRoot(); dir != "" {
		cmd.Dir = dir
	}
	if cfg := utils.ConfigFromContext(ctx); cfg != nil && cfg.Isolation != config.IsolationNone {
		cmd.Env = utils.SafeEnv(cfg)
	}
	output, err := cmd.CombinedOutput()
	out := string(output)
	if verbose && len(out) > 0 {
		fmt.Fprint(os.Stdout, out)
	}
	return out, err
}

func validateLinkCall(ctx context.Context, output string) error {
	if ctx == nil {
		return fmt.Errorf("invalid linking context")
	}
	if output == "" {
		return fmt.Errorf("output file name is required")
	}
	if err := utils.ValidateCLIPath(output); err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}
	return nil
}

func ldForTarget() string {
	if isWasmTarget() {
		if _, err := exec.LookPath("wasm-ld"); err == nil {
			return "wasm-ld"
		}
		return "ld.lld"
	}
	switch {
	case strings.Contains(Target, "arm"):
		return "arm-linux-gnueabihf-ld"
	case strings.Contains(Target, "riscv"):
		return "riscv64-unknown-elf-ld"
	default:
		return "ld"
	}
}

func gccForTarget() string {
	if isWasmTarget() {
		if _, err := exec.LookPath("emcc"); err == nil {
			return "emcc"
		}
		return "clang"
	}
	switch {
	case strings.Contains(Target, "arm"):
		return "arm-linux-gnueabihf-gcc"
	case strings.Contains(Target, "riscv"):
		return "riscv64-unknown-elf-gcc"
	default:
		return "gcc"
	}
}

func clangForTarget() string {
	return "clang"
}

func Link(ctx context.Context, obj, bin string, verbose bool, mode string, noSymbolCheck bool, sanitize bool, strict bool, libs []string) error {
	if err := utils.CheckFileExists(obj); err != nil {
		return err
	}
	info, err := os.Stat(obj)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fmt.Errorf("object file %s is empty", obj)
	}
	if err := utils.EnsureDir(bin); err != nil {
		return err
	}
	if shouldSkipLinker() {
		return linkFlatBinary(ctx, obj, bin)
	}
	if !noSymbolCheck {
		if err := CheckDuplicateSymbols(ctx, []string{obj}, verbose); err != nil {
			return err
		}
	}

	if runtime.GOOS == "windows" {
		return linkWindowsImpl(ctx, obj, bin, verbose, mode, sanitize, libs)
	}

	var linkErr error
	switch mode {
	case "raw":
		if linkErr = utils.CheckTool(ldForTarget()); linkErr != nil {
			return linkErr
		}
		linkErr = linkWithLd(ctx, obj, bin, verbose, libs)
	case "c":
		if useZig() {
			linkErr = linkWithZig(ctx, []string{obj}, bin, verbose, Target, sanitize, strict, libs)
			break
		}
		if linkErr = utils.CheckTool(gccForTarget()); linkErr != nil {
			return linkErr
		}
		linkErr = linkWithGcc(ctx, obj, bin, verbose, false, sanitize, strict, libs)
	case "auto":
		linkErr = tryAutoLink(ctx, obj, bin, verbose, sanitize, strict, libs)
	default:
		return fmt.Errorf("unsupported mode: %s (valid: auto, c, raw)", mode)
	}
	if linkErr != nil {
		return linkErr
	}
	if cfg := utils.ConfigFromContext(ctx); cfg != nil && cfg.DeterministicStrip {
		_, _ = utils.ScrubHostPaths(bin, utils.GetExecutionRoot())
	}
	return nil
}

func LinkMultiple(ctx context.Context, objFiles []string, bin string, verbose bool, mode string, noSymbolCheck bool, sanitize bool, strict bool, libs []string) error {
	if shouldSkipLinker() {
		if len(objFiles) != 1 {
			return fmt.Errorf("flat binary link requires exactly one object")
		}
		return linkFlatBinary(ctx, objFiles[0], bin)
	}
	if len(objFiles) == 0 {
		return fmt.Errorf("no object files to link")
	}
	sort.Strings(objFiles)
	for _, obj := range objFiles {
		info, err := os.Stat(obj)
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			return fmt.Errorf("object file %s is empty", obj)
		}
	}
	if err := utils.EnsureDir(bin); err != nil {
		return err
	}
	if !noSymbolCheck {
		if err := CheckDuplicateSymbols(ctx, objFiles, verbose); err != nil {
			return err
		}
	}

	if runtime.GOOS == "windows" {
		return linkMultipleWindowsImpl(ctx, objFiles, bin, verbose, mode, sanitize, libs)
	}

	var linkErr error
	switch mode {
	case "raw":
		if linkErr = utils.CheckTool(ldForTarget()); linkErr != nil {
			return linkErr
		}
		linkErr = linkMultipleWithLd(ctx, objFiles, bin, verbose, libs)
	case "c":
		if useZig() {
			linkErr = linkWithZig(ctx, objFiles, bin, verbose, Target, sanitize, strict, libs)
			break
		}
		if linkErr = utils.CheckTool(gccForTarget()); linkErr != nil {
			return linkErr
		}
		linkErr = linkMultipleWithGcc(ctx, objFiles, bin, verbose, false, sanitize, strict, libs)
	case "auto":
		linkErr = tryAutoLinkMultiple(ctx, objFiles, bin, verbose, sanitize, strict, libs)
	default:
		return fmt.Errorf("unsupported mode: %s (valid: auto, c, raw)", mode)
	}
	if linkErr != nil {
		return linkErr
	}
	if cfg := utils.ConfigFromContext(ctx); cfg != nil && cfg.DeterministicStrip {
		_, _ = utils.ScrubHostPaths(bin, utils.GetExecutionRoot())
	}
	return nil
}

func tryAutoLink(ctx context.Context, obj, bin string, verbose bool, sanitize bool, strict bool, libs []string) error {
	var lastErr error
	if ForceLD {
		if err := utils.CheckTool(ldForTarget()); err == nil {
			return linkWithLd(ctx, obj, bin, verbose, libs)
		}
		return fmt.Errorf("ld not found")
	}
	if useZig() {
		if verbose {
			fmt.Println("Strict mode: using zig for deterministic linking")
		}
		if err := linkWithZig(ctx, []string{obj}, bin, verbose, Target, sanitize, strict, libs); err == nil {
			return nil
		}
	}
	if strict {
		if _, err := exec.LookPath(clangForTarget()); err == nil {
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
	if err := utils.CheckTool(gccForTarget()); err == nil {
		err = linkWithGcc(ctx, obj, bin, verbose, true, sanitize, strict, libs)
		if err == nil {
			return nil
		}
	}
	if err := utils.CheckTool(ldForTarget()); err == nil {
		if err := linkWithLd(ctx, obj, bin, verbose, libs); err == nil {
			return nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("auto linking failed: no suitable linker")
}

func linkWithClang(ctx context.Context, obj, bin string, verbose bool, allowNoPieFallback bool, sanitize bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	args := []string{obj, "-o", bin}
	if isWasmTarget() {
		args = append([]string{"--target=wasm32-unknown-unknown"}, args...)
	}
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
		args = append(args, "-fsanitize-address-use-after-return=always")
		args = append(args, "-fsanitize-address-use-after-scope")
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", clangForTarget(), strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, clangForTarget(), args)
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
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
	output2, err2 := runLinkerCommand(ctx, verbose, clangForTarget(), argsWithNoPie)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return fmt.Errorf("clang (with -no-pie) failed (use -verbose for details)")
	}
	return fmt.Errorf("clang -no-pie failed: %w\n%s", err2, output2)
}

func linkWithGcc(ctx context.Context, obj, bin string, verbose bool, allowNoPieFallback bool, sanitize bool, strict bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	args := []string{obj, "-o", bin}
	if isWasmTarget() && gccForTarget() == "clang" {
		args = append([]string{"--target=wasm32-unknown-unknown"}, args...)
	}
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
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", gccForTarget(), strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, gccForTarget(), args)
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
		fmt.Printf("Running: %s %s\n", gccForTarget(), strings.Join(argsWithNoPie, " "))
	}
	output2, err2 := runLinkerCommand(ctx, verbose, gccForTarget(), argsWithNoPie)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return fmt.Errorf("gcc (with -no-pie) failed (use -verbose for details)")
	}
	return fmt.Errorf("gcc -no-pie failed: %w\n%s", err2, output2)
}

func linkWithZig(ctx context.Context, objFiles []string, bin string, verbose bool, target string, sanitize bool, strict bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	if !zig.IsAvailable() {
		return fmt.Errorf("zig not available")
	}
	return zig.Link(ctx, objFiles, bin, verbose, target, sanitize, strict, libs, Shared, LdScript, TextAddr, LdFlags)
}

func linkWithLd(ctx context.Context, obj, bin string, verbose bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	args := []string{obj, "-o", bin}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", ldForTarget(), strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, ldForTarget(), args)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if !verbose {
			return fmt.Errorf("ld link failed (use -verbose for details)")
		}
		return fmt.Errorf("ld failed: %w\n%s", err, output)
	}
	return nil
}

func tryAutoLinkMultiple(ctx context.Context, objFiles []string, bin string, verbose bool, sanitize bool, strict bool, libs []string) error {
	sort.Strings(objFiles)
	if ForceLD {
		if err := utils.CheckTool(ldForTarget()); err == nil {
			return linkMultipleWithLd(ctx, objFiles, bin, verbose, libs)
		}
		return fmt.Errorf("ld not found")
	}
	var lastErr error
	if useZig() {
		if verbose {
			fmt.Println("Strict mode: using zig for deterministic linking")
		}
		if err := linkWithZig(ctx, objFiles, bin, verbose, Target, sanitize, strict, libs); err == nil {
			return nil
		}
	}
	if strict {
		if _, err := exec.LookPath(clangForTarget()); err == nil {
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
	if err := utils.CheckTool(gccForTarget()); err == nil {
		err = linkMultipleWithGcc(ctx, objFiles, bin, verbose, true, sanitize, strict, libs)
		if err == nil {
			return nil
		}
	}
	if err := utils.CheckTool(ldForTarget()); err == nil {
		if err := linkMultipleWithLd(ctx, objFiles, bin, verbose, libs); err == nil {
			return nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("auto linking failed: no suitable linker")
}

func linkMultipleWithClang(ctx context.Context, objFiles []string, bin string, verbose bool, allowNoPieFallback bool, sanitize bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	args := append(objFiles, "-o", bin)
	if isWasmTarget() {
		args = append([]string{"--target=wasm32-unknown-unknown"}, args...)
	}
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
		args = append(args, "-fsanitize-address-use-after-return=always")
		args = append(args, "-fsanitize-address-use-after-scope")
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", clangForTarget(), strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, clangForTarget(), args)
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
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
	output2, err2 := runLinkerCommand(ctx, verbose, clangForTarget(), argsWithNoPie)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return fmt.Errorf("clang (with -no-pie) failed (use -verbose for details)")
	}
	return fmt.Errorf("clang -no-pie failed: %w\n%s", err2, output2)
}

func linkMultipleWithGcc(ctx context.Context, objFiles []string, bin string, verbose bool, allowNoPieFallback bool, sanitize bool, strict bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	args := append(objFiles, "-o", bin)
	if isWasmTarget() && gccForTarget() == "clang" {
		args = append([]string{"--target=wasm32-unknown-unknown"}, args...)
	}
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
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", gccForTarget(), strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, gccForTarget(), args)
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
		fmt.Printf("Running: %s %s\n", gccForTarget(), strings.Join(argsWithNoPie, " "))
	}
	output2, err2 := runLinkerCommand(ctx, verbose, gccForTarget(), argsWithNoPie)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return fmt.Errorf("gcc (with -no-pie) failed (use -verbose for details)")
	}
	return fmt.Errorf("gcc -no-pie failed: %w\n%s", err2, output2)
}

func linkMultipleWithLd(ctx context.Context, objFiles []string, bin string, verbose bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	args := append(objFiles, "-o", bin)
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: %s %s\n", ldForTarget(), strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, ldForTarget(), args)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if !verbose {
			return fmt.Errorf("ld link failed (use -verbose for details)")
		}
		return fmt.Errorf("ld failed: %w\n%s", err, output)
	}
	return nil
}

func linkWindowsImpl(ctx context.Context, obj, bin string, verbose bool, mode string, sanitize bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	if err := utils.CheckTool("clang"); err != nil {
		return err
	}
	args := []string{obj, "-o", bin, "-fuse-ld=lld"}
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: clang %s\n", strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, "clang", args)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if !verbose {
			return fmt.Errorf("clang link failed (use -verbose for details)")
		}
		return fmt.Errorf("clang failed: %w\n%s", err, output)
	}
	return nil
}

func linkMultipleWindowsImpl(ctx context.Context, objFiles []string, bin string, verbose bool, mode string, sanitize bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	if err := utils.CheckTool("clang"); err != nil {
		return err
	}
	args := append(objFiles, "-o", bin, "-fuse-ld=lld")
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
	}
	for _, lib := range libs {
		args = append(args, "-l"+lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if LdFlags != "" {
		args = append(args, strings.Fields(LdFlags)...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		fmt.Printf("Running: clang %s\n", strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, "clang", args)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if !verbose {
			return fmt.Errorf("clang link failed (use -verbose for details)")
		}
		return fmt.Errorf("clang failed: %w\n%s", err, output)
	}
	return nil
}
