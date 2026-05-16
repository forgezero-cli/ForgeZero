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
)

func main() {
	var (
		srcPath       string
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
	)

	flag.StringVar(&srcPath, "asm", "", "assembler source file")
	flag.StringVar(&srcPath, "assembler", "", "assembler source file (alias)")
	flag.StringVar(&dirPath, "dir", "", "directory containing assembly files (recursive)")
	flag.BoolVar(&debug, "debug", false, "emit debug information")
	flag.BoolVar(&verbose, "verbose", false, "print executed commands")
	flag.StringVar(&outBin, "out", "", "output binary name")
	flag.StringVar(&outObj, "out-obj", "", "output object file name (only with -asm)")
	flag.IntVar(&timeoutSec, "timeout", 60, "timeout in seconds for external commands")
	flag.StringVar(&mode, "mode", "", "linking mode: auto, c, raw")
	flag.BoolVar(&keepObj, "keep-obj", false, "keep temporary object files when using -dir")
	flag.BoolVar(&clean, "clean", false, "remove all build artifacts (.fz_objs, .fz_cache and binaries) from the directory")
	flag.BoolVar(&noCache, "no-cache", false, "disable incremental cache rebuild")
	flag.BoolVar(&noSymbolCheck, "no-symbol-check", false, "disable duplicate symbol pre-check")
	flag.StringVar(&configPath, "config", "", "config file path (default: .fz.yaml, fz.yaml, .fz.yml, fz.yml)")
	showVersion := flag.Bool("version", false, "show version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: fz [options] (-asm <file> | -dir <directory> | (no arguments with config file))\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nSupported source extensions: .asm (NASM), .s/.S (GAS), .fasm (FASM)\n")
	}

	flag.Parse()

	if mode == "" {
		mode = "auto"
	}

	if *showVersion {
		fmt.Println("fz version 1.2")
		os.Exit(0)
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
		fmt.Fprintln(os.Stderr, "error: no config file found and neither -asm nor -dir specified")
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
		fmt.Fprintln(os.Stderr, "error: either -asm or -dir must be provided (or set in config)")
		os.Exit(2)
	}
	if srcPath != "" && dirPath != "" {
		fmt.Fprintln(os.Stderr, "error: cannot specify both -asm and -dir")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	if srcPath != "" {
		if err := utils.CheckFileExists(srcPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
		ext := filepath.Ext(srcPath)
		if !utils.SupportedExtension(ext) {
			fmt.Fprintf(os.Stderr, "error: unsupported file extension %s\n", ext)
			os.Exit(2)
		}
		binName, objName := utils.DeriveNames(srcPath, outBin, outObj)
		if verbose {
			fmt.Printf("Assembling %s -> %s\n", srcPath, objName)
		}
		if err := assembler.Assemble(ctx, srcPath, objName, debug, verbose, mode); err != nil {
			fmt.Fprintf(os.Stderr, "assemble error: %v\n", err)
			os.Exit(1)
		}
		if verbose {
			fmt.Printf("Linking %s -> %s (mode: %s)\n", objName, binName, mode)
		}
		if err := linker.Link(ctx, objName, binName, verbose, mode, noSymbolCheck); err != nil {
			fmt.Fprintf(os.Stderr, "link error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Built: %s\n", binName)
		return
	}

	if dirPath != "" {
		info, err := os.Stat(dirPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot access directory: %v\n", err)
			os.Exit(2)
		}
		if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "error: -dir argument must be a directory, got: %s\n", dirPath)
			os.Exit(2)
		}
		if outBin != "" {
			if st, err := os.Stat(outBin); err == nil && st.IsDir() {
				fmt.Fprintf(os.Stderr, "error: output path %s is a directory, cannot write binary\n", outBin)
				os.Exit(2)
			}
		}
		res, err := builder.BuildDir(ctx, dirPath, outBin, debug, verbose, mode, keepObj, noCache, noSymbolCheck)
		if err != nil {
			fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
			os.Exit(1)
		}
		if !keepObj && verbose {
			fmt.Printf("Removed temporary object directory: %s\n", res.ObjDir)
		}
		fmt.Printf("Built: %s\n", res.Binary)
	}
}
