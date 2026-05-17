package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fz/internal/assembler"
	"fz/internal/builder"
	"fz/internal/config"
	"fz/internal/linker"
	"fz/internal/man"
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

var version = "1.3.0"

func printHelp() {
	cyan := "\033[36m"
	green := "\033[32m"
	blue := "\033[34m"
	bold := "\033[1m"
	reset := "\033[0m"

	fmt.Fprintln(os.Stderr, "\n"+cyan+bold+"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"+reset)
	fmt.Fprintln(os.Stderr, cyan+bold+"  fz - assembly swiss army knife "+reset)
	fmt.Fprintln(os.Stderr, cyan+bold+"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"+reset+"\n")

	fmt.Fprintln(os.Stderr, bold+"Usage:"+reset)
	fmt.Fprintln(os.Stderr, "  "+green+"fz [options] (-asm <file> | -cc <file> | -dir <dir>)"+reset+"\n")

	fmt.Fprintln(os.Stderr, blue+bold+"Build Source:"+reset)
	fmt.Fprintln(os.Stderr, "  "+cyan+"-asm, --assembler <file>     "+reset+"Assembler source (.asm, .s, .S, .fasm)")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-cc <file>                   "+reset+"C source (strict warnings enabled)")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-dir <directory>             "+reset+"Build all supported files recursively\n")

	fmt.Fprintln(os.Stderr, blue+bold+"Output Control:"+reset)
	fmt.Fprintln(os.Stderr, "  "+cyan+"-out <name>                  "+reset+"Output binary name")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-out-obj <name>              "+reset+"Object file name (single file only)")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-keep-obj                    "+reset+"Keep temporary object files (when using -dir)\n")

	fmt.Fprintln(os.Stderr, blue+bold+"Linking Mode:"+reset)
	fmt.Fprintln(os.Stderr, "  "+cyan+"-mode <auto|c|raw>           "+reset+"Linking mode (default: auto)\n")

	fmt.Fprintln(os.Stderr, blue+bold+"C‑specific & Sanitizers:"+reset)
	fmt.Fprintln(os.Stderr, "  "+cyan+"-sanitize                    "+reset+"Enable ASan + UBSan (default: on)")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-no-sanitize                 "+reset+"Disable sanitizers")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-strict                      "+reset+"Use clang + advanced sanitizers (use-after-return, etc.)\n")

	fmt.Fprintln(os.Stderr, blue+bold+"Debugging & Verbosity:"+reset)
	fmt.Fprintln(os.Stderr, "  "+cyan+"-debug                       "+reset+"Emit debug symbols (-g)")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-verbose                     "+reset+"Print every command executed\n")

	fmt.Fprintln(os.Stderr, blue+bold+"Performance / Cache:"+reset)
	fmt.Fprintln(os.Stderr, "  "+cyan+"-no-cache                    "+reset+"Disable incremental cache")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-no-symbol-check             "+reset+"Skip duplicate symbol pre‑check\n")

	fmt.Fprintln(os.Stderr, blue+bold+"Other:"+reset)
	fmt.Fprintln(os.Stderr, "  "+cyan+"-clean                       "+reset+"Remove all build artifacts (.fz_objs, .fz_cache, binaries)")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-watch                       "+reset+"Watch files and auto‑rebuild")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-json                        "+reset+"Output build report in JSON (for CI/CD)")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-config <file>               "+reset+"Config file (default: .fz.yaml, fz.yaml, ...)")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-man                         "+reset+"Generate roff man page and exit\n")

	fmt.Fprintln(os.Stderr, blue+bold+"Info:"+reset)
	fmt.Fprintln(os.Stderr, "  "+cyan+"-h, --help                   "+reset+"Show this help")
	fmt.Fprintln(os.Stderr, "  "+cyan+"-v, --version                "+reset+"Show version\n")

	fmt.Fprintln(os.Stderr, blue+bold+"Examples:"+reset)
	fmt.Fprintln(os.Stderr, "  "+green+"  fz -asm boot.asm"+reset)
	fmt.Fprintln(os.Stderr, "  "+green+"  fz -cc main.c -strict -verbose"+reset)
	fmt.Fprintln(os.Stderr, "  "+green+"  fz -dir ./src -out myapp -watch"+reset)
	fmt.Fprintln(os.Stderr, "  "+green+"  fz -json -cc test.c"+reset)
	fmt.Fprintln(os.Stderr, "  "+green+"  fz -dir . -clean"+reset+"\n")

	fmt.Fprintln(os.Stderr, blue+bold+"Supported extensions:"+reset+" "+cyan+".asm, .s, .S, .fasm, .c"+reset+"\n")
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

	flag.Usage = printHelp
	flag.Parse()

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

	cfgFile := configPath
	if cfgFile == "" {
		cfgFile = config.DefaultConfigPath()
	}
	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			if jsonOutput {
				report := BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: err.Error()}
				json.NewEncoder(os.Stdout).Encode(report)
			} else {
				fmt.Fprintf(os.Stderr, "config error: %v\n", err)
			}
			os.Exit(2)
		}
	} else {
		var err error
		cfg, err = config.LoadMerged("")
		if err != nil {
			if jsonOutput {
				report := BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: err.Error()}
				json.NewEncoder(os.Stdout).Encode(report)
			} else {
				fmt.Fprintf(os.Stderr, "config error: %v\n", err)
			}
			os.Exit(2)
		}
	}
	if cfg != nil {
		cfg.MergeFromFlags(srcPath, dirPath, outBin, outObj, debug, verbose, keepObj, noCache, mode)
		if verbose && !jsonOutput && (configPath != "" || cfgFile != "") {
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
		if targetDir == "" {
			errMsg := "-clean requires -dir or source_dir in config"
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
		srcPath = cfg.SourceFile
		dirPath = cfg.SourceDir
		debug = cfg.Debug
		verbose = cfg.Verbose
		outBin = cfg.Output
		outObj = cfg.OutObj
		if cfg.Mode != "" {
			mode = cfg.Mode
		}
		keepObj = cfg.KeepObj
		noCache = cfg.NoCache
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
			if verbose && !jsonOutput {
				fmt.Printf("Linking %s -> %s (mode: %s)\n", objName, binName, mode)
			}
			if err := linker.Link(ctx, objName, binName, verbose, mode, noSymbolCheck, sanitize, strict); err != nil {
				return err
			}
			if !jsonOutput {
				fmt.Printf("Built: %s\n", binName)
			}
			return nil
		}
		if dirPath != "" {
			info, err := os.Stat(dirPath)
			if err != nil {
				return err
			}
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", dirPath)
			}
			if outBin != "" {
				if st, err := os.Stat(outBin); err == nil && st.IsDir() {
					return fmt.Errorf("output path %s is a directory", outBin)
				}
			}
			res, err := builder.BuildDir(ctx, dirPath, outBin, debug, verbose, mode, keepObj, noCache, noSymbolCheck, sanitize, strict)
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
			watchTarget = "."
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
		if cfgFile != "" {
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
