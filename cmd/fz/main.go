package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fz/internal/assembler"
	"fz/internal/builder"
	"fz/internal/config"
	"fz/internal/linker"
	"fz/internal/utils"
	"fz/internal/watcher"
)

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
	)

	flag.StringVar(&asmPath, "asm", "", "assembler source file (.asm, .s, .S, .fasm)")
	flag.StringVar(&asmPath, "assembler", "", "alias for -asm")
	flag.StringVar(&ccPath, "cc", "", "C source file (compiles with -Wall -Wextra -Werror -Wpedantic -Wshadow -Wconversion)")
	flag.StringVar(&dirPath, "dir", "", "directory containing source files (recursive)")
	flag.BoolVar(&debug, "debug", false, "emit debug information")
	flag.BoolVar(&verbose, "verbose", false, "print executed commands")
	flag.StringVar(&outBin, "out", "", "output binary name")
	flag.StringVar(&outObj, "out-obj", "", "output object file name (only with single file)")
	flag.IntVar(&timeoutSec, "timeout", 60, "timeout in seconds for external commands")
	flag.StringVar(&mode, "mode", "", "linking mode: auto, c, raw")
	flag.BoolVar(&keepObj, "keep-obj", false, "keep temporary object files when using -dir")
	flag.BoolVar(&clean, "clean", false, "remove all build artifacts (.fz_objs, .fz_cache and binaries) from the directory")
	flag.BoolVar(&noCache, "no-cache", false, "disable incremental cache rebuild")
	flag.BoolVar(&noSymbolCheck, "no-symbol-check", false, "disable duplicate symbol pre-check")
	flag.StringVar(&configPath, "config", "", "config file path (default: .fz.yaml, fz.yaml, .fz.yml, fz.yml)")
	flag.BoolVar(&watch, "watch", false, "watch source files and automatically rebuild")
	flag.BoolVar(&sanitize, "sanitize", true, "enable sanitizers (address, undefined) for C code")
	flag.BoolVar(&noSanitize, "no-sanitize", false, "disable sanitizers")
	showVersion := flag.Bool("version", false, "show version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: fz [options] ( -asm <file> | -cc <file> | -dir <directory> | (no arguments with config file) )\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nSupported extensions: .asm/.s/.S/.fasm (assembler), .c (C with strict flags)\n")
	}

	flag.Parse()

	if mode == "" {
		mode = "auto"
	}
	if noSanitize {
		sanitize = false
	}

	if *showVersion {
		fmt.Println("fz version 1.1.0")
		os.Exit(0)
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
		fmt.Fprintln(os.Stderr, "error: specify only one of -asm, -cc, or -dir")
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
	if cfgFile != "" {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
			os.Exit(2)
		}
		cfg.MergeFromFlags(srcPath, dirPath, outBin, outObj, debug, verbose, keepObj, noCache, mode)
		if verbose {
			fmt.Printf("Loaded config from %s\n", cfgFile)
		}
	} else if srcPath == "" && dirPath == "" && !clean {
		fmt.Fprintln(os.Stderr, "error: no config file found and none of -asm, -cc, -dir specified")
		flag.Usage()
		os.Exit(2)
	}

	if clean {
		targetDir := dirPath
		if targetDir == "" && cfg != nil && cfg.SourceDir != "" {
			targetDir = cfg.SourceDir
		}
		if targetDir == "" {
			fmt.Fprintln(os.Stderr, "error: -clean requires -dir <directory> or source_dir in config")
			os.Exit(2)
		}
		if err := builder.CleanDir(targetDir, verbose); err != nil {
			fmt.Fprintf(os.Stderr, "clean failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Cleaned %s\n", targetDir)
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
		fmt.Fprintln(os.Stderr, "error: either -asm, -cc, or -dir must be provided (or set in config)")
		os.Exit(2)
	}
	if srcPath != "" && dirPath != "" {
		fmt.Fprintln(os.Stderr, "error: cannot specify both a single file and -dir")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	build := func() error {
		if srcPath != "" {
			if err := utils.CheckFileExists(srcPath); err != nil {
				return err
			}
			ext := filepath.Ext(srcPath)
			if !utils.SupportedExtension(ext) {
				return fmt.Errorf("unsupported file extension %s", ext)
			}
			binName, objName := utils.DeriveNames(srcPath, outBin, outObj)
			if verbose {
				if ext == ".c" {
					fmt.Printf("Compiling %s -> %s\n", srcPath, objName)
				} else {
					fmt.Printf("Assembling %s -> %s\n", srcPath, objName)
				}
			}
			if err := assembler.Assemble(ctx, srcPath, objName, debug, verbose, mode); err != nil {
				return err
			}
			if verbose {
				fmt.Printf("Linking %s -> %s (mode: %s)\n", objName, binName, mode)
			}
			if err := linker.Link(ctx, objName, binName, verbose, mode, noSymbolCheck, sanitize); err != nil {
				return err
			}
			fmt.Printf("Built: %s\n", binName)
			return nil
		}
		if dirPath != "" {
			info, err := os.Stat(dirPath)
			if err != nil {
				return err
			}
			if !info.IsDir() {
				return fmt.Errorf("-dir argument must be a directory, got: %s", dirPath)
			}
			if outBin != "" {
				if st, err := os.Stat(outBin); err == nil && st.IsDir() {
					return fmt.Errorf("output path %s is a directory, cannot write binary", outBin)
				}
			}
			res, err := builder.BuildDir(ctx, dirPath, outBin, debug, verbose, mode, keepObj, noCache, noSymbolCheck, sanitize)
			if err != nil {
				return err
			}
			if !keepObj && verbose {
				fmt.Printf("Removed temporary object directory: %s\n", res.ObjDir)
			}
			fmt.Printf("Built: %s\n", res.Binary)
			return nil
		}
		return fmt.Errorf("no source to build")
	}

	if err := build(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
		if !watch {
			os.Exit(1)
		}
	}

	if watch {
		w, err := watcher.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
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
			fmt.Fprintf(os.Stderr, "cannot watch directory: %v\n", err)
			os.Exit(1)
		}
		if cfgFile != "" {
			if err := w.Add(cfgFile); err != nil {
				fmt.Fprintf(os.Stderr, "cannot watch config: %v\n", err)
			}
		}
		fmt.Printf("Watching %s for changes...\n", watchTarget)
		w.Watch(500*time.Millisecond, func(string) error {
			fmt.Println("\nChange detected, rebuilding...")
			ctx2, cancel2 := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
			defer cancel2()
			origCtx := ctx
			ctx = ctx2
			err := build()
			ctx = origCtx
			if err != nil {
				fmt.Fprintf(os.Stderr, "rebuild failed: %v\n", err)
			}
			return nil
		})
		select {}
	}
}
