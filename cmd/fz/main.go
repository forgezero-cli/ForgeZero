// Copyright (c) 2026 Alex Voste. MIT License.
// PROPERTY OF FORGEZERO CORE TEAM.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fz/internal/assembler"
	"fz/internal/audit"
	"fz/internal/bench"
	"fz/internal/builder"
	"fz/internal/compilecommands"
	"fz/internal/config"
	"fz/internal/doctor"
	fzvfs "fz/internal/fs"
	"fz/internal/ignore"
	initpkg "fz/internal/init"
	"fz/internal/linker"
	"fz/internal/man"
	"fz/internal/pkgman"
	"fz/internal/sbom"
	"fz/internal/shell"
	"fz/internal/updater"
	"fz/internal/utils"
	"fz/internal/verify"
	"fz/internal/watcher"
)

type BuildReport struct {
	Status      string   `json:"status"`
	ExitCode    int      `json:"exit_code"`
	DurationMs  int64    `json:"duration_ms"`
	Binary      string   `json:"binary,omitempty"`
	SourceFiles []string `json:"source_files,omitempty"`
	ObjectFiles []string `json:"object_files,omitempty"`
	Error       string   `json:"error,omitempty"`
}

const (
	versionCore     = "3.1.0-Aegis"
	versionCodename = "Sovereign Engineering Update"
)

var version = "3.1.0 Aegis @latest"

func versionText() string {
	var b strings.Builder
	b.Grow(256)
	b.WriteString("ForgeZero v")
	b.WriteString(versionCore)
	b.WriteString(" [")
	b.WriteString(versionCodename)
	b.WriteString("]\nBuild: ")
	b.WriteString(time.Now().Format("2006-01-02"))
	b.WriteString(" | OS: ")
	b.WriteString(runtime.GOOS)
	b.WriteByte('/')
	b.WriteString(runtime.GOARCH)
	b.WriteString(" | VFS: ")
	b.WriteString(fzvfs.ImplName())
	b.WriteString(" | Security: Aegis-Hardened\n")
	b.WriteString("(c) Alex Voste. Binary Integrity: Verified.\n")
	return b.String()
}

func outputVersion() {
	fmt.Print(versionText())
}

func helpText() string {
	var b strings.Builder
	b.Grow(4096)
	b.WriteString(`
fz – assembly & C build tool (ForgeZero `)
	b.WriteString(versionCore)
	b.WriteString(`)

Usage:
  fz [options] (-asm <file> | -cc <file> | -dir <dir> | (no args with config))
  fz audit [options]
  fz sbom [options]
  fz doctor [options]
  fz verify [options]
  fz bench [options]
  fz pm <subcommand> [args]

Options:
  -asm <file>            Assembler source (.asm, .s, .S, .fasm)
  -cc <file>             C source (strict warnings enabled)
  -dir <dir>             Build all supported files recursively
  -out <name>            Output binary name
  -out-obj <name>        Object file name (single file only)
  -mode <auto|c|raw>     Linking mode (default: auto)
  -debug                 Emit debug symbols (-g)
  -verbose               Print every command executed
  -keep-obj              Keep temporary object files (when using -dir)
  -no-cache              Disable incremental cache
  -no-symbol-check       Skip duplicate symbol pre‑check
  -sanitize              Enable sanitizers for C (default: true)
  -no-sanitize           Disable sanitizers
  -strict                Enable aggressive sanitizers (use-after-return, use-after-scope) – prefers clang
  -toolchain <auto|zig>  Select toolchain: auto or zig
  -clean                 Remove all build artifacts (.fz_objs, .fz_cache, binaries)
  -watch                 Watch files and auto‑rebuild
  -json                  Output build report in JSON (for CI/CD)
  -config <file>         Config file (default: .fz.yaml, fz.yaml, .fz.yml, fz.yml)
  -man                   Generate roff man page and exit
  -format <elf32|elf64|bin> Output format: elf64 (default), elf32, bin (flat binary)
  -T <file>              Linker script (passed to ld)
  -Ttext <addr>          Set text segment address
  -j <n>                 Number of parallel jobs (0 = auto = CPU cores)
  -target <triple>       Target triple (default: x86_64-linux-gnu, experimental: wasm)
  -type <executable|static> Build type: executable (default) or static (library)
  -lib                   Shortcut for -type static
  -compile-commands      Generate compile_commands.json for LSP and exit
  -init                  Initialize project: create .fz.yaml and .fzignore
  -shell                 Run interactive shell
  -update                Update fz to the latest version
  -h, --help             Show this help
  -v, --version          Show version

Examples:
  fz -asm boot.asm
  fz -cc main.c -strict -verbose
  fz -dir ./src -out myapp -watch
  fz -json -cc test.c
  fz -dir . -clean
  fz -asm boot.asm -format bin -out boot.bin
  fz -target arm-linux-gnueabihf -cc test.c -out test_arm
  fz sbom -out sbom.json
  fz doctor -root .
  fz doctor -json
  fz verify --update
  fz bench -dir ./src -json

Supported extensions: .asm, .s, .S, .fasm, .c, .cpp, .cc, .cxx

Aegis Security & Integrity (v3.1.0):
  doctor [options]        Self-audit: toolchain reachability, R/W permissions, platform
                          -root <dir>   project root (default: cwd)
                          -json         machine-readable report; exit 1 if unhealthy
  audit [options]         SAST scan: secrets, license risks, vendor keyword matches
                          -config -vendor -verbose -json
  sbom [options]          Supply Chain (SBOM): CycloneDX JSON, BLAKE3 per component
                          -config -vendor -target -out <path> -json
  verify [options]        Source tree BLAKE3 manifest integrity
                          -root <dir> -manifest <file> -update -json
  bench [options]         Nanosecond build phase profiler
                          -asm|-cc|-dir -out -mode -target -toolchain -n -json -verbose

Aegis technical (internal architecture):
  FileSystem VFS          internal/fs: Unix or Windows backend via build tags
                          OpenVerified: Lstat + SameFile TOCTOU hardening on reads
                          SecureWriteFile: temp 0600, close, atomic rename
  RunCommand              All subprocesses (git, ar, zig, fasm, gcc, ld, nasm, …)
                          exec.LookPath resolution, ValidateCLIArg per token,
                          deterministicEnv (LC_ALL=C, TZ=UTC, SOURCE_DATE_EPOCH)

Package Manager (fz pm):
  add <repo> [version]    Clone and add package to project
  remove <name>           Remove installed package
  list                    Show installed packages
  update                  Update all installed packages
  catalog                 List available packages from catalog
  search <keyword>        Search catalog
  install <name>          Install package from catalog (with hash verification)
`)
	return b.String()
}

