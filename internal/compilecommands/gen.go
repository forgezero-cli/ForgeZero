package compilecommands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"fz/internal/assembler"
	"fz/internal/builder"
	"fz/internal/config"
)

type CompileCommand struct {
	Directory string   `json:"directory"`
	File      string   `json:"file"`
	Arguments []string `json:"arguments"`
}

func Generate(cfg *config.Config, rootDir string) error {
	if cfg == nil {
		cfg = &config.Config{}
	}
	srcFiles, err := builder.CollectSourceFiles(cfg, []string{rootDir})
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	var commands []CompileCommand
	for _, src := range srcFiles {
		ext := strings.ToLower(filepath.Ext(src))
		if ext != ".c" && ext != ".cpp" && ext != ".cc" && ext != ".cxx" {
			continue
		}
		absSrc, err := filepath.Abs(src)
		if err != nil {
			return err
		}
		if seen[absSrc] {
			continue
		}
		seen[absSrc] = true
		dir := filepath.Dir(absSrc)
		args := []string{assembler.CCForTarget()}
		args = append(args, "-c", absSrc)
		strictFlags := []string{"-Wall", "-Wextra", "-Werror", "-Wpedantic", "-Wshadow", "-Wconversion"}
		args = append(args, strictFlags...)
		if cfg.Debug {
			args = append(args, "-g")
		}
		commands = append(commands, CompileCommand{
			Directory: dir,
			File:      absSrc,
			Arguments: args,
		})
	}
	data, err := json.MarshalIndent(commands, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("compile_commands.json", data, 0o644)
}
