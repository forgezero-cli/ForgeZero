package linker

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
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
	AutoBuild    bool
)

var (
	linkerOnce      sync.Once
	preferredLinker string // "lld", "mold", or "" for system ld
	hasLld          bool
	hasMold         bool
)

func detectLinker() {
	linkerOnce.Do(func() {
		if _, err := exec.LookPath("ld.lld"); err == nil {
			hasLld = true
			preferredLinker = "lld"
			return
		}
		if _, err := exec.LookPath("mold"); err == nil {
			hasMold = true
			preferredLinker = "mold"
			return
		}
		preferredLinker = ""
	})
}

func getLinkerName() string {
	detectLinker()
	if preferredLinker == "lld" {
		return "ld.lld"
	}
	if preferredLinker == "mold" {
		return "mold"
	}
	return "ld"
}

func getFuseLdFlag() string {
	detectLinker()
	if preferredLinker == "lld" {
		return "-fuse-ld=lld"
	}
	if preferredLinker == "mold" {
		return "-fuse-ld=mold"
	}
	return ""
}

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
			return "", errors.New("invalid argument for response file")
		}
		if err := utils.ValidateCLIArg(arg); err != nil {
			_ = f.Close()
			_ = os.Remove(name)
			return "", errors.New("invalid argument for response file: " + err.Error())
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
		return "", errors.New("linker executable is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, isReal := runner.(*RealCmdRunner); isReal {
		if err := utils.ValidateCLIArg(name); err != nil {
			return "", errors.New("invalid linker name: " + err.Error())
		}
		resolved, err := utils.FindExecutable(ctx, name)
		if err != nil {
			return "", errors.New("linker not found: " + err.Error())
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
	if base == "ld" || base == "ld.lld" || base == "wasm-ld" || base == "mold" {
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

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()

	out := buf.String()

	if verbose && len(out) > 0 {
		os.Stdout.WriteString(out)
	}

	return out, err
}

func validateLinkCall(ctx context.Context, output string) error {
	if ctx == nil {
		return errors.New("invalid linking context")
	}
	if output == "" {
		return errors.New("output file name is required")
	}
	if err := utils.ValidateCLIPath(output); err != nil {
		return errors.New("invalid output path: " + err.Error())
	}
	return nil
}

func ldForTarget() string {
	detectLinker()
	if isWasmTarget() {
		if _, err := exec.LookPath("wasm-ld"); err == nil {
			return "wasm-ld"
		}
		return "ld.lld"
	}
	if preferredLinker != "" {
		if preferredLinker == "lld" {
			return "ld.lld"
		}
		if preferredLinker == "mold" {
			return "mold"
		}
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
	if tcFlag := flag.Lookup("toolchain"); tcFlag != nil && tcFlag.Value.String() == "zig" {
		return "zig"
	}

	if muslFlag := flag.Lookup("musl"); muslFlag != nil && muslFlag.Value.String() != "" {
		return "zig"
	}

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
		return errors.New("object file " + obj + "is empty")
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
		return errors.New("unsupported mode: " + mode + "(valid: auto, c, raw)")
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
			return errors.New("flat binary link requires exactly one object")
		}
		return linkFlatBinary(ctx, objFiles[0], bin)
	}
	if len(objFiles) == 0 {
		return errors.New("no object files to link")
	}
	sort.Strings(objFiles)
	for _, obj := range objFiles {
		info, err := os.Stat(obj)
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			return errors.New("object file " + obj + "is empty")
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
		return errors.New("unsupported mode: " + mode + " (valid: auto, c, raw)")
	}
	if linkErr != nil {
		return linkErr
	}
	if cfg := utils.ConfigFromContext(ctx); cfg != nil && cfg.DeterministicStrip {
		_, _ = utils.ScrubHostPaths(bin, utils.GetExecutionRoot())
	}
	return nil
}

func writeStderr(s string) {
	os.Stderr.WriteString(s)
}

func AutoBuildProject(ctx context.Context) error {
	if !AutoBuild {
		return nil
	}

	toolchain := gccForTarget()
	if err := utils.CheckTool(toolchain); err != nil {
		writeStderr("toolchain: ")
		writeStderr(toolchain)
		writeStderr(" not found\n")
		return err
	}

	files := discoverSourceFiles(".")
	if len(files) == 0 {
		writeStderr("no source files found\n")
		return nil
	}

	backend := detectBackend(files)
	return runBuild(files, backend)
}

func discoverSourceFiles(root string) []string {
	var files []string
	exts := map[string]bool{
		".c": true, ".cpp": true, ".cc": true, ".cxx": true,
		".s": true, ".S": true,
		".asm": true, ".nasm": true,
		".fasm": true,
	}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && exts[filepath.Ext(path)] {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		_ = err
	}

	return files
}

func detectBackend(files []string) string {
	for _, f := range files {
		ext := filepath.Ext(f)
		switch ext {
		case ".c", ".cpp", ".cc", ".cxx":
			return "gcc"
		case ".s", ".S":
			return "gas"
		case ".asm", ".nasm":
			return "nasm"
		case ".fasm":
			return "fasm"
		}
	}
	return "gcc"
}

func runBuild(files []string, backend string) error {
	var args []string
	switch backend {
	case "gcc":
		args = append([]string{"gcc"}, files...)
	case "nasm":
		args = append([]string{"nasm", "-f", "elf64"}, files...)
	case "fasm":
		args = append([]string{"fasm"}, files...)
	case "gas":
		args = append([]string{"gcc", "-c"}, files...)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func printInfo(msg string) {
	os.Stdout.WriteString(msg + "\n")
}

func tryAutoLink(ctx context.Context, obj, bin string, verbose bool, sanitize bool, strict bool, libs []string) error {
	var lastErr error
	if ForceLD {
		if err := utils.CheckTool(ldForTarget()); err == nil {
			return linkWithLd(ctx, obj, bin, verbose, libs)
		}
		return errors.New("ld not found")
	}
	isObjC := false

	srcFile := strings.TrimSuffix(obj, filepath.Ext(obj)) + ".m"
	if _, err := os.Stat(srcFile); err == nil {
		isObjC = true
	}

	if isObjC {
		if verbose {
			printInfo("Objective-C detected! Bypassing Zig linker to use Clang with -lobjc")
		}

		libs = append(libs, "objc")

		if err := linkWithClang(ctx, obj, bin, verbose, true, sanitize, libs); err == nil {
			return nil
		}

		return errors.New("objective-c linking failed via Clang")
	}

	if useZig() {
		if verbose {
			printInfo("Strict mode: using zig for deterministic linking")
		}
		if err := linkWithZig(ctx, []string{obj}, bin, verbose, Target, sanitize, strict, libs); err == nil {
			return nil
		}
	}
	if strict {
		if _, err := exec.LookPath(clangForTarget()); err == nil {
			if verbose {
				printInfo("Strict mode: using clang for better sanitizers")
			}
			err = linkWithClang(ctx, obj, bin, verbose, true, sanitize, libs)
			if err == nil {
				return nil
			}
		} else if verbose {
			printInfo("clang not found, falling back to gcc (limited strict mode)")
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
	return errors.New("auto linking failed: no suitable linker")
}

func linkWithClang(ctx context.Context, obj, bin string, verbose bool, allowNoPieFallback bool, sanitize bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	args := []string{obj, "-o", bin}
	if isWasmTarget() {
		args = append([]string{"--target=wasm32-unknown-unknown"}, args...)
	}
	if fuse := getFuseLdFlag(); fuse != "" {
		args = append(args, fuse)
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
		printInfo("Running: " + clangForTarget() + " " + strings.Join(args, " "))
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
			return errors.New("clang link failed (use -verbose for details)")
		}
		return errors.New("clang failed: " + err.Error() + "\n" + output)
	}
	argsWithNoPie := append([]string{"-no-pie"}, args...)
	if verbose {
		printInfo("clang failed, retrying with -no-pie")
	}
	output2, err2 := runLinkerCommand(ctx, verbose, clangForTarget(), argsWithNoPie)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return errors.New("clang (with -no-pie) failed (use -verbose for details)")
	}
	return errors.New("clang -no-pie failed:" + err2.Error() + "\n" + output2)
}

func linkWithGcc(ctx context.Context, obj, bin string, verbose bool, allowNoPieFallback bool, sanitize bool, strict bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	args := []string{obj, "-o", bin}
	if isWasmTarget() && gccForTarget() == "clang" {
		args = append([]string{"--target=wasm32-unknown-unknown"}, args...)
	}
	if fuse := getFuseLdFlag(); fuse != "" {
		args = append(args, fuse)
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
		printInfo("Running: " + gccForTarget() + " " + strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, gccForTarget(), args)
	if err == nil {
		return nil
	}
	if !allowNoPieFallback {
		if !verbose {
			return errors.New("gcc link failed (use -verbose for details)")
		}
		return errors.New("gcc failed: " + err.Error() + "\n" + output)
	}
	if verbose {
		printInfo("gcc failed, retrying with -no-pie")
	}
	argsWithNoPie := append([]string{"-no-pie"}, args...)
	if verbose {
		printInfo("Running: " + gccForTarget() + " " + strings.Join(argsWithNoPie, " "))
	}
	output2, err2 := runLinkerCommand(ctx, verbose, gccForTarget(), argsWithNoPie)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return errors.New("gcc (with -no-pie) failed (use -verbose for details)")
	}
	return errors.New("gcc -no-pie failed: " + err2.Error() + "\n" + output2)
}

func linkWithZig(ctx context.Context, objFiles []string, bin string, verbose bool, target string, sanitize bool, strict bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	if !zig.IsAvailable() {
		return errors.New("zig not available")
	}
	return zig.Link(ctx, objFiles, bin, verbose, target, sanitize, strict, libs, Shared, LdScript, TextAddr, LdFlags)
}

func linkWithLd(ctx context.Context, obj, bin string, verbose bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	linker := ldForTarget()
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
		printInfo("Running: " + linker + " " + strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, linker, args)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if !verbose {
			return errors.New("ld link failed (use -verbose for details)")
		}
		return errors.New(linker + " failed: " + err.Error() + "\n" + output)
	}
	return nil
}

func tryAutoLinkMultiple(ctx context.Context, objFiles []string, bin string, verbose bool, sanitize bool, strict bool, libs []string) error {
	sort.Strings(objFiles)
	if ForceLD {
		if err := utils.CheckTool(ldForTarget()); err == nil {
			return linkMultipleWithLd(ctx, objFiles, bin, verbose, libs)
		}
		return errors.New("ld not found")
	}
	var lastErr error
	if useZig() {
		if verbose {
			printInfo("Strict mode: using zig for deterministic linking")
		}
		if err := linkWithZig(ctx, objFiles, bin, verbose, Target, sanitize, strict, libs); err == nil {
			return nil
		}
	}
	if strict {
		if _, err := exec.LookPath(clangForTarget()); err == nil {
			if verbose {
				printInfo("Strict mode: using clang for better sanitizers")
			}
			err = linkMultipleWithClang(ctx, objFiles, bin, verbose, true, sanitize, libs)
			if err == nil {
				return nil
			}
		} else if verbose {
			printInfo("clang not found, falling back to gcc (limited strict mode)")
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
	return errors.New("auto linking failed: no suitable linker")
}

func linkMultipleWithClang(ctx context.Context, objFiles []string, bin string, verbose bool, allowNoPieFallback bool, sanitize bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	args := append(objFiles, "-o", bin)
	if isWasmTarget() {
		args = append([]string{"--target=wasm32-unknown-unknown"}, args...)
	}
	if fuse := getFuseLdFlag(); fuse != "" {
		args = append(args, fuse)
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
		printInfo("Running: " + clangForTarget() + " " + strings.Join(args, " "))
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
			return errors.New("clang link failed (use -verbose for details)")
		}
		return errors.New("clang failed: " + err.Error() + "\n" + output)
	}
	argsWithNoPie := append([]string{"-no-pie"}, args...)
	if verbose {
		printInfo("clang failed, retrying with -no-pie\n")
	}
	output2, err2 := runLinkerCommand(ctx, verbose, clangForTarget(), argsWithNoPie)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return errors.New("clang (with -no-pie) failed (use -verbose for details)")
	}
	return errors.New("clang -no-pie failed: " + err2.Error() + "\n" + output2)
}

func linkMultipleWithGcc(ctx context.Context, objFiles []string, bin string, verbose bool, allowNoPieFallback bool, sanitize bool, strict bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	args := append(objFiles, "-o", bin)
	if isWasmTarget() && gccForTarget() == "clang" {
		args = append([]string{"--target=wasm32-unknown-unknown"}, args...)
	}
	if fuse := getFuseLdFlag(); fuse != "" {
		args = append(args, fuse)
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
		printInfo("Running: " + gccForTarget() + " " + strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, gccForTarget(), args)
	if err == nil {
		return nil
	}
	if !allowNoPieFallback {
		if !verbose {
			return errors.New("gcc link failed (use -verbose for details)")
		}
		return errors.New("gcc failed: " + err.Error() + "\n" + output)
	}
	if verbose {
		printInfo("gcc failed, retrying with -no-pie")
	}
	argsWithNoPie := append([]string{"-no-pie"}, args...)
	if verbose {
		printInfo("Running: " + gccForTarget() + " " + strings.Join(argsWithNoPie, " "))
	}
	output2, err2 := runLinkerCommand(ctx, verbose, gccForTarget(), argsWithNoPie)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return errors.New("gcc (with -no-pie) failed (use -verbose for details)")
	}
	return errors.New("gcc -no-pie failed: " + err2.Error() + "\n" + output2)
}

func linkMultipleWithLd(ctx context.Context, objFiles []string, bin string, verbose bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	linker := ldForTarget()
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
		printInfo("Running: " + linker + " " + strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, linker, args)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if !verbose {
			return errors.New("ld link failed (use -verbose for details)")
		}
		return errors.New(linker + " failed: " + err.Error() + "\n" + output)
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
		printInfo("Running: clang: " + strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, "clang", args)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if !verbose {
			return errors.New("clang link failed (use -verbose for details)")
		}
		return errors.New("clang failed: " + err.Error() + "\n" + output)
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
		printInfo("Running: clang " + strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, "clang", args)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if !verbose {
			return errors.New("clang link failed (use -verbose for details)")
		}
		return errors.New("clang failed: " + err.Error() + "\n" + output)
	}
	return nil
}