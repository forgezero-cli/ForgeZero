package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var supportedToolchains = map[string]struct{}{
	"auto":  {},
	"zig":   {},
	"fasm":  {},
	"nasm":  {},
	"gas":   {},
	"gcc":   {},
	"clang": {},
	"ld":    {},
}

type IsolationMode string

const (
	IsolationNone     IsolationMode = "none"
	IsolationStandard IsolationMode = "standard"
	IsolationStrict   IsolationMode = "strict"
)

func (m IsolationMode) String() string {
	return string(m)
}

func (m *IsolationMode) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err == nil {
		s = strings.ToLower(strings.TrimSpace(s))
		switch s {
		case "", "none":
			*m = IsolationNone
			return nil
		case "standard", "true":
			*m = IsolationStandard
			return nil
		case "strict":
			*m = IsolationStrict
			return nil
		}
		return fmt.Errorf("invalid isolation: %s", s)
	}
	var b bool
	if err := node.Decode(&b); err == nil {
		if b {
			*m = IsolationStandard
			return nil
		}
		*m = IsolationNone
		return nil
	}
	return fmt.Errorf("invalid isolation value")
}

type Flags struct {
	Asm []string `yaml:"asm"`
	Cc  []string `yaml:"cc"`
	Ld  []string `yaml:"ld"`
}

type Hook struct {
	Cmd      string `yaml:"cmd"`
	Critical bool   `yaml:"critical"`
}

type Hooks struct {
	PreBuild  []Hook `yaml:"pre_build"`
	OnFailure string `yaml:"on_failure"`
}

type Config struct {
	Name               string            `yaml:"name"`
	SourceDir          string            `yaml:"source_dir"`
	SourceDirs         []string          `yaml:"source_dirs"`
	SourceFiles        []string          `yaml:"source_files"`
	SourceFile         string            `yaml:"source_file"`
	Output             string            `yaml:"output"`
	OutObj             string            `yaml:"out_obj"`
	Mode               string            `yaml:"mode"`
	Toolchain          string            `yaml:"toolchain"`
	Debug              bool              `yaml:"debug"`
	Verbose            bool              `yaml:"verbose"`
	KeepObj            bool              `yaml:"keep_obj"`
	NoCache            bool              `yaml:"no_cache"`
	OptimizationLevel  int               `yaml:"optimization_level"`
	Exclude            []string          `yaml:"exclude"`
	Include            []string          `yaml:"include"`
	Libs               []string          `yaml:"libs"`
	IgnoreFile         string            `yaml:"ignore_file"`
	AuditIgnore        []string          `yaml:"audit_ignore"`
	ToolChecksums      map[string]string `yaml:"tool_checksums"`
	Flags              Flags             `yaml:"flags"`
	Isolation          IsolationMode     `yaml:"isolation"`
	DeterministicStrip bool              `yaml:"deterministic_strip"`
	ToolchainSettings  struct {
		SearchPriority []string          `yaml:"search_priority"`
		EnvAllow       []string          `yaml:"env_allow"`
		ToolPaths      map[string]string `yaml:"tool_paths"`
	} `yaml:"toolchain_opts"`
	Hooks Hooks `yaml:"hooks"`
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
	if c.SourceDir == "" && len(c.SourceDirs) == 0 && c.SourceFile == "" && len(c.SourceFiles) == 0 {
		return nil
	}
	if c.SourceDir != "" && len(c.SourceDirs) > 0 {
		return fmt.Errorf("cannot set both source_dir and source_dirs")
	}
	if c.SourceFile != "" && len(c.SourceFiles) > 0 {
		return fmt.Errorf("cannot set both source_file and source_files")
	}
	if c.Mode == "" {
		c.Mode = "auto"
	}
	if c.Mode != "auto" && c.Mode != "c" && c.Mode != "raw" {
		return fmt.Errorf("invalid mode: %s", c.Mode)
	}
	if c.Toolchain == "" {
		c.Toolchain = "auto"
	}
	c.Toolchain = strings.TrimSpace(strings.ToLower(c.Toolchain))
	if _, ok := supportedToolchains[c.Toolchain]; !ok {
		return fmt.Errorf("invalid toolchain: %s", c.Toolchain)
	}
	if c.Isolation == "" {
		c.Isolation = IsolationNone
	}
	switch c.Isolation {
	case IsolationNone, IsolationStandard, IsolationStrict:
	default:
		return fmt.Errorf("invalid isolation: %s", c.Isolation)
	}
	if c.IgnoreFile == "" {
		c.IgnoreFile = ".fzignore"
	}
	return nil
}

func (c *Config) IsolationEnabled() bool {
	return c.Isolation != IsolationNone
}

func (c *Config) IsStrictIsolation() bool {
	return c.Isolation == IsolationStrict
}

