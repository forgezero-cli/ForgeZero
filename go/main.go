package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func deriveNames(src, outFlag, outObjFlag string) (bin, obj string) {
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	objDefault := base + ".o"
	binDefault := base
	if runtime.GOOS == "windows" && filepath.Ext(binDefault) == "" {
		binDefault += ".exe"
	}
	if outObjFlag != "" {
		obj = outObjFlag
	} else {
		obj = objDefault
	}
	if outFlag != "" {
		bin = outFlag
	} else {
		bin = binDefault
	}
	return
}

func assemble(ctx context.Context, src, obj string, debug, verbose bool, mode string) error {
	ext := strings.ToLower(filepath.Ext(src))
	switch ext {
	case ".asm": // NASM
		args := []string{"-felf64", src, "-o", obj}
		if debug {
			args = append([]string{"-g"}, args...)
		}
		if verbose {
			fmt.Println("Running:", "nasm", strings.Join(args, " "))
		}
		return runCommand(ctx, "nasm", args...)

	case ".s", ".S":
		args := []string{"-c", src, "-o", obj}
		if debug {
			args = append([]string{"-g"}, args...)
		}
		if verbose {
			fmt.Println("Running:", "gcc", strings.Join(args, " "))
		}
		return runCommand(ctx, "gcc", args...)

	case ".fasm":
		args := []string{src, obj}
		if verbose {
			fmt.Println("Running:", "fasm", strings.Join(args, " "))
		}
		return runCommand(ctx, "fasm", args...)

	default:
		return fmt.Errorf("unsupported source extension: %s", ext)
	}
}

func linkWithGcc(ctx context.Context, obj, bin string, verbose bool, extraArgs ...string) error {
	args := append([]string{obj, "-o", bin}, extraArgs...)
	if verbose {
		fmt.Println("Running:", "gcc", strings.Join(args, " "))
	}
	return runCommand(ctx, "gcc", args...)
}

func linkWithLd(ctx context.Context, obj, bin string, verbose bool, extraArgs ...string) error {
	args := append([]string{obj, "-o", bin}, extraArgs...)
	if verbose {
		fmt.Println("Running:", "ld", strings.Join(args, " "))
	}
	return runCommand(ctx, "ld", args...)
}

func link(ctx context.Context, obj, bin string, verbose bool, mode string) error {
	if mode == "raw" {
		return linkWithLd(ctx, obj, bin, verbose)
	}

	err := linkWithGcc(ctx, obj, bin, verbose)
	if err == nil {
		return nil
	}

	if verbose {
		fmt.Println("Link with gcc failed, retrying with -no-pie")
	}
	err2 := linkWithGcc(ctx, obj, bin, verbose, "-no-pie")
	if err2 == nil {
		return nil
	}

	if verbose {
		fmt.Println("gcc retries failed, trying ld directly")
	}
	return linkWithLd(ctx, obj, bin, verbose)
}

func main() {
	fmt.Println("B-Asm then in GOLANG")
	asm := flag.String("assembler", "", "path to assembler source (required)")
	debug := flag.Bool("debug", false, "enable debug flags")
	verbose := flag.Bool("verbose", false, "print commands")
	out := flag.String("out", "", "binary name (default: derived from source)")
	outObj := flag.String("out-obj", "", "object name (default: <src-base>.o)")
	timeout := flag.Int("timeout", 60, "timeout seconds for external commands")
	mode := flag.String("mode", "auto", "entry/link mode: auto|c|raw")
	flag.Parse()

	if *asm == "" {
		fmt.Fprintln(os.Stderr, "error: --assembler required")
		flag.Usage()
		os.Exit(2)
	}

	binName, objName := deriveNames(*asm, *out, *outObj)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	if err := assemble(ctx, *asm, objName, *debug, *verbose, *mode); err != nil {
		log.Fatalf("assemble failed: %v", err)
	}

	if err := link(ctx, objName, binName, *verbose, *mode); err != nil {
		log.Fatalf("link failed: %v", err)
	}

	fmt.Println("Built:", binName)
}
