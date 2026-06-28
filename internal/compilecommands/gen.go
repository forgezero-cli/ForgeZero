/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package compilecommands

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/assembler"
	"github.com/forgezero-cli/ForgeZero/internal/builder"
	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
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
	return utils.SecureWriteFile("compile_commands.json", data)
}
