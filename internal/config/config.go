package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Flags struct {
	Asm []string `yaml:"asm"`
	Ld  []string `yaml:"ld"`
}

type Config struct {
	Name       string   `yaml:"name"`
	SourceDir  string   `yaml:"source_dir"`
	SourceFile string   `yaml:"source_file"`
	Output     string   `yaml:"output"`
	OutObj     string   `yaml:"out_obj"`
	Mode       string   `yaml:"mode"`
	Debug      bool     `yaml:"debug"`
	Verbose    bool     `yaml:"verbose"`
	KeepObj    bool     `yaml:"keep_obj"`
	NoCache    bool     `yaml:"no_cache"`
	Exclude    []string `yaml:"exclude"`
	Flags      Flags    `yaml:"flags"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse YAML: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.SourceDir == "" && c.SourceFile == "" {
		return fmt.Errorf("either source_dir or source_file must be set")
	}
	if c.SourceDir != "" && c.SourceFile != "" {
		return fmt.Errorf("cannot set both source_dir and source_file")
	}
	if c.Mode == "" {
		c.Mode = "auto"
	}
	if c.Mode != "auto" && c.Mode != "c" && c.Mode != "raw" {
		return fmt.Errorf("invalid mode: %s", c.Mode)
	}
	return nil
}

func (c *Config) MergeFromFlags(srcPath, dirPath, outBin, outObj string, debug, verbose, keepObj, noCache bool, mode string) {
	if srcPath != "" {
		c.SourceFile = srcPath
		c.SourceDir = ""
	}
	if dirPath != "" {
		c.SourceDir = dirPath
		c.SourceFile = ""
	}
	if outBin != "" {
		c.Output = outBin
	}
	if outObj != "" {
		c.OutObj = outObj
	}
	if debug {
		c.Debug = debug
	}
	if verbose {
		c.Verbose = verbose
	}
	if keepObj {
		c.KeepObj = keepObj
	}
	if noCache {
		c.NoCache = noCache
	}
	if mode != "" && mode != "auto" {
		c.Mode = mode
	}
}

func DefaultConfigPath() string {
	paths := []string{".fz.yaml", "fz.yaml", ".fz.yml", "fz.yml"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