func printHelp() {
	fmt.Fprint(os.Stderr, helpText())
}

func auditMain(args []string) {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	configPath := fs.String("config", "", "config file path")
	jsonOutput := fs.Bool("json", false, "machine-readable output")
	verbose := fs.Bool("verbose", false, "print verbose audit output")
	vendorDir := fs.String("vendor", "vendor", "vendor directory to scan")
	fs.Parse(args)
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit failed: %v\n", err)
		os.Exit(1)
	}
	utils.SetExecutionRoot(root)
	var cfg *config.Config
	if *configPath != "" {
		cfg, err = config.Load(*configPath)
	} else {
		cfg, err = config.LoadMerged("")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit failed: %v\n", err)
		os.Exit(1)
	}
	if cfg != nil {
		for k, v := range cfg.ToolChecksums {
			utils.ToolChecksums.Store(k, v)
		}
	}
	if *verbose {
		fmt.Fprintf(os.Stderr, "audit: scanning project root %s using vendor dir %s\n", root, *vendorDir)
	}
	result, err := audit.ScanProject(context.Background(), root, *vendorDir, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit failed: %v\n", err)
		os.Exit(1)
	}
	if len(result.Findings) == 0 {
		if *jsonOutput {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "clean", "findings": []any{}})
		} else {
			fmt.Println("audit passed: no vulnerabilities found")
		}
		return
	}
	if *jsonOutput {
		_ = json.NewEncoder(os.Stdout).Encode(result)
		return
	}
	for _, finding := range result.Findings {
		fmt.Printf("[%s] %s\n", finding.Package, finding.Summary)
		fmt.Printf("  path: %s\n", finding.Path)
		if finding.Version != "" {
			fmt.Printf("  version: %s\n", finding.Version)
		}
		if finding.URL != "" {
			fmt.Printf("  url: %s\n", finding.URL)
		}
	}
	os.Exit(1)
}

func sbomMain(args []string) {
	fs := flag.NewFlagSet("sbom", flag.ExitOnError)
	configPath := fs.String("config", "", "config file path")
	jsonOutput := fs.Bool("json", false, "machine-readable output")
	verbose := fs.Bool("verbose", false, "print verbose sbom generation output")
	vendorDir := fs.String("vendor", "vendor", "vendor directory to scan")
	outPath := fs.String("out", "sbom.json", "output SBOM file path")
	target := fs.String("target", "", "target triple to annotate in SBOM")
	fs.Parse(args)
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sbom failed: %v\n", err)
		os.Exit(1)
	}
	utils.SetExecutionRoot(root)
	var cfg *config.Config
	if *configPath != "" {
		cfg, err = config.Load(*configPath)
	} else {
		cfg, err = config.LoadMerged("")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "sbom failed: %v\n", err)
		os.Exit(1)
	}
	if cfg != nil {
		for k, v := range cfg.ToolChecksums {
			utils.ToolChecksums.Store(k, v)
		}
	}
	if *verbose {
		fmt.Fprintf(os.Stderr, "sbom: generating SBOM for project root %s using vendor dir %s\n", root, *vendorDir)
	}
	doc, err := sbom.Generate(root, *vendorDir, version, cfg, *target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sbom failed: %v\n", err)
		os.Exit(1)
	}
	data, err := sbom.Marshal(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sbom failed: %v\n", err)
		os.Exit(1)
	}
	if *jsonOutput {
		fmt.Println(string(data))
		return
	}
	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "sbom failed: %v\n", err)
		os.Exit(1)
	}
	if *verbose {
		fmt.Fprintf(os.Stderr, "sbom written to %s\n", *outPath)
	}
}