func (c *Config) MergeFromFlags(srcPath, dirPath, outBin, outObj string, debug, verbose, keepObj, noCache bool, mode, toolchain, isolation string) {
	if srcPath != "" {
		c.SourceFile = srcPath
		c.SourceDir = ""
		c.SourceDirs = nil
		c.SourceFiles = nil
	}
	if dirPath != "" {
		c.SourceDir = dirPath
		c.SourceDirs = nil
		c.SourceFiles = nil
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
	if toolchain != "" && toolchain != "auto" {
		c.Toolchain = toolchain
	}
	if isolation != "" {
		switch isolation {
		case "none", "standard", "strict":
			c.Isolation = IsolationMode(strings.ToLower(isolation))
		default:
			c.Isolation = IsolationNone
		}
	}
}

func IsValidToolchain(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	_, ok := supportedToolchains[name]
	return ok
}

func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}
	if other.Name != "" {
		c.Name = other.Name
	}
	if other.SourceDir != "" {
		c.SourceDir = other.SourceDir
		c.SourceDirs = nil
		c.SourceFiles = nil
	}
	if len(other.SourceDirs) > 0 {
		c.SourceDirs = other.SourceDirs
		c.SourceDir = ""
	}
	if len(other.SourceFiles) > 0 {
		c.SourceFiles = other.SourceFiles
		c.SourceFile = ""
	}
	if other.SourceFile != "" {
		c.SourceFile = other.SourceFile
		c.SourceDir = ""
		c.SourceDirs = nil
		c.SourceFiles = nil
	}
	if other.Output != "" {
		c.Output = other.Output
	}
	if other.OutObj != "" {
		c.OutObj = other.OutObj
	}
	if other.Mode != "" {
		c.Mode = other.Mode
	}
	if other.Debug {
		c.Debug = other.Debug
	}
	if other.Verbose {
		c.Verbose = other.Verbose
	}
	if other.KeepObj {
		c.KeepObj = other.KeepObj
	}
	if other.NoCache {
		c.NoCache = other.NoCache
	}
	if other.Isolation != "" {
		c.Isolation = other.Isolation
	}
	if len(other.Exclude) > 0 {
		c.Exclude = other.Exclude
	}
	if len(other.Include) > 0 {
		c.Include = other.Include
	}
	if len(other.Libs) > 0 {
		c.Libs = other.Libs
	}
	if other.IgnoreFile != "" {
		c.IgnoreFile = other.IgnoreFile
	}
	if len(other.AuditIgnore) > 0 {
		c.AuditIgnore = other.AuditIgnore
	}
	if len(other.ToolChecksums) > 0 {
		if c.ToolChecksums == nil {
			c.ToolChecksums = make(map[string]string)
		}
		for k, v := range other.ToolChecksums {
			c.ToolChecksums[k] = v
		}
	}
	if len(other.Flags.Asm) > 0 {
		c.Flags.Asm = other.Flags.Asm
	}
	if len(other.Flags.Cc) > 0 {
		c.Flags.Cc = other.Flags.Cc
	}
	if len(other.Flags.Ld) > 0 {
		c.Flags.Ld = other.Flags.Ld
	}
	if other.OptimizationLevel > 0 {
		c.OptimizationLevel = other.OptimizationLevel
	}
}

func FindConfigs() (system, user, local string) {
	systemPaths := []string{"/etc/fz/config.yaml", "/etc/fz.yaml"}
	for _, p := range systemPaths {
		if _, err := os.Stat(p); err == nil {
			system = p
			break
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		userPaths := []string{
			filepath.Join(home, ".config", "fz", "config.yaml"),
			filepath.Join(home, ".fz.yaml"),
		}
		for _, p := range userPaths {
			if _, err := os.Stat(p); err == nil {
				user = p
				break
			}
		}
	}
	localPaths := []string{".fz.yaml", "fz.yaml", ".fz.yml", "fz.yml"}
	for _, p := range localPaths {
		if _, err := os.Stat(p); err == nil {
			local = p
			break
		}
	}
	return
}

func LoadMerged(explicitPath string) (*Config, error) {
	var cfg Config
	if explicitPath != "" {
		explicitCfg, err := Load(explicitPath)
		if err != nil {
			return nil, err
		}
		cfg.Merge(explicitCfg)
		return &cfg, nil
	}
	systemPath, userPath, localPath := FindConfigs()
	if systemPath != "" {
		if sysCfg, err := Load(systemPath); err == nil {
			cfg.Merge(sysCfg)
		}
	}
	if userPath != "" {
		if userCfg, err := Load(userPath); err == nil {
			cfg.Merge(userCfg)
		}
	}
	if localPath != "" {
		if localCfg, err := Load(localPath); err == nil {
			cfg.Merge(localCfg)
		}
	}
	return &cfg, nil
}

func DefaultConfigPath() string {
	_, _, local := FindConfigs()
	return local
}
