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

// AUTHOR: @alexvoste

package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/variables"

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
		return errors.New("invalid isolation: " + s)
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
	return errors.New("invalid isolation value")
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
	Name    string `yaml:"name"`
	Profile string `yaml:"profile"`
	Target  string `yaml:"target"`
	Sysroot string `yaml:"sysroot"`

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
	Scripts            []string          `yaml:"scripts"`
	Libs               []string          `yaml:"libs"`
	IgnoreFile         string            `yaml:"ignore_file"`
	AuditIgnore        []string          `yaml:"audit_ignore"`
	ToolChecksums      map[string]string `yaml:"tool_checksums"`
	Variables          map[string]string `yaml:"variables"`
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

func (c *Config) expand() {
	if len(c.Variables) == 0 {
		return
	}
	vars := c.Variables
	c.Name = variables.ExpandString(c.Name, vars)
	c.Profile = variables.ExpandString(c.Profile, vars)
	c.Target = variables.ExpandString(c.Target, vars)
	c.Sysroot = variables.ExpandString(c.Sysroot, vars)
	c.SourceDir = variables.ExpandString(c.SourceDir, vars)
	variables.ExpandSlice(c.SourceDirs, vars)
	variables.ExpandSlice(c.SourceFiles, vars)
	c.SourceFile = variables.ExpandString(c.SourceFile, vars)
	c.Output = variables.ExpandString(c.Output, vars)
	c.OutObj = variables.ExpandString(c.OutObj, vars)
	c.Mode = variables.ExpandString(c.Mode, vars)
	c.Toolchain = variables.ExpandString(c.Toolchain, vars)
	c.IgnoreFile = variables.ExpandString(c.IgnoreFile, vars)
	variables.ExpandSlice(c.Exclude, vars)
	variables.ExpandSlice(c.Include, vars)
	variables.ExpandSlice(c.Scripts, vars)
	variables.ExpandSlice(c.Libs, vars)
	variables.ExpandSlice(c.AuditIgnore, vars)
	variables.ExpandSlice(c.Flags.Asm, vars)
	variables.ExpandSlice(c.Flags.Cc, vars)
	variables.ExpandSlice(c.Flags.Ld, vars)
	variables.ExpandMap(c.ToolChecksums, vars)
	variables.ExpandMap(c.ToolchainSettings.ToolPaths, vars)
	variables.ExpandSlice(c.ToolchainSettings.SearchPriority, vars)
	variables.ExpandSlice(c.ToolchainSettings.EnvAllow, vars)
	for i := range c.Hooks.PreBuild {
		c.Hooks.PreBuild[i].Cmd = variables.ExpandString(c.Hooks.PreBuild[i].Cmd, vars)
	}
	c.Hooks.OnFailure = variables.ExpandString(c.Hooks.OnFailure, vars)
	c.Isolation = IsolationMode(variables.ExpandString(string(c.Isolation), vars))
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("cannot read config file " + path + ": " + err.Error())
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.New("cannot parse YAML: " + err.Error())
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
		return errors.New("cannot set both source_dir and source_dirs")
	}
	if c.SourceFile != "" && len(c.SourceFiles) > 0 {
		return errors.New("cannot set both source_file and source_files")
	}
	if c.Mode == "" {
		c.Mode = "auto"
	}
	if c.Mode != "auto" && c.Mode != "c" && c.Mode != "raw" {
		return errors.New("invalid mode: " + c.Mode)
	}
	if c.Profile == "" {
		c.Profile = "balanced"
	}
	c.Profile = strings.TrimSpace(strings.ToLower(c.Profile))
	if c.Profile != "balanced" && c.Profile != "powered" && c.Profile != "performance" {
		return errors.New("invalid profile: " + c.Profile)
	}
	if c.Toolchain == "" {
		c.Toolchain = "auto"
	}
	c.Toolchain = strings.TrimSpace(strings.ToLower(c.Toolchain))
	if _, ok := supportedToolchains[c.Toolchain]; !ok {
		return errors.New("invalid toolchain: " + c.Toolchain)
	}
	if c.Isolation == "" {
		c.Isolation = IsolationNone
	}
	switch c.Isolation {
	case IsolationNone, IsolationStandard, IsolationStrict:
	default:
		return errors.New("invalid isolation: " + string(c.Isolation))
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
	if other.Target != "" {
		c.Target = other.Target
	}
	if other.Sysroot != "" {
		c.Sysroot = other.Sysroot
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
	if other.Profile != "" {
		c.Profile = other.Profile
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
	if len(other.Scripts) > 0 {
		c.Scripts = other.Scripts
	}
	if len(other.Variables) > 0 {
		if c.Variables == nil {
			c.Variables = make(map[string]string)
		}
		for k, v := range other.Variables {
			c.Variables[k] = v
		}
	}
}

func FindConfigs() (system, user, local string) {
	systemPaths := []string{"/etc/github.com/forgezero-cli/ForgeZero/config.yaml", "/etc/fz.yaml"}
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
		cfg.expand()
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
	cfg.expand()
	return &cfg, nil
}

func DefaultConfigPath() string {
	_, _, local := FindConfigs()
	return local
}

func GenerateFromScan(root string) (*Config, error) {
	var sourceDirs []string
	var includeDirs []string
	found := make(map[string]bool)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".fz_objs" || name == "build" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".c", ".cpp", ".cc", ".cxx", ".asm", ".s", ".S", ".fasm":
			dir := filepath.Dir(path)
			if !found[dir] {
				sourceDirs = append(sourceDirs, dir)
				found[dir] = true
			}
		case ".h", ".hpp":
			dir := filepath.Dir(path)
			key := "inc:" + dir
			if !found[key] {
				includeDirs = append(includeDirs, dir)
				found[key] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(sourceDirs) == 0 {
		return nil, errors.New("no source files found")
	}

	cfg := &Config{
		SourceDirs: sourceDirs,
		Include:    includeDirs,
		Output:     "app",
		Mode:       "auto",
		Profile:    "balanced",
		Debug:      false,
		Verbose:    false,
		KeepObj:    false,
		NoCache:    false,
	}
	return cfg, nil
}