func doctorMain(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "machine-readable output")
	rootPath := fs.String("root", "", "project root (default: cwd)")
	fs.Parse(args)
	root := *rootPath
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "doctor failed: %v\n", err)
			os.Exit(1)
		}
		root = cwd
	}
	report, err := doctor.Run(context.Background(), doctor.Options{Root: root})
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctor failed: %v\n", err)
		os.Exit(1)
	}
	if *jsonOutput {
		data, merr := doctor.MarshalJSON(report)
		if merr != nil {
			fmt.Fprintf(os.Stderr, "doctor failed: %v\n", merr)
			os.Exit(1)
		}
		fmt.Println(string(data))
	} else {
		fmt.Print(doctor.FormatHuman(report))
	}
	if !report.Healthy {
		os.Exit(1)
	}
}

func verifyMain(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	rootPath := fs.String("root", ".", "project root directory")
	manifestPath := fs.String("manifest", "blake3.manifest", "manifest file path")
	updateManifest := fs.Bool("update", false, "update manifest file")
	jsonOutput := fs.Bool("json", false, "machine-readable output")
	fs.Parse(args)
	if err := utils.ValidateCLIPath(*rootPath); err != nil {
		fmt.Fprintf(os.Stderr, "verify failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(*manifestPath); err != nil {
		fmt.Fprintf(os.Stderr, "verify failed: %v\n", err)
		os.Exit(2)
	}
	root := filepath.Clean(*rootPath)
	manifest := filepath.Clean(*manifestPath)
	if *updateManifest {
		if err := verify.WriteManifest(manifest, root); err != nil {
			fmt.Fprintf(os.Stderr, "verify failed: %v\n", err)
			os.Exit(1)
		}
		if *jsonOutput {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "updated", "manifest": manifest})
			return
		}
		fmt.Printf("manifest updated: %s\n", manifest)
		return
	}
	result, err := verify.VerifyRoot(root, manifest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "verify failed: %v\n", err)
		os.Exit(1)
	}
	if len(result.Missing) == 0 && len(result.Modified) == 0 && len(result.Extra) == 0 {
		if *jsonOutput {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "clean"})
			return
		}
		fmt.Println("verify passed: source tree integrity intact")
		return
	}
	if *jsonOutput {
		_ = json.NewEncoder(os.Stdout).Encode(result)
		os.Exit(1)
	}
	if len(result.Missing) > 0 {
		fmt.Println("missing files:")
		for _, path := range result.Missing {
			fmt.Printf("  %s\n", path)
		}
	}
	if len(result.Modified) > 0 {
		fmt.Println("modified files:")
		for _, path := range result.Modified {
			fmt.Printf("  %s\n", path)
		}
	}
	if len(result.Extra) > 0 {
		fmt.Println("extra files:")
		for _, path := range result.Extra {
			fmt.Printf("  %s\n", path)
		}
	}
	os.Exit(1)
}

