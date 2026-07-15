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

package linker

import (
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
	"sync/atomic"
	"time"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
	"github.com/forgezero-cli/ForgeZero/internal/zig"
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

var PreferredLinker string

var (
	linkerOnce      sync.Once
	preferredLinker string
	hasLld          bool
	hasMold         bool
	toolPathCache   sync.Map
	bufferPool      = sync.Pool{New: func() any { return new(bytes.Buffer) }}
	extMap          = map[string]bool{
		".c": true, ".cpp": true, ".cc": true, ".cxx": true,
		".s": true, ".S": true,
		".asm": true, ".nasm": true,
		".fasm": true,
	}
	targetInfo   atomic.Value // stores *targetInfoState
	flagInitOnce sync.Once
	toolchainVal string
	muslVal      string
)

var ErrSkip = errors.New("skip this linker attempt")

type targetInfoState struct {
	target string
	isWasm bool
	isArm  bool
	isRisc bool
}

func getTargetInfo() (isWasm, isArm, isRisc bool) {
	v := targetInfo.Load()
	if v != nil {
		if st, ok := v.(*targetInfoState); ok {
			if st.target == Target {
				return st.isWasm, st.isArm, st.isRisc
			}
		}
	}
	st := &targetInfoState{
		target: Target,
		isWasm: strings.Contains(Target, "wasm") || strings.Contains(Target, "wasm32"),
		isArm:  strings.Contains(Target, "arm"),
		isRisc: strings.Contains(Target, "riscv"),
	}
	targetInfo.Store(st)
	return st.isWasm, st.isArm, st.isRisc
}

func cachedLookPath(name string) (string, error) {
	if v, ok := toolPathCache.Load(name); ok {
		return v.(string), nil
	}
	p, err := exec.LookPath(name)
	if err == nil {
		toolPathCache.Store(name, p)
	}
	return p, err
}

func detectLinker() {
	linkerOnce.Do(func() {
		if _, err := cachedLookPath("ld.lld"); err == nil {
			hasLld = true
			preferredLinker = "lld"
			return
		}
		if _, err := cachedLookPath("mold"); err == nil {
			hasMold = true
			preferredLinker = "mold"
			return
		}
		preferredLinker = ""
	})
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

func ResetLinkerDetection() {
	linkerOnce = sync.Once{}
	preferredLinker = ""
	hasLld = false
	hasMold = false
	toolPathCache = sync.Map{}
}

func SetRunner(r CmdRunner) {
	runner = r
}

func ResetRunner() {
	runner = &RealCmdRunner{}
}

func useZig() bool {
	if ZigRequested {
		return true
	}
	return ZigEnabled
}

func SetPreferredLinker(s string) {
	PreferredLinker = strings.ToLower(strings.TrimSpace(s))
}

func isWasmTarget() bool {
	w, _, _ := getTargetInfo()
	return w
}

func getLdFlags() []string {
	if LdFlags == "" {
		return nil
	}
	return strings.Fields(LdFlags)
}

func initFlags() {
	flagInitOnce.Do(func() {
		if f := flag.Lookup("toolchain"); f != nil {
			toolchainVal = f.Value.String()
		}
		if f := flag.Lookup("musl"); f != nil {
			muslVal = f.Value.String()
		}
	})
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

	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	cmd.Stdout = buf
	cmd.Stderr = buf

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
	w, arm, risc := getTargetInfo()
	if w {
		if _, err := cachedLookPath("wasm-ld"); err == nil {
			return "wasm-ld"
		}
		return "ld.lld"
	}
	if arm {
		return "arm-linux-gnueabihf-ld"
	}
	if risc {
		return "riscv64-unknown-elf-ld"
	}
	return "ld"
}

func gccForTarget() string {
	initFlags()
	if toolchainVal == "zig" || muslVal != "" {
		return "zig"
	}
	w, arm, risc := getTargetInfo()
	if w {
		if _, err := cachedLookPath("emcc"); err == nil {
			return "emcc"
		}
		return "clang"
	}
	if arm {
		return "arm-linux-gnueabihf-gcc"
	}
	if risc {
		return "riscv64-unknown-elf-gcc"
	}
	return "gcc"
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
		return errors.New("object file " + obj + " is empty")
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
		return linkWindowsImpl(ctx, []string{obj}, bin, verbose, sanitize, libs)
	}

	var linkErr error
	switch mode {
	case "raw":
		if linkErr = utils.CheckTool(ldForTarget()); linkErr != nil {
			return linkErr
		}
		linkErr = linkWithLd(ctx, []string{obj}, bin, verbose, libs)
	case "c":
		if useZig() {
			linkErr = linkWithZig(ctx, []string{obj}, bin, verbose, Target, sanitize, strict, libs)
			break
		}
		if linkErr = utils.CheckTool(gccForTarget()); linkErr != nil {
			return linkErr
		}
		linkErr = linkWithGcc(ctx, []string{obj}, bin, verbose, false, sanitize, strict, libs)
	case "auto":
		linkErr = tryAutoLink(ctx, []string{obj}, bin, verbose, sanitize, strict, libs)
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
	return LinkMultipleParallel(ctx, objFiles, bin, verbose, mode, noSymbolCheck, sanitize, strict, libs, runtime.GOMAXPROCS(0))
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
	return runBuild(ctx, files, backend)
}

func discoverSourceFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if extMap[filepath.Ext(path)] {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

func detectBackend(files []string) string {
	hasC := false
	hasGas := false
	hasNasm := false
	hasFasm := false
	for _, f := range files {
		ext := filepath.Ext(f)
		switch ext {
		case ".c", ".cpp", ".cc", ".cxx":
			hasC = true
		case ".s", ".S":
			hasGas = true
		case ".asm", ".nasm":
			hasNasm = true
		case ".fasm":
			hasFasm = true
		}
	}
	if hasC {
		return "gcc"
	}
	if hasNasm {
		return "nasm"
	}
	if hasFasm {
		return "fasm"
	}
	if hasGas {
		return "gas"
	}
	return "gcc"
}

func runBuild(ctx context.Context, files []string, backend string) error {
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
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func printInfo(msg string) {
	os.Stdout.WriteString(msg + "\n")
}

func buildLinkArgs(objs []string, bin string, sanitize bool, strict bool, libs []string, wasm bool, useFuseLd bool) []string {
	args := make([]string, 0, len(objs)+32)
	args = append(args, objs...)
	args = append(args, "-o", bin)
	if wasm {
		args = append(args, "--target=wasm32-unknown-unknown")
	}
	if useFuseLd {
		if fuse := getFuseLdFlag(); fuse != "" {
			args = append(args, fuse)
		}
	}
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
		if strict {
			args = append(args, "-fsanitize-address-use-after-return=always", "-fsanitize-address-use-after-scope")
		}
	}
	for _, lib := range libs {
		args = append(args, "-l", lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if flags := getLdFlags(); len(flags) > 0 {
		args = append(args, flags...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	return args
}

func runCompiler(ctx context.Context, compiler string, objs []string, bin string, verbose bool, allowNoPie bool, sanitize bool, strict bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	w, _, _ := getTargetInfo()
	wasm := w && compiler != "zig"
	useFuseLd := compiler != "zig"
	args := buildLinkArgs(objs, bin, sanitize, strict, libs, wasm, useFuseLd)
	if verbose {
		os.Stdout.WriteString("Running: " + compiler + " ")
		for i, a := range args {
			if i > 0 {
				os.Stdout.WriteString(" ")
			}
			os.Stdout.WriteString(a)
		}
		os.Stdout.WriteString("\n")
	}
	output, err := runLinkerCommand(ctx, verbose, compiler, args)
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if !allowNoPie {
		if !verbose {
			return errors.New(compiler + " link failed (use -verbose for details)")
		}
		var b strings.Builder
		b.WriteString(compiler)
		b.WriteString(" failed: ")
		b.WriteString(err.Error())
		b.WriteString("\n")
		b.WriteString(output)
		return errors.New(b.String())
	}
	args2 := make([]string, 0, len(args)+1)
	args2 = append(args2, "-no-pie")
	args2 = append(args2, args...)
	if verbose {
		printInfo(compiler + " failed, retrying with -no-pie")
		os.Stdout.WriteString("Running: " + compiler + " ")
		for i, a := range args2 {
			if i > 0 {
				os.Stdout.WriteString(" ")
			}
			os.Stdout.WriteString(a)
		}
		os.Stdout.WriteString("\n")
	}
	output2, err2 := runLinkerCommand(ctx, verbose, compiler, args2)
	if err2 == nil {
		return nil
	}
	if !verbose {
		return errors.New(compiler + " (with -no-pie) failed (use -verbose for details)")
	}
	var b strings.Builder
	b.WriteString(compiler)
	b.WriteString(" -no-pie failed: ")
	b.WriteString(err2.Error())
	b.WriteString("\n")
	b.WriteString(output2)
	return errors.New(b.String())
}

func linkWithGcc(ctx context.Context, objs []string, bin string, verbose bool, allowNoPie bool, sanitize bool, strict bool, libs []string) error {
	return runCompiler(ctx, gccForTarget(), objs, bin, verbose, allowNoPie, sanitize, strict, libs)
}

func linkWithClang(ctx context.Context, objs []string, bin string, verbose bool, allowNoPie bool, sanitize bool, libs []string) error {
	return runCompiler(ctx, clangForTarget(), objs, bin, verbose, allowNoPie, sanitize, false, libs)
}

func linkWithLd(ctx context.Context, objs []string, bin string, verbose bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	linker := ldForTarget()
	args := make([]string, 0, len(objs)+len(libs)+8)
	args = append(args, objs...)
	args = append(args, "-o", bin)
	for _, lib := range libs {
		args = append(args, "-l", lib)
	}
	args = ApplyLdFlags(args, LdScript, TextAddr)
	if flags := getLdFlags(); len(flags) > 0 {
		args = append(args, flags...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		os.Stdout.WriteString("Running: " + linker + " ")
		for i, a := range args {
			if i > 0 {
				os.Stdout.WriteString(" ")
			}
			os.Stdout.WriteString(a)
		}
		os.Stdout.WriteString("\n")
	}
	output, err := runLinkerCommand(ctx, verbose, linker, args)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if !verbose {
			return errors.New("ld link failed (use -verbose for details)")
		}
		var b strings.Builder
		b.WriteString(linker)
		b.WriteString(" failed: ")
		b.WriteString(err.Error())
		b.WriteString("\n")
		b.WriteString(output)
		return errors.New(b.String())
	}
	return nil
}

func linkWithZig(ctx context.Context, objs []string, bin string, verbose bool, target string, sanitize bool, strict bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	if !zig.IsAvailable() {
		return errors.New("zig not available")
	}
	return zig.Link(ctx, objs, bin, verbose, target, sanitize, strict, libs, Shared, LdScript, TextAddr, LdFlags)
}

func tryAutoLink(ctx context.Context, objs []string, bin string, verbose bool, sanitize bool, strict bool, libs []string) error {
	if ForceLD {
		if err := utils.CheckTool(ldForTarget()); err == nil {
			return linkWithLd(ctx, objs, bin, verbose, libs)
		}
		return errors.New("ld not found")
	}
	var lastErr error

	if len(objs) == 1 {
		srcFile := strings.TrimSuffix(objs[0], filepath.Ext(objs[0])) + ".m"
		if _, err := os.Stat(srcFile); err == nil {
			if verbose {
				printInfo("Objective-C detected! Bypassing Zig linker to use Clang with -lobjc")
			}
			libs = append(libs, "objc")
			if err := linkWithClang(ctx, objs, bin, verbose, true, sanitize, libs); err == nil {
				return nil
			}
			return errors.New("objective-c linking failed via Clang")
		}
	}

	type attempt struct {
		name string
		fn   func() error
	}
	attempts := []attempt{
		{
			name: "zig",
			fn: func() error {
				if !useZig() || !zig.IsAvailable() {
					return ErrSkip
				}
				return linkWithZig(ctx, objs, bin, verbose, Target, sanitize, strict, libs)
			},
		},
		{
			name: "clang",
			fn: func() error {
				if !strict {
					return ErrSkip
				}
				if _, err := cachedLookPath(clangForTarget()); err != nil {
					if verbose {
						printInfo("clang not found, falling back to gcc (limited strict mode)")
					}
					return ErrSkip
				}
				if verbose {
					printInfo("Strict mode: using clang for better sanitizers")
				}
				return linkWithClang(ctx, objs, bin, verbose, true, sanitize, libs)
			},
		},
		{
			name: "gcc",
			fn: func() error {
				if err := utils.CheckTool(gccForTarget()); err != nil {
					return ErrSkip
				}
				return linkWithGcc(ctx, objs, bin, verbose, true, sanitize, strict, libs)
			},
		},
		{
			name: "ld",
			fn: func() error {
				if err := utils.CheckTool(ldForTarget()); err != nil {
					return ErrSkip
				}
				return linkWithLd(ctx, objs, bin, verbose, libs)
			},
		},
	}

	for _, att := range attempts {
		err := att.fn()
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrSkip) {
			continue
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		lastErr = err
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("auto linking failed: no suitable linker")
}

func linkWindowsImpl(ctx context.Context, objs []string, bin string, verbose bool, sanitize bool, libs []string) error {
	if err := validateLinkCall(ctx, bin); err != nil {
		return err
	}
	if err := utils.CheckTool("clang"); err != nil {
		return err
	}
	args := make([]string, 0, len(objs)+16)
	args = append(args, objs...)
	args = append(args, "-o", bin, "-fuse-ld=lld")
	if sanitize {
		args = append(args, "-fsanitize=address", "-fsanitize=undefined")
	}
	for _, lib := range libs {
		args = append(args, "-l", lib)
	}
	args = ApplyGccLdFlags(args, LdScript, TextAddr)
	if flags := getLdFlags(); len(flags) > 0 {
		args = append(args, flags...)
	}
	if Shared {
		args = append(args, "-shared")
	}
	if verbose {
		os.Stdout.WriteString("Running: clang ")
		for i, a := range args {
			if i > 0 {
				os.Stdout.WriteString(" ")
			}
			os.Stdout.WriteString(a)
		}
		os.Stdout.WriteString("\n")
	}
	output, err := runLinkerCommand(ctx, verbose, "clang", args)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if !verbose {
			return errors.New("clang link failed (use -verbose for details)")
		}
		var b strings.Builder
		b.WriteString("clang failed: ")
		b.WriteString(err.Error())
		b.WriteString("\n")
		b.WriteString(output)
		return errors.New(b.String())
	}
	return nil
}
