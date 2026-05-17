package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"fz/internal/assembler"
	"fz/internal/builder"
	"fz/internal/config"
	"fz/internal/ignore"
	initpkg "fz/internal/init"
	"fz/internal/linker"
	"fz/internal/man"
	"fz/internal/shell"
	"fz/internal/utils"
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

var version = "1.7.0"

func printHelp() {
	fmt.Fprintf(os.Stderr, `
fz – assembly & C build tool

Usage:
  fz [options] (-asm <file> | -cc <file> | -dir <dir> | (no args with config))

Options:
  -asm <file>            Assembler source (.asm, .s, .S, .fasm)
  -cc <file>             C source (compiled with -Wall -Wextra -Werror -Wpedantic -Wshadow -Wconversion)
  -dir <dir>             Build all supported files in directory (recursive)
  -out <name>            Output binary name
  -out-obj <name>        Object file name (single file only)
  -mode <auto|c|raw>     Linking mode (default: auto)
  -debug                 Emit debug information (-g)
  -verbose               Print executed commands
  -keep-obj              Keep temporary object files when using -dir
  -no-cache              Disable incremental cache
  -no-symbol-check       Skip duplicate symbol pre‑check
  -sanitize              Enable sanitizers for C (default: true)
  -no-sanitize           Disable sanitizers
  -strict                Enable aggressive sanitizers (use-after-return, use-after-scope) – prefers clang
  -clean                 Remove all build artifacts (.fz_objs, .fz_cache, binaries)
  -watch                 Watch source files and rebuild automatically
  -json                  Output build report in JSON format (CI/CD)
  -config <file>         Config file path (default: .fz.yaml, fz.yaml, .fz.yml, fz.yml)
  -man                   Generate roff man page and exit
  -format <elf|bin>      Output format: elf (default) or bin (flat binary, no linking)
  -h, --help             Show this help
  -v, --version          Show version
-j <n>                 Number of parallel jobs (0 = auto = CPU cores)

Examples:
  fz -asm boot.asm
  fz -cc main.c -strict -verbose
  fz -dir ./src -out myapp -watch
  fz -json -cc test.c
  fz -dir . -clean
  fz -asm boot.asm -format bin -out boot.bin

Supported extensions: .asm, .s, .S, .fasm, .c
`)
}

func main() {
	var (
		asmPath       string
		ccPath        string
		dirPath       string
		debug         bool
		verbose       bool
		outBin        string
		outObj        string
		timeoutSec    int
		mode          string
		keepObj       bool
		clean         bool
		noCache       bool
		configPath    string
		noSymbolCheck bool
		watch         bool
		sanitize      bool
		noSanitize    bool
		strict        bool
		jsonOutput    bool
		showVersion   bool
		showHelp      bool
		showMan       bool
		format        string
		initMode      bool
		ldScript      string
		textAddr      string
		shellMode     bool
		jobs          int
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

	flag.Usage = printHelp
	flag.Parse()
	if initMode {
		if err := initpkg.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "init failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("project initialized. edit .fz.yaml to configure ur build.")
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
			json.NewEncoder(os.Stdout).Encode(report)
		} else {
			fmt.Printf("fz version %s\n", version)
		}
		os.Exit(0)
	}
	if err := linker.SetOutputFormat(format); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	linker.LdScript = ldScript
	linker.TextAddr = textAddr

	if format != "elf32" && format != "elf64" && format != "bin" {
		fmt.Fprintln(os.Stderr, "error: -format must be elf or bin")
		os.Exit(2)
	}

	if mode == "" {
		mode = "auto"
	}
	if noSanitize {
		sanitize = false
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
			json.NewEncoder(os.Stdout).Encode(report)
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
	var err error
	if configPath != "" {
		cfg, err = config.Load(configPath)
	} else {
		cfg, err = config.LoadMerged("")
	}
	if err != nil {
		if jsonOutput {
			report := BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: err.Error()}
			json.NewEncoder(os.Stdout).Encode(report)
		} else {
			fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		}
		os.Exit(2)
	}
	if cfg != nil {
		cfg.MergeFromFlags(srcPath, dirPath, outBin, outObj, debug, verbose, keepObj, noCache, mode)
		if verbose && !jsonOutput {
			fmt.Printf("Loaded config from %s\n", func() string {
				if configPath != "" {
					return configPath
				}
				return config.DefaultConfigPath()
			}())
		}
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
				json.NewEncoder(os.Stdout).Encode(report)
			} else {
				fmt.Fprintln(os.Stderr, errMsg)
			}
			os.Exit(2)
		}
		if err := builder.CleanDir(targetDir, verbose); err != nil {
			if jsonOutput {
				report := BuildReport{Status: "error", ExitCode: 1, DurationMs: 0, Error: err.Error()}
				json.NewEncoder(os.Stdout).Encode(report)
			} else {
				fmt.Fprintf(os.Stderr, "clean failed: %v\n", err)
			}
			os.Exit(1)
		}
		if jsonOutput {
			report := BuildReport{Status: "success", ExitCode: 0, DurationMs: 0, Binary: "cleaned"}
			json.NewEncoder(os.Stdout).Encode(report)
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
			json.NewEncoder(os.Stdout).Encode(report)
		} else {
			fmt.Fprintln(os.Stderr, errMsg)
		}
		os.Exit(2)
	}
	if srcPath != "" && dirPath != "" {
		errMsg := "cannot specify both single file and -dir"
		if jsonOutput {
			report := BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: errMsg}
			json.NewEncoder(os.Stdout).Encode(report)
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
				if ext == ".c" {
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
			var err error
			if cfg != nil && cfg.IgnoreFile != "" {
				if _, err := os.Stat(cfg.IgnoreFile); err == nil {
					if ignoreMatcher, err = ignore.LoadIgnoreFile(cfg.IgnoreFile); err != nil {
						if verbose {
							fmt.Printf("warning: cannot load ignore file %s: %v\n", cfg.IgnoreFile, err)
						}
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
			res, err := builder.BuildDir(ctx, dirs, outBin, debug, verbose, mode, keepObj, noCache, noSymbolCheck, sanitize, strict, exclude, sourceFilesList, ignoreMatcher, includes, libs, jobs)
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
			json.NewEncoder(os.Stdout).Encode(report)
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
		json.NewEncoder(os.Stdout).Encode(report)
	}

	if watch {
		w, err := watcher.New()
		if err != nil {
			if jsonOutput {
				report := BuildReport{Status: "error", ExitCode: 1, DurationMs: 0, Error: err.Error()}
				json.NewEncoder(os.Stdout).Encode(report)
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
				json.NewEncoder(os.Stdout).Encode(report)
			} else {
				fmt.Fprintf(os.Stderr, "cannot watch: %v\n", err)
			}
			os.Exit(1)
		}
		if configPath != "" {
			w.Add(configPath)
		} else if cfgFile := config.DefaultConfigPath(); cfgFile != "" {
			w.Add(cfgFile)
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
