package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fz/internal/assembler"
	"fz/internal/linker"
	"fz/internal/utils"
)

func main() {
	var (
		srcPath    string
		debug      bool
		verbose    bool
		outBin     string
		outObj     string
		timeoutSec int
		mode       string
	)

	flag.StringVar(&srcPath, "asm", "", "assembler source file (required)")
	flag.StringVar(&srcPath, "assembler", "", "assembler source file (required)")
	flag.BoolVar(&debug, "debug", false, "emit debug information")
	flag.BoolVar(&verbose, "verbose", false, "print executed commands")
	flag.StringVar(&outBin, "out", "", "output binary name")
	flag.StringVar(&outObj, "out-obj", "", "output object file name")
	flag.IntVar(&timeoutSec, "timeout", 60, "timeout in seconds for external commands")
	flag.StringVar(&mode, "mode", "auto", "linking mode: auto, c, raw")
	showVersion := flag.Bool("version", false, "show version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: fz [options] -asm <file>\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nSupported source extensions: .asm (NASM), .s/.S (GAS), .fasm (FASM)\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Println("fz version 1.0")
		os.Exit(0)
	}

	if srcPath == "" {
		fmt.Fprintln(os.Stderr, "error: source file required (-asm or --assembler)")
		flag.Usage()
		os.Exit(2)
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

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
	if err := linker.Link(ctx, objName, binName, verbose, mode); err != nil {
		fmt.Fprintf(os.Stderr, "link error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Built: %s\n", binName)
}
