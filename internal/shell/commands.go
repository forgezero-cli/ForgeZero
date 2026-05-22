package shell

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fz/internal/assembler"
	"fz/internal/builder"
	"fz/internal/linker"
	"fz/internal/utils"
)

func cmdBuild(state *BuildState) error {
	if state.SourcePath == "" {
		return fmt.Errorf("no source path set")
	}

	if state.SourceType == "dir" {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		var dirs []string
		if state.SourcePath != "" {
			dirs = []string{state.SourcePath}
		} else {
			dirs = []string{"."}
		}
		res, err := builder.BuildDir(ctx, dirs, state.Out, state.Debug, state.Verbose, state.Mode,
			state.KeepObj, state.NoCache, state.NoSymbolCheck, state.Sanitize, state.Strict,
			nil, nil, nil, nil, nil, 1, "executable")
		if err != nil {
			return err
		}
		state.Out = res.Binary
		return nil
	}

	ext := filepath.Ext(state.SourcePath)
	if !utils.SupportedExtension(ext) {
		return fmt.Errorf("unsupported extension: %s", ext)
	}
	binName, objName := utils.DeriveNames(state.SourcePath, state.Out, "")
	if state.Verbose {
		if ext == ".c" || ext == ".cpp" || ext == ".cc" || ext == ".cxx" {
			fmt.Printf("Compiling %s -> %s\n", state.SourcePath, objName)
		} else {
			fmt.Printf("Assembling %s -> %s\n", state.SourcePath, objName)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if state.Format != "" {
		if err := linker.SetOutputFormat(state.Format); err != nil {
			return err
		}
	}
	if assembler.IsBinFormat() && strings.HasSuffix(strings.ToLower(objName), ".o") {
		objName = binName
	}
	if err := assembler.Assemble(ctx, state.SourcePath, objName, state.Debug, state.Verbose, state.Mode); err != nil {
		return err
	}
	if state.Verbose {
		fmt.Printf("Linking %s -> %s (mode: %s)\n", objName, binName, state.Mode)
	}
	if err := linker.Link(ctx, objName, binName, state.Verbose, state.Mode, state.NoSymbolCheck, state.Sanitize, state.Strict, nil); err != nil {
		return err
	}
	state.Out = binName
	return nil
}

func cmdClean(state *BuildState) error {
	if state.SourceType != "dir" {
		return fmt.Errorf("clean only works for directory builds")
	}
	return builder.CleanDir(state.SourcePath, state.Verbose)
}

func cmdSet(state *BuildState, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: set key=value")
	}
	parts := strings.SplitN(args[1], "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, use key=value")
	}
	key, val := parts[0], parts[1]
	switch key {
	case "mode":
		state.Mode = val
	case "format":
		state.Format = val
	case "strict":
		state.Strict = val == "true"
	case "sanitize":
		state.Sanitize = val == "true"
	case "verbose":
		state.Verbose = val == "true"
	case "debug":
		state.Debug = val == "true"
	case "no-cache":
		state.NoCache = val == "true"
	case "no-symbol-check":
		state.NoSymbolCheck = val == "true"
	case "keep-obj":
		state.KeepObj = val == "true"
	case "ld-script":
		state.LdScript = val
	case "text-addr":
		state.TextAddr = val
	case "out":
		state.Out = val
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
	fmt.Printf("Set %s = %s\n", key, val)
	return nil
}

func cmdShow(state *BuildState) {
	fmt.Printf("Mode: %s\n", state.Mode)
	fmt.Printf("Format: %s\n", state.Format)
	fmt.Printf("Strict: %v\n", state.Strict)
	fmt.Printf("Sanitize: %v\n", state.Sanitize)
	fmt.Printf("Verbose: %v\n", state.Verbose)
	fmt.Printf("Debug: %v\n", state.Debug)
	fmt.Printf("NoCache: %v\n", state.NoCache)
	fmt.Printf("NoSymbolCheck: %v\n", state.NoSymbolCheck)
	fmt.Printf("KeepObj: %v\n", state.KeepObj)
	fmt.Printf("LdScript: %s\n", state.LdScript)
	fmt.Printf("TextAddr: %s\n", state.TextAddr)
	fmt.Printf("Output: %s\n", state.Out)
	fmt.Printf("Source: %s (type: %s)\n", state.SourcePath, state.SourceType)
}

func cmdHelp() {
	fmt.Println(`Commands:
  build               Build project with current settings
  clean               Remove build artifacts
  set key=value       Change a setting (mode, format, strict, ...)
  show                Show current settings
  watch               Start watch mode (auto-rebuild)
  exit, quit          Exit shell
  help                Show this help`)
}
