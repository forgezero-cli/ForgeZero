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

package config

import (
	"os"
	"path/filepath"
	"strings"

	fzerr "github.com/forgezero-cli/ForgeZero/internal/errors"
	"github.com/forgezero-cli/ForgeZero/internal/fzp"
)

func isFZPPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".fz" || ext == ".fzp"
}

func LoadFZP(path string) (*Config, error) {
	if path == "" {
		return &Config{}, nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fzerr.NewMsg(fzerr.CodePreprocessFailed, "cannot stat fzp config "+path+": "+err.Error())
	}
	proc := fzp.NewProcessor(fzp.Options{RootDir: filepath.Dir(path)})
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	_, err = proc.Process(path, fzp.Options{RootDir: filepath.Dir(path)})
	if err != nil {
		return nil, err
	}
	defs, err := proc.ParseDefinitions(string(data))
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	for k, v := range defs {
		if strings.EqualFold(k, "OUTPUT") {
			cfg.Output = v
		} else if strings.EqualFold(k, "MODE") {
			cfg.Mode = v
		} else if strings.EqualFold(k, "SOURCE_DIR") {
			cfg.SourceDir = v
		}
	}
	cfg.expand()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadMergedFZP(paths []string) (*Config, error) {
	var cfg Config
	for _, path := range paths {
		if path == "" {
			continue
		}
		child, err := LoadFZP(path)
		if err != nil {
			return nil, err
		}
		cfg.Merge(child)
	}
	cfg.expand()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}
