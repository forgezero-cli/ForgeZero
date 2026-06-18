package shell

import (
	"context"
	"errors"
	"os"
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
		return errors.New("no source path set")
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
		return errors.New("unsupported extension: " + ext)
	}
	binName, objName := utils.DeriveNames(state.SourcePath, state.Out, "")
	if state.Verbose {
		if ext == ".c" || ext == ".cpp" || ext == ".cc" || ext == ".cxx" {
			os.Stdout.WriteString("Compiling " + state.SourcePath + " -> " + objName + "\n")
		} else {
			os.Stdout.WriteString("Assembling " + state.SourcePath + " -> " + objName + "\n")
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
		os.Stdout.WriteString("Linking " + objName + " -> " + binName + " (mode: " + state.Mode + ")\n")
	}
	if err := linker.Link(ctx, objName, binName, state.Verbose, state.Mode, state.NoSymbolCheck, state.Sanitize, state.Strict, nil); err != nil {
		return err
	}
	state.Out = binName
	return nil
}

func cmdClean(state *BuildState) error {
	if state.SourceType != "dir" {
		return errors.New("clean only works for directory builds")
	}
	return builder.CleanDir(state.SourcePath, state.Verbose)
}

func cmdSet(state *BuildState, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: set key=value")
	}
	parts := strings.SplitN(args[1], "=", 2)
	if len(parts) != 2 {
		return errors.New("invalid format, use key=value")
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
		return errors.New("unknown key: " + key)
	}
	os.Stdout.WriteString("Set " + key + " = " + val + "\n")
	return nil
}

func cmdShow(state *BuildState) {
	os.Stdout.WriteString("Mode: " + state.Mode + "\n")
	os.Stdout.WriteString("Format: " + state.Format + "\n")
	os.Stdout.WriteString("Strict: " + boolStr(state.Strict) + "\n")
	os.Stdout.WriteString("Sanitize: " + boolStr(state.Sanitize) + "\n")
	os.Stdout.WriteString("Verbose: " + boolStr(state.Verbose) + "\n")
	os.Stdout.WriteString("Debug: " + boolStr(state.Debug) + "\n")
	os.Stdout.WriteString("NoCache: " + boolStr(state.NoCache) + "\n")
	os.Stdout.WriteString("NoSymbolCheck: " + boolStr(state.NoSymbolCheck) + "\n")
	os.Stdout.WriteString("KeepObj: " + boolStr(state.KeepObj) + "\n")
	os.Stdout.WriteString("LdScript: " + state.LdScript + "\n")
	os.Stdout.WriteString("TextAddr: " + state.TextAddr + "\n")
	os.Stdout.WriteString("Output: " + state.Out + "\n")
	os.Stdout.WriteString("Source: " + state.SourcePath + " (type: " + state.SourceType + ")\n")
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func cmdHelp() {
	os.Stdout.WriteString(helpText)
}

const helpText = `Commands:
  build               Build project with current settings
  clean               Remove build artifacts
  set key=value       Change a setting (mode, format, strict, ...)
  show                Show current settings
  watch               Start watch mode (auto-rebuild)
  exit, quit          Exit shell
  help                Show this help`