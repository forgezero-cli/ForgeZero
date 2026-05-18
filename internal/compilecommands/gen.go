package compilecommands

import (
	"encoding/json"
	"os"
	"path/filepath"

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
	srcFiles, err := builder.CollectSourceFiles(cfg, []string{"."})
	if err != nil {
		return err
	}
	var commands []CompileCommand
	for _, src := range srcFiles {
		absSrc, err := filepath.Abs(src)
		if err != nil {
			return err
		}
		ext := filepath.Ext(src)
		if ext != ".c" && ext != ".cpp" && ext != ".cc" && ext != ".cxx" {
			continue
		}
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
