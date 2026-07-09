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