func benchMain(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	asmPath := fs.String("asm", "", "assembler source file")
	ccPath := fs.String("cc", "", "C source file")
	dirPath := fs.String("dir", "", "source directory")
	outBin := fs.String("out", "bench-out", "output binary path")
	mode := fs.String("mode", "auto", "link mode: auto, c, raw")
	target := fs.String("target", "x86_64-linux-gnu", "target triple")
	toolchain := fs.String("toolchain", "auto", "toolchain: auto or zig")
	jsonOutput := fs.Bool("json", false, "machine-readable output")
	verbose := fs.Bool("verbose", false, "verbose output")
	timeoutSec := fs.Int("timeout", 60, "timeout in seconds")
	jobs := fs.Int("j", 0, "parallel jobs")
	fs.Parse(args)
	if err := utils.ValidateCLIPath(*asmPath); err != nil {
		fmt.Fprintf(os.Stderr, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(*ccPath); err != nil {
		fmt.Fprintf(os.Stderr, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(*dirPath); err != nil {
		fmt.Fprintf(os.Stderr, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(*outBin); err != nil {
		fmt.Fprintf(os.Stderr, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIArg(*mode); err != nil {
		fmt.Fprintf(os.Stderr, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIArg(*target); err != nil {
		fmt.Fprintf(os.Stderr, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIArg(*toolchain); err != nil {
		fmt.Fprintf(os.Stderr, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if *mode != "auto" && *mode != "c" && *mode != "raw" {
		fmt.Fprintln(os.Stderr, "bench failed: invalid mode")
		os.Exit(2)
	}
	if *toolchain != "auto" && *toolchain != "zig" {
		fmt.Fprintln(os.Stderr, "bench failed: invalid toolchain")
		os.Exit(2)
	}
	if *jobs <= 0 {
		*jobs = runtime.NumCPU()
	}
	if *asmPath == "" && *ccPath == "" && *dirPath == "" {
		fmt.Fprintln(os.Stderr, "bench failed: missing source path")
		os.Exit(2)
	}
	if *asmPath != "" && *ccPath != "" {
		fmt.Fprintln(os.Stderr, "bench failed: specify only one of -asm or -cc")
		os.Exit(2)
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bench failed: %v\n", err)
		os.Exit(1)
	}
	utils.SetExecutionRoot(root)
	if *toolchain == "zig" {
		assembler.ZigRequested = true
		linker.ZigRequested = true
	}
	if utils.CheckTool("zig") == nil {
		assembler.ZigEnabled = true
		linker.ZigEnabled = true
	}
	if *dirPath != "" {
		*dirPath = filepath.Clean(*dirPath)
	}
	if *asmPath != "" {
		*asmPath = filepath.Clean(*asmPath)
	}
	if *ccPath != "" {
		*ccPath = filepath.Clean(*ccPath)
	}
	if *outBin != "" {
		*outBin = filepath.Clean(*outBin)
	}
	benchTimer := bench.NewTimer()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSec)*time.Second)
	defer cancel()
	if *asmPath != "" {
		objName := strings.TrimSuffix(filepath.Base(*asmPath), filepath.Ext(*asmPath)) + ".o"
		benchTimer.Stage("assemble", func() error {
			return assembler.Assemble(ctx, *asmPath, objName, false, *verbose, *mode)
		})
		benchTimer.Stage("link", func() error {
			return linker.Link(ctx, objName, *outBin, *verbose, *mode, false, true, false, nil)
		})
	} else if *ccPath != "" {
		objName := strings.TrimSuffix(filepath.Base(*ccPath), filepath.Ext(*ccPath)) + ".o"
		benchTimer.Stage("compile", func() error {
			return assembler.Assemble(ctx, *ccPath, objName, false, *verbose, *mode)
		})
		benchTimer.Stage("link", func() error {
			return linker.Link(ctx, objName, *outBin, *verbose, *mode, false, true, false, nil)
		})
	} else {
		benchTimer.Stage("build_directory", func() error {
			_, err := builder.BuildDir(ctx, []string{*dirPath}, *outBin, false, *verbose, *mode, false, false, false, true, false, nil, nil, nil, nil, nil, *jobs, "executable")
			return err
		})
	}
	err = benchTimer.Error()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bench failed: %v\n", err)
		os.Exit(1)
	}
	if *jsonOutput {
		data, _ := benchTimer.JSON()
		fmt.Println(string(data))
		return
	}
	fmt.Print(benchTimer.Report())
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "audit":
			auditMain(os.Args[2:])
			return
		case "sbom":
			sbomMain(os.Args[2:])
			return
		case "verify":
			verifyMain(os.Args[2:])
			return
		case "bench":
			benchMain(os.Args[2:])
			return
		case "doctor":
			doctorMain(os.Args[2:])
			return
		case "version":
			outputVersion()
			return
		}
	}
	var (
		asmPath            string
		ccPath             string
		dirPath            string
		debug              bool
		verbose            bool
		outBin             string
		outObj             string
		timeoutSec         int
		mode               string
		keepObj            bool
		clean              bool
		noCache            bool
		configPath         string
		noSymbolCheck      bool
		watch              bool
		sanitize           bool
		noSanitize         bool
		strict             bool
		jsonOutput         bool
		showVersion        bool
		showHelp           bool
		showMan            bool
		format             string
		initMode           bool
		ldScript           string
		textAddr           string
		shellMode          bool
		jobs               int
		updateMode         bool
		buildType          string
		libMode            bool
		target             string
		toolchain          string
		genCompileCommands bool
		shared             bool
		ccFlags            string
		ldFlags            string
		forceFASM          bool
	)

	flag.StringVar(&asmPath, "asm", "", "")
	flag.StringVar(&asmPath, "assembler", "", "")
	flag.StringVar(&ccPath, "cc", "", "")
	flag.StringVar(&dirPath, "dir", "", "")
	flag.BoolVar(&debug, "debug", false, "")
	flag.BoolVar(&verbose, "verbose", false, "")
	flag.StringVar(&outBin, "out", "", "")
	flag.StringVar(&outObj, "out-obj", "", "")
	flag.IntVar(&timeoutSec, "timeout", 60, "")
	flag.StringVar(&mode, "mode", "", "")
	flag.BoolVar(&keepObj, "keep-obj", false, "")
	flag.BoolVar(&clean, "clean", false, "")
	flag.BoolVar(&noCache, "no-cache", false, "")
	flag.BoolVar(&noSymbolCheck, "no-symbol-check", false, "")
	flag.StringVar(&configPath, "config", "", "")
	flag.BoolVar(&watch, "watch", false, "")
	flag.BoolVar(&sanitize, "sanitize", true, "")
	flag.BoolVar(&noSanitize, "no-sanitize", false, "")
	flag.BoolVar(&strict, "strict", false, "")
	flag.BoolVar(&jsonOutput, "json", false, "")
	flag.BoolVar(&showVersion, "v", false, "")
	flag.BoolVar(&showVersion, "version", false, "")
	flag.BoolVar(&showHelp, "h", false, "")
	flag.BoolVar(&showHelp, "help", false, "")
	flag.BoolVar(&showMan, "man", false, "")
	flag.StringVar(&format, "format", "elf64", "")
	flag.BoolVar(&initMode, "init", false, "initialize project: create .fz.yaml and .fzignore")
	flag.StringVar(&ldScript, "T", "", "linker script file (passed to ld via -T)")
	flag.StringVar(&textAddr, "Ttext", "", "set text segment address (passed to ld)")
	flag.BoolVar(&shellMode, "shell", false, "run interactive shell")
	flag.IntVar(&jobs, "j", 1, "number of parallel jobs (0 = auto = CPU cores)")
	flag.BoolVar(&updateMode, "update", false, "update fz to the latest version")
	flag.StringVar(&buildType, "type", "executable", "build type: executable (default) or static")
	flag.BoolVar(&libMode, "lib", false, "build static library (archive)")
	flag.StringVar(&target, "target", "x86_64-linux-gnu", "target triple (e.g., x86_64-linux-gnu, arm-linux-gnueabihf, riscv64-unknown-elf)")
	flag.StringVar(&toolchain, "toolchain", "auto", "toolchain to use: auto or zig")
	flag.BoolVar(&genCompileCommands, "compile-commands", false, "generate compile_commands.json for LSP and exit")
	flag.BoolVar(&shared, "shared", false, "build shared library instead of executable")
	flag.StringVar(&ccFlags, "cc-flag", "", "additional C compiler flags (space-separated)")
	flag.StringVar(&ldFlags, "ld-flag", "", "additional linker flags (space-separated)")
	flag.BoolVar(&forceFASM, "fasm", false, "use FASM instead of NASM for .asm files")
	flag.Usage = printHelp
	flag.Parse()

	if err := utils.ValidateCLIPath(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config path: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(asmPath); err != nil {
		fmt.Fprintf(os.Stderr, "invalid asm path: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(ccPath); err != nil {
		fmt.Fprintf(os.Stderr, "invalid cc path: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(dirPath); err != nil {
		fmt.Fprintf(os.Stderr, "invalid dir path: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(outBin); err != nil {
		fmt.Fprintf(os.Stderr, "invalid output path: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(outObj); err != nil {
		fmt.Fprintf(os.Stderr, "invalid object output path: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(ldScript); err != nil {
		fmt.Fprintf(os.Stderr, "invalid linker script path: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(textAddr); err != nil {
		fmt.Fprintf(os.Stderr, "invalid text address: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIArg(mode); err != nil {
		fmt.Fprintf(os.Stderr, "invalid mode: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIArg(format); err != nil {
		fmt.Fprintf(os.Stderr, "invalid format: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIArg(target); err != nil {
		fmt.Fprintf(os.Stderr, "invalid target: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIArg(toolchain); err != nil {
		fmt.Fprintf(os.Stderr, "invalid toolchain: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIArg(buildType); err != nil {
		fmt.Fprintf(os.Stderr, "invalid build type: %v\n", err)
		os.Exit(2)
	}
	if _, err := utils.ValidateFlagTokens([]byte(ccFlags)); err != nil {
		fmt.Fprintf(os.Stderr, "invalid C compiler flags: %v\n", err)
		os.Exit(2)
	}
	if _, err := utils.ValidateFlagTokens([]byte(ldFlags)); err != nil {
		fmt.Fprintf(os.Stderr, "invalid linker flags: %v\n", err)
		os.Exit(2)
	}
	if mode != "" && mode != "auto" && mode != "c" && mode != "raw" {
		fmt.Fprintln(os.Stderr, "error: -mode must be auto, c, or raw")
		os.Exit(2)
	}
	if toolchain != "" && toolchain != "auto" && toolchain != "zig" {
		fmt.Fprintln(os.Stderr, "error: -toolchain must be auto or zig")
		os.Exit(2)
	}

	if initMode {
		if err := initpkg.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "init failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("project initialized. edit .fz.yaml to configure ur build.")
		return
	}

	assembler.ForceFASM = forceFASM

	ctx := context.Background()
	if len(os.Args) >= 2 && os.Args[1] == "pm" {
		if len(os.Args) < 3 {
			fmt.Println("Usage: fz pm <add|remove|list|update|catalog|search|install> [args]")
			return
		}
		subcmd := os.Args[2]
		switch subcmd {
		case "add":
			if len(os.Args) < 4 {
				fmt.Println("Usage: fz pm add <repo-url> [version]")
				return
			}
			pkgURL := os.Args[3]
			ver := ""
			if len(os.Args) > 4 {
				ver = os.Args[4]
			}
			if err := pkgman.Add(ctx, pkgURL, ver); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		case "remove":
			if len(os.Args) < 4 {
				fmt.Println("Usage: fz pm remove <repo-url>")
				return
			}
			if err := pkgman.Remove(ctx, os.Args[3]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		case "list":
			if len(os.Args) == 3 {
				pkgman.List()
			} else if os.Args[3] == "catalog" {
				pkgman.ListCatalog(ctx)
			} else {
				fmt.Println("Usage: fz pm list [catalog]")
			}
		case "update":
			if err := pkgman.Update(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		case "catalog":
			pkgman.ListCatalog(ctx)
		case "search":
			if len(os.Args) < 4 {
				fmt.Println("Usage: fz pm search <keyword>")
				return
			}
			if err := pkgman.SearchCatalog(ctx, os.Args[3]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		case "install":
			if len(os.Args) < 4 {
				fmt.Println("Usage: fz pm install <catalog-package-name>")
				return
			}
			if err := pkgman.InstallFromCatalog(ctx, os.Args[3]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Printf("Unknown pm subcommand: %s\n", subcmd)
		}
		return
	}

	assembler.CcFlags = ccFlags
	linker.LdFlags = ldFlags
	linker.Shared = shared
	assembler.Target = target
	linker.Target = target
	if libMode {
		buildType = "static"
	}
	if buildType != "executable" && buildType != "static" {
		fmt.Fprintf(os.Stderr, "error: -type must be executable or static")
		os.Exit(2)
	}
	if updateMode {
		if err := updater.UpdateSelf(version); err != nil {
			fmt.Fprintf(os.Stderr, "update failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if jobs <= 0 {
		jobs = runtime.NumCPU()
	}
	if shellMode {
		shell.Run()
		return
	}
	if showMan {
		fmt.Print(man.GenerateManPage(version))
		os.Exit(0)
	}
	if showHelp {
		printHelp()
		os.Exit(0)
	}
	if showVersion {
		if jsonOutput {
			report := BuildReport{Status: "info", ExitCode: 0, DurationMs: 0, Binary: version}
			_ = json.NewEncoder(os.Stdout).Encode(report)
		} else {
			outputVersion()
		}
		os.Exit(0)
	}
	if err := linker.SetOutputFormat(format); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	linker.LdScript = ldScript
	linker.TextAddr = textAddr
	if mode == "" {
		mode = "auto"
	}
	if noSanitize {
		sanitize = false
	}
	root, err := os.Getwd()
	if err == nil {
		utils.SetExecutionRoot(root)
	}
	if watch && jsonOutput {
		fmt.Fprintln(os.Stderr, "error: -watch and -json cannot be used together")
		os.Exit(2)
	}
	srcProvided := 0
	if asmPath != "" {
		srcProvided++
	}
	if ccPath != "" {
		srcProvided++
	}
	if dirPath != "" {
		srcProvided++
	}
	if srcProvided > 1 {
		errMsg := "specify only one of -asm, -cc, or -dir"
		if jsonOutput {
			report := BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: errMsg}
			_ = json.NewEncoder(os.Stdout).Encode(report)
		} else {
			fmt.Fprintln(os.Stderr, errMsg)
		}
		os.Exit(2)
	}
	srcPath := asmPath
	if ccPath != "" {
		srcPath = ccPath
	}

	var cfg *config.Config
	if configPath != "" {
		cfg, err = config.Load(configPath)
		if err != nil {
			if jsonOutput {
				report := BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: err.Error()}
				_ = json.NewEncoder(os.Stdout).Encode(report)
			} else {
				fmt.Fprintf(os.Stderr, "config error: %v\n", err)
			}
			os.Exit(2)
		}
	} else {
		cfg, err = config.LoadMerged("")
	}
	if err != nil {
		if jsonOutput {
			report := BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: err.Error()}
			_ = json.NewEncoder(os.Stdout).Encode(report)
		} else {
			fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		}
		os.Exit(2)
	}
	if cfg != nil {
		for k, v := range cfg.ToolChecksums {
			utils.ToolChecksums.Store(k, v)
		}
		cfg.MergeFromFlags(srcPath, dirPath, outBin, outObj, debug, verbose, keepObj, noCache, mode, toolchain)
		if verbose && !jsonOutput {
			fmt.Printf("Loaded config from %s\n", func() string {
				if configPath != "" {
					return configPath
				}
				return config.DefaultConfigPath()
			}())
		}
		if len(cfg.Flags.Asm) > 0 {
			assembler.AsmFlags = cfg.Flags.Asm
		}
		if len(cfg.Flags.Cc) > 0 {
			assembler.CcFlags = strings.Join(cfg.Flags.Cc, " ")
		}
		if len(cfg.Flags.Ld) > 0 {
			linker.LdFlags = strings.Join(cfg.Flags.Ld, " ")
		}
		if cfg.Toolchain == "zig" || toolchain == "zig" {
			assembler.ZigRequested = true
			linker.ZigRequested = true
		}
	}
	if utils.CheckTool("zig") == nil {
		assembler.ZigEnabled = true
		linker.ZigEnabled = true
	}

	if genCompileCommands {
		dirs := []string{"."}
		if cfg != nil && len(cfg.SourceDirs) > 0 {
			dirs = cfg.SourceDirs
		} else if cfg != nil && cfg.SourceDir != "" {
			dirs = []string{cfg.SourceDir}
		}
		if err := compilecommands.Generate(cfg, dirs[0]); err != nil {
			fmt.Fprintf(os.Stderr, "error generating compile_commands.json: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("compile_commands.json generated")
		return
	}

	if clean {
		targetDir := dirPath
		if targetDir == "" && cfg != nil && cfg.SourceDir != "" {
			targetDir = cfg.SourceDir
		}
		if targetDir == "" && cfg != nil && len(cfg.SourceDirs) > 0 {
			targetDir = cfg.SourceDirs[0]
		}
		if targetDir == "" {
			errMsg := "-clean requires -dir or source_dir/source_dirs in config"
			if jsonOutput {
				report := BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: errMsg}
				_ = json.NewEncoder(os.Stdout).Encode(report)
			} else {
				fmt.Fprintln(os.Stderr, errMsg)
			}
			os.Exit(2)
		}
		if err := builder.CleanDir(targetDir, verbose); err != nil {
			if jsonOutput {
				report := BuildReport{Status: "error", ExitCode: 1, DurationMs: 0, Error: err.Error()}
				_ = json.NewEncoder(os.Stdout).Encode(report)
			} else {
				fmt.Fprintf(os.Stderr, "clean failed: %v\n", err)
			}
			os.Exit(1)
		}
		if jsonOutput {
			report := BuildReport{Status: "success", ExitCode: 0, DurationMs: 0, Binary: "cleaned"}
			_ = json.NewEncoder(os.Stdout).Encode(report)
		} else {
			fmt.Printf("Cleaned %s\n", targetDir)
		}
		return
	}

	if cfg != nil {
		if srcPath == "" && dirPath == "" {
			if cfg.SourceFile != "" {
				srcPath = cfg.SourceFile
			}
			if cfg.SourceDir != "" {
				dirPath = cfg.SourceDir
			}
			if len(cfg.SourceDirs) > 0 {
				dirPath = "dummy"
			}
		}
		debug = cfg.Debug
		verbose = cfg.Verbose
		if cfg.Output != "" {
			outBin = cfg.Output
		}
		if cfg.OutObj != "" {
			outObj = cfg.OutObj
		}
		if cfg.Mode != "" {
			mode = cfg.Mode
		}
		if cfg.KeepObj {
			keepObj = true
		}
		if cfg.NoCache {
			noCache = true
		}
	}

	if srcPath == "" && dirPath == "" {
		errMsg := "missing source: use -asm, -cc, -dir, or config"
		if jsonOutput {
			report := BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: errMsg}
			_ = json.NewEncoder(os.Stdout).Encode(report)
		} else {
			fmt.Fprintln(os.Stderr, errMsg)
		}
		os.Exit(2)
	}
	if srcPath != "" && dirPath != "" {
		errMsg := "cannot specify both single file and -dir"
		if jsonOutput {
			report := BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: errMsg}
			_ = json.NewEncoder(os.Stdout).Encode(report)
		} else {
			fmt.Fprintln(os.Stderr, errMsg)
		}
		os.Exit(2)
	}

	assembler.OutputFormat = format

	startTime := time.Now()
	var sourceFiles []string
	var objectFiles []string
	var finalBinary string
	var buildErr error

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	build := func() error {
		if srcPath != "" {
			sourceFiles = append(sourceFiles, srcPath)
			if err := utils.CheckFileExists(srcPath); err != nil {
				return err
			}
			ext := filepath.Ext(srcPath)
			if !utils.SupportedExtension(ext) {
				return fmt.Errorf("unsupported extension: %s", ext)
			}
			binName, objName := utils.DeriveNames(srcPath, outBin, outObj)
			objectFiles = append(objectFiles, objName)
			finalBinary = binName
			if verbose && !jsonOutput {
				if ext == ".c" || ext == ".cpp" || ext == ".cc" || ext == ".cxx" {
					fmt.Printf("Compiling %s -> %s\n", srcPath, objName)
				} else {
					fmt.Printf("Assembling %s -> %s\n", srcPath, objName)
				}
			}
			if err := assembler.Assemble(ctx, srcPath, objName, debug, verbose, mode); err != nil {
				return err
			}
			if format == "bin" {
				if err := utils.CopyFile(objName, binName); err != nil {
					return err
				}
				if !jsonOutput {
					fmt.Printf("Built: %s\n", binName)
				}
				return nil
			}
			if verbose && !jsonOutput {
				fmt.Printf("Linking %s -> %s (mode: %s)\n", objName, binName, mode)
			}
			if err := linker.Link(ctx, objName, binName, verbose, mode, noSymbolCheck, sanitize, strict, nil); err != nil {
				return err
			}
			if !jsonOutput {
				fmt.Printf("Built: %s\n", binName)
			}
			return nil
		}
		if dirPath != "" || (cfg != nil && len(cfg.SourceDirs) > 0) {
			if format == "bin" {
				return fmt.Errorf("-format bin is not supported for directory builds")
			}
			var dirs []string
			if cfg != nil && len(cfg.SourceDirs) > 0 {
				dirs = cfg.SourceDirs
			} else {
				if dirPath == "" {
					dirPath = "."
				}
				dirs = []string{dirPath}
			}
			for _, d := range dirs {
				info, err := os.Stat(d)
				if err != nil {
					return err
				}
				if !info.IsDir() {
					return fmt.Errorf("%s is not a directory", d)
				}
			}
			if outBin != "" {
				if st, err := os.Stat(outBin); err == nil && st.IsDir() {
					return fmt.Errorf("output path %s is a directory", outBin)
				}
			}
			var exclude []string
			if cfg != nil {
				exclude = cfg.Exclude
			}
			var ignoreMatcher *ignore.IgnoreMatcher
			if cfg != nil && cfg.IgnoreFile != "" {
				if _, err := os.Stat(cfg.IgnoreFile); err == nil {
					if ignoreMatcher, err = ignore.LoadIgnoreFile(cfg.IgnoreFile); err != nil && verbose {
						fmt.Printf("warning: cannot load ignore file %s: %v\n", cfg.IgnoreFile, err)
					}
				}
			}
			var includes []string
			if cfg != nil {
				includes = cfg.Include
			}
			var sourceFilesList []string
			if cfg != nil {
				sourceFilesList = cfg.SourceFiles
			}
			var libs []string
			if cfg != nil {
				libs = cfg.Libs
			}
			res, err := builder.BuildDir(ctx, dirs, outBin, debug, verbose, mode, keepObj, noCache, noSymbolCheck, sanitize, strict, exclude, sourceFilesList, ignoreMatcher, includes, libs, jobs, buildType)
			if err != nil {
				return err
			}
			objectFiles = res.ObjectFiles
			finalBinary = res.Binary
			if !jsonOutput {
				if !keepObj && verbose {
					fmt.Printf("Removed object dir: %s\n", res.ObjDir)
				}
				fmt.Printf("Built: %s\n", res.Binary)
			}
			return nil
		}
		return fmt.Errorf("no source to build")
	}

	buildErr = build()
	durationMs := time.Since(startTime).Milliseconds()

	if buildErr != nil {
		if jsonOutput {
			report := BuildReport{
				Status:      "error",
				ExitCode:    1,
				DurationMs:  durationMs,
				Binary:      finalBinary,
				SourceFiles: sourceFiles,
				ObjectFiles: objectFiles,
				Error:       buildErr.Error(),
			}
			_ = json.NewEncoder(os.Stdout).Encode(report)
		} else {
			fmt.Fprintf(os.Stderr, "build failed: %v\n", buildErr)
		}
		if !watch {
			os.Exit(1)
		}
	} else if jsonOutput {
		report := BuildReport{
			Status:      "success",
			ExitCode:    0,
			DurationMs:  durationMs,
			Binary:      finalBinary,
			SourceFiles: sourceFiles,
			ObjectFiles: objectFiles,
		}
		_ = json.NewEncoder(os.Stdout).Encode(report)
	}

	if watch {
		w, err := watcher.New()
		if err != nil {
			if jsonOutput {
				report := BuildReport{Status: "error", ExitCode: 1, DurationMs: 0, Error: err.Error()}
				_ = json.NewEncoder(os.Stdout).Encode(report)
			} else {
				fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
			}
			os.Exit(1)
		}
		defer w.Close()
		watchTarget := dirPath
		if srcPath != "" {
			watchTarget = filepath.Dir(srcPath)
		}
		if watchTarget == "" {
			if cfg != nil && len(cfg.SourceDirs) > 0 {
				watchTarget = cfg.SourceDirs[0]
			} else {
				watchTarget = "."
			}
		}
		if err := w.AddRecursive(watchTarget); err != nil {
			if jsonOutput {
				report := BuildReport{Status: "error", ExitCode: 1, DurationMs: 0, Error: err.Error()}
				_ = json.NewEncoder(os.Stdout).Encode(report)
			} else {
				fmt.Fprintf(os.Stderr, "cannot watch: %v\n", err)
			}
			os.Exit(1)
		}
		if configPath != "" {
			_ = w.Add(configPath)
		} else if cfgFile := config.DefaultConfigPath(); cfgFile != "" {
			_ = w.Add(cfgFile)
		}
		if !jsonOutput {
			fmt.Printf("Watching %s for changes...\n", watchTarget)
		}
		w.Watch(500*time.Millisecond, func(string) error {
			if !jsonOutput {
				fmt.Println("\nChange detected, rebuilding...")
			}
			ctx2, cancel2 := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
			defer cancel2()
			origCtx := ctx
			ctx = ctx2
			err := build()
			ctx = origCtx
			if err != nil {
				if !jsonOutput {
					fmt.Fprintf(os.Stderr, "rebuild failed: %v\n", err)
				}
			} else if !jsonOutput {
				fmt.Println("Rebuild successful.")
			}
			return nil
		})
		select {}
	}
}
