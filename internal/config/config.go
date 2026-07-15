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
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/forgezero-cli/ForgeZero/internal/config/toml"
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

type CacheMode string

const (
	IsolationNone     IsolationMode = "none"
	IsolationStandard IsolationMode = "standard"
	IsolationStrict   IsolationMode = "strict"

	CacheModeDisk CacheMode = "disk"
	CacheModeRAM  CacheMode = "ram"
	CacheModeOff  CacheMode = "off"
)

func (m IsolationMode) String() string {
	return string(m)
}

func (m CacheMode) String() string {
	return string(m)
}

func (m *CacheMode) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	return m.unmarshalValue(s)
}

func (m *CacheMode) UnmarshalText(text []byte) error {
	return m.unmarshalValue(string(text))
}

func (m *CacheMode) unmarshalValue(value string) error {
	s := strings.ToLower(strings.TrimSpace(value))
	switch s {
	case "", "disk":
		*m = CacheModeDisk
	case "ram":
		*m = CacheModeRAM
	case "off":
		*m = CacheModeOff
	default:
		return NewErrorDetail(ErrorInvalidCacheMode, s)
	}
	return nil
}

func loadYAML(data []byte, cfg *Config) error {
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return NewErrorDetail(ErrorParseYAML, err.Error())
	}
	os.Stderr.WriteString("WARNING: YAML config format is deprecated. Please migrate to TOML.\n")
	return nil
}

func loadTOML(data []byte, cfg *Config) error {
	if err := toml.Unmarshal(data, cfg); err != nil {
		return NewErrorDetail(ErrorParseTOML, err.Error())
	}
	if !strings.Contains(string(data), "[dep_build]") {
		cfg.DepBuild.Enabled = true
	}
	return nil
}

func isConfigFilePath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".toml" || ext == ".yaml" || ext == ".yml"
}

func isTOMLPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".toml"
}

func (m *IsolationMode) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err == nil {
		return m.unmarshalValue(s)
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
	return NewErrorDetail(ErrorInvalidIsolation, "invalid isolation value")
}

func (m *IsolationMode) UnmarshalText(text []byte) error {
	return m.unmarshalValue(string(text))
}

func (m *IsolationMode) unmarshalValue(value string) error {
	s := strings.ToLower(strings.TrimSpace(value))
	switch s {
	case "", "none":
		*m = IsolationNone
	case "standard", "true":
		*m = IsolationStandard
	case "strict":
		*m = IsolationStrict
	default:
		return NewErrorDetail(ErrorInvalidIsolation, s)
	}
	return nil
}

type Flags struct {
	Asm []string `yaml:"asm" toml:"asm"`
	Cc  []string `yaml:"cc" toml:"cc"`
	Ld  []string `yaml:"ld" toml:"ld"`
}

type Hook struct {
	Cmd      string `yaml:"cmd" toml:"cmd"`
	Critical bool   `yaml:"critical" toml:"critical"`
}

type Hooks struct {
	PreBuild  []Hook `yaml:"pre_build" toml:"pre_build"`
	OnFailure string `yaml:"on_failure" toml:"on_failure"`
}

type BuildRule struct {
	Name    string   `yaml:"name" toml:"name"`
	Action  string   `yaml:"action" toml:"action"`
	Inputs  []string `yaml:"inputs" toml:"inputs"`
	Outputs []string `yaml:"outputs" toml:"outputs"`
	Depfile string   `yaml:"depfile" toml:"depfile"`
}

type ISOConfig struct {
	Enabled       bool     `yaml:"enabled" toml:"enabled"`
	SourceDir     string   `yaml:"source_dir" toml:"source_dir"`
	Output        string   `yaml:"output" toml:"output"`
	VolumeID      string   `yaml:"volume_id" toml:"volume_id"`
	BootImage     string   `yaml:"boot_image" toml:"boot_image"`
	BootCatalog   string   `yaml:"boot_catalog" toml:"boot_catalog"`
	BootLoadSize  string   `yaml:"boot_load_size" toml:"boot_load_size"`
	NoEmulBoot    bool     `yaml:"no_emul_boot" toml:"no_emul_boot"`
	BootInfoTable bool     `yaml:"boot_info_table" toml:"boot_info_table"`
	Joliet        bool     `yaml:"joliet" toml:"joliet"`
	RockRidge     bool     `yaml:"rock_ridge" toml:"rock_ridge"`
	Hybrid        bool     `yaml:"hybrid" toml:"hybrid"`
	CustomArgs    []string `yaml:"custom_args" toml:"custom_args"`
}

type PreprocessConfig struct {
	Enabled bool              `yaml:"enabled" toml:"enabled"`
	Inputs  []string          `yaml:"inputs" toml:"inputs"`
	Outputs []string          `yaml:"outputs" toml:"outputs"`
	Defines map[string]string `yaml:"defines" toml:"defines"`
}

type DepBuildConfig struct {
	Enabled      bool              `yaml:"enabled" toml:"enabled"`
	SkipTests    bool              `yaml:"skip_tests" toml:"skip_tests"`
	BuildTargets []string          `yaml:"build_targets" toml:"build_targets"`
	Outputs      []string          `yaml:"outputs" toml:"outputs"`
	Include      []string          `yaml:"include" toml:"include"`
	Environment  map[string]string `yaml:"environment" toml:"environment"`
	PreBuild     []string          `yaml:"pre_build" toml:"pre_build"`
	PostBuild    []string          `yaml:"post_build" toml:"post_build"`
	ExcludeFiles []string          `yaml:"exclude_files" toml:"exclude_files"`
	OnlyFiles    []string          `yaml:"only_files" toml:"only_files"`
}

type AutoBuildConfig struct {
	Enabled            bool              `yaml:"enabled" toml:"enabled"`
	LogLevel           string            `yaml:"log_level" toml:"log_level"`
	ContinueOnError    bool              `yaml:"continue_on_error" toml:"continue_on_error"`
	BuildOrder         []string          `yaml:"build_order" toml:"build_order"`
	DefaultSkipTests   bool              `yaml:"default_skip_tests" toml:"default_skip_tests"`
	DefaultEnvironment map[string]string `yaml:"default_environment" toml:"default_environment"`
}

type Config struct {
	Name    string `yaml:"name" toml:"name"`
	Profile string `yaml:"profile" toml:"profile"`
	Target  string `yaml:"target" toml:"target"`
	Sysroot string `yaml:"sysroot" toml:"sysroot"`

	SourceDir          string            `yaml:"source_dir" toml:"source_dir"`
	SourceDirs         []string          `yaml:"source_dirs" toml:"source_dirs"`
	SourceFiles        []string          `yaml:"source_files" toml:"source_files"`
	SourceFile         string            `yaml:"source_file" toml:"source_file"`
	Output             string            `yaml:"output" toml:"output"`
	OutObj             string            `yaml:"out_obj" toml:"out_obj"`
	Mode               string            `yaml:"mode" toml:"mode"`
	Toolchain          string            `yaml:"toolchain" toml:"toolchain"`
	Debug              bool              `yaml:"debug" toml:"debug"`
	Verbose            bool              `yaml:"verbose" toml:"verbose"`
	KeepObj            bool              `yaml:"keep_obj" toml:"keep_obj"`
	NoCache            bool              `yaml:"no_cache" toml:"no_cache"`
	CacheMode          CacheMode         `yaml:"cache_mode" toml:"cache_mode"`
	OptimizationLevel  int               `yaml:"optimization_level" toml:"optimization_level"`
	Exclude            []string          `yaml:"exclude" toml:"exclude"`
	Include            []string          `yaml:"include" toml:"include"`
	Scripts            []string          `yaml:"scripts" toml:"scripts"`
	Libs               []string          `yaml:"libs" toml:"libs"`
	AutoBuildDeps      bool              `yaml:"auto_build_deps" toml:"auto_build_deps"`
	IgnoreFile         string            `yaml:"ignore_file" toml:"ignore_file"`
	AuditIgnore        []string          `yaml:"audit_ignore" toml:"audit_ignore"`
	ToolChecksums      map[string]string `yaml:"tool_checksums" toml:"tool_checksums"`
	Variables          map[string]string `yaml:"variables" toml:"variables"`
	Flags              Flags             `yaml:"flags" toml:"flags"`
	Isolation          IsolationMode     `yaml:"isolation" toml:"isolation"`
	DeterministicStrip bool              `yaml:"deterministic_strip" toml:"deterministic_strip"`
	ToolchainSettings  struct {
		SearchPriority []string          `yaml:"search_priority" toml:"search_priority"`
		EnvAllow       []string          `yaml:"env_allow" toml:"env_allow"`
		ToolPaths      map[string]string `yaml:"tool_paths" toml:"tool_paths"`
	} `yaml:"toolchain_opts" toml:"toolchain_opts"`
	Preprocess PreprocessConfig `yaml:"preprocess" toml:"preprocess"`
	Hooks      Hooks            `yaml:"hooks" toml:"hooks"`
	BuildRules []BuildRule      `yaml:"build_rules" toml:"build_rules"`
	ISO        ISOConfig        `yaml:"iso" toml:"iso"`
	DepBuild   DepBuildConfig   `yaml:"dep_build" toml:"dep_build"`
	AutoBuild  AutoBuildConfig  `yaml:"auto_build" toml:"auto_build"`
}

func (c *Config) expand() {
	if c == nil {
		return
	}
	if c.Variables == nil {
		c.Variables = make(map[string]string, 4)
	}
	if _, ok := c.Variables["PWD"]; !ok {
		if wd, err := os.Getwd(); err == nil && wd != "" {
			c.Variables["PWD"] = wd
		}
	}
	if _, ok := c.Variables["HOME"]; !ok {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			c.Variables["HOME"] = home
		}
	}
	if _, ok := c.Variables["TARGET"]; !ok && c.Target != "" {
		c.Variables["TARGET"] = c.Target
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
	for i := range c.BuildRules {
		c.BuildRules[i].Action = variables.ExpandString(c.BuildRules[i].Action, vars)
		variables.ExpandSlice(c.BuildRules[i].Inputs, vars)
		variables.ExpandSlice(c.BuildRules[i].Outputs, vars)
		c.BuildRules[i].Depfile = variables.ExpandString(c.BuildRules[i].Depfile, vars)
	}
	c.Isolation = IsolationMode(variables.ExpandString(string(c.Isolation), vars))
	c.ISO.SourceDir = variables.ExpandString(c.ISO.SourceDir, vars)
	c.ISO.Output = variables.ExpandString(c.ISO.Output, vars)
	c.ISO.VolumeID = variables.ExpandString(c.ISO.VolumeID, vars)
	c.ISO.BootImage = variables.ExpandString(c.ISO.BootImage, vars)
	c.ISO.BootCatalog = variables.ExpandString(c.ISO.BootCatalog, vars)
	c.ISO.BootLoadSize = variables.ExpandString(c.ISO.BootLoadSize, vars)
	variables.ExpandSlice(c.ISO.CustomArgs, vars)
}

func (c *Config) fillDefaults() {
	if c == nil {
		return
	}
	if c.Mode == "" {
		c.Mode = "auto"
	}
	if c.Profile == "" {
		c.Profile = "balanced"
	}
	if !c.AutoBuildDeps {
		c.AutoBuildDeps = true
	}
	if c.Toolchain == "" {
		c.Toolchain = "auto"
	}
	if c.Isolation == "" {
		c.Isolation = IsolationNone
	}
	if c.IgnoreFile == "" {
		c.IgnoreFile = ".fzignore"
	}
	if c.CacheMode == "" {
		c.CacheMode = CacheModeDisk
	}
}

func splitOverrideList(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (c *Config) ApplySetOverrides(overrides []string) error {
	if c == nil {
		return nil
	}
	for _, raw := range overrides {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			return NewErrorDetail(ErrorInvalidOverride, entry)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if strings.HasPrefix(key, "variables.") {
			if c.Variables == nil {
				c.Variables = make(map[string]string)
			}
			c.Variables[strings.TrimPrefix(key, "variables.")] = value
			continue
		}
		if strings.HasPrefix(key, "toolchain_opts.") {
			nested := strings.TrimPrefix(key, "toolchain_opts.")
			switch nested {
			case "search_priority":
				c.ToolchainSettings.SearchPriority = splitOverrideList(value)
			case "env_allow":
				c.ToolchainSettings.EnvAllow = splitOverrideList(value)
			default:
				if strings.HasPrefix(nested, "tool_paths.") {
					if c.ToolchainSettings.ToolPaths == nil {
						c.ToolchainSettings.ToolPaths = make(map[string]string)
					}
					c.ToolchainSettings.ToolPaths[strings.TrimPrefix(nested, "tool_paths.")] = value
				} else {
					return NewErrorDetail(ErrorUnsupportedOverride, key)
				}
			}
			continue
		}
		if strings.HasPrefix(key, "hooks.") {
			switch strings.TrimPrefix(key, "hooks.") {
			case "on_failure":
				c.Hooks.OnFailure = value
			default:
				return NewErrorDetail(ErrorUnsupportedOverride, key)
			}
			continue
		}
		if strings.HasPrefix(key, "iso.") {
			switch strings.TrimPrefix(key, "iso.") {
			case "enabled":
				parsed, err := strconv.ParseBool(value)
				if err != nil {
					return NewErrorCause(ErrorInvalidOverride, "iso.enabled", err)
				}
				c.ISO.Enabled = parsed
			case "output":
				c.ISO.Output = value
			default:
				return NewErrorDetail(ErrorUnsupportedOverride, key)
			}
			continue
		}
		switch strings.ToLower(key) {
		case "name":
			c.Name = value
		case "profile":
			c.Profile = value
		case "target":
			c.Target = value
		case "sysroot":
			c.Sysroot = value
		case "source_dir":
			c.SourceDir = value
			c.SourceDirs = nil
			c.SourceFiles = nil
			c.SourceFile = ""
		case "source_dirs":
			c.SourceDirs = splitOverrideList(value)
			c.SourceDir = ""
			c.SourceFiles = nil
			c.SourceFile = ""
		case "source_file":
			c.SourceFile = value
			c.SourceDirs = nil
			c.SourceDir = ""
			c.SourceFiles = nil
		case "source_files":
			c.SourceFiles = splitOverrideList(value)
			c.SourceFile = ""
			c.SourceDirs = nil
			c.SourceDir = ""
		case "output":
			c.Output = value
		case "out_obj":
			c.OutObj = value
		case "mode":
			c.Mode = value
		case "toolchain":
			c.Toolchain = value
		case "debug":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return NewErrorDetail(ErrorInvalidOverride, "debug: "+err.Error())
			}
			c.Debug = parsed
		case "verbose":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return NewErrorDetail(ErrorInvalidOverride, "verbose: "+err.Error())
			}
			c.Verbose = parsed
		case "keep_obj":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return NewErrorDetail(ErrorInvalidOverride, "keep_obj: "+err.Error())
			}
			c.KeepObj = parsed
		case "no_cache":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return NewErrorDetail(ErrorInvalidOverride, "no_cache: "+err.Error())
			}
			c.NoCache = parsed
		case "optimization_level", "opt_level":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return NewErrorDetail(ErrorInvalidOverride, "optimization_level: "+err.Error())
			}
			c.OptimizationLevel = parsed
		case "cache_mode":
			var mode CacheMode
			if err := mode.UnmarshalText([]byte(value)); err != nil {
				return err
			}
			c.CacheMode = mode
		case "isolation":
			var mode IsolationMode
			if err := mode.UnmarshalText([]byte(value)); err != nil {
				return err
			}
			c.Isolation = mode
		case "ignore_file":
			c.IgnoreFile = value
		case "exclude":
			c.Exclude = splitOverrideList(value)
		case "include":
			c.Include = splitOverrideList(value)
		case "scripts":
			c.Scripts = splitOverrideList(value)
		case "libs":
			c.Libs = splitOverrideList(value)
		case "audit_ignore":
			c.AuditIgnore = splitOverrideList(value)
		case "flags.asm":
			c.Flags.Asm = splitOverrideList(value)
		case "flags.cc":
			c.Flags.Cc = splitOverrideList(value)
		case "flags.ld":
			c.Flags.Ld = splitOverrideList(value)
		default:
			return NewErrorDetail(ErrorUnsupportedOverride, key)
		}
	}
	return nil
}

func GenerateConfigH(templatePath, outputPath string, cfg *Config) error {
	if cfg == nil {
		return NewErrorDetail(ErrorInvalidConfig, "config is nil")
	}
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}
	content := expandConfigTemplate(string(data), cfg)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(content), 0o644)
}

func expandConfigTemplate(content string, cfg *Config) string {
	pattern := regexp.MustCompile(`\$\{([A-Za-z0-9_]+)\}`)
	return pattern.ReplaceAllStringFunc(content, func(token string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(token, "${"), "}")
		if value, ok := lookupConfigValue(cfg, name); ok {
			return value
		}
		if env, ok := os.LookupEnv(name); ok {
			return env
		}
		return token
	})
}

func lookupConfigValue(cfg *Config, name string) (string, bool) {
	if cfg == nil {
		return "", false
	}
	switch strings.ToUpper(name) {
	case "NAME":
		return cfg.Name, true
	case "PROFILE":
		return cfg.Profile, true
	case "TARGET":
		return cfg.Target, true
	case "SYSROOT":
		return cfg.Sysroot, true
	case "SOURCE_DIR":
		return cfg.SourceDir, true
	case "SOURCE_FILE":
		return cfg.SourceFile, true
	case "OUTPUT":
		return cfg.Output, true
	case "OUT_OBJ":
		return cfg.OutObj, true
	case "MODE":
		return cfg.Mode, true
	case "TOOLCHAIN":
		return cfg.Toolchain, true
	case "DEBUG":
		return strconv.FormatBool(cfg.Debug), true
	case "VERBOSE":
		return strconv.FormatBool(cfg.Verbose), true
	case "KEEP_OBJ":
		return strconv.FormatBool(cfg.KeepObj), true
	case "NO_CACHE":
		return strconv.FormatBool(cfg.NoCache), true
	case "CACHE_MODE":
		return cfg.CacheMode.String(), true
	case "OPTIMIZATION_LEVEL":
		return strconv.Itoa(cfg.OptimizationLevel), true
	case "ISOLATION":
		return cfg.Isolation.String(), true
	case "IGNORE_FILE":
		return cfg.IgnoreFile, true
	}
	if cfg.Variables != nil {
		if value, ok := cfg.Variables[name]; ok {
			return value, true
		}
		if value, ok := cfg.Variables[strings.ToUpper(name)]; ok {
			return value, true
		}
		if value, ok := cfg.Variables[strings.ToLower(name)]; ok {
			return value, true
		}
	}
	return "", false
}

func mergeStrings(dst, src []string) []string {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return append([]string(nil), src...)
	}
	seen := make(map[string]struct{}, len(dst)+len(src))
	out := make([]string, 0, len(dst)+len(src))
	for _, v := range dst {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	for _, v := range src {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func mergeStringMap(dst, src map[string]string) map[string]string {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[string]string, len(src))
	}
	for k, v := range src {
		if v == "" {
			continue
		}
		dst[k] = v
	}
	return dst
}

func Load(path string) (*Config, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, NewErrorDetail(ErrorFileStat, path+": "+err.Error())
	}
	if cfg, ok := loadConfigCache(path, fi); ok {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewErrorDetail(ErrorFileRead, path+": "+err.Error())
	}
	var cfg Config
	if err := loadConfigData(path, data, &cfg, make(map[string]struct{})); err != nil {
		return nil, err
	}
	cfg.expand()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	storeConfigCache(path, fi, &cfg)
	return &cfg, nil
}

func loadConfigData(path string, data []byte, cfg *Config, seen map[string]struct{}) error {
	if cfg == nil {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".toml":
		if err := loadTOML(data, cfg); err != nil {
			return err
		}
	case ".yaml", ".yml":
		if err := loadYAML(data, cfg); err != nil {
			return err
		}
	default:
		if err := loadTOML(data, cfg); err == nil {
			break
		} else if err := loadYAML(data, cfg); err == nil {
			break
		} else {
			return NewErrorDetail(ErrorParseTOML, "unknown config format")
		}
	}
	return resolveConfigIncludes(path, cfg, seen)
}

func resolveConfigIncludes(path string, cfg *Config, seen map[string]struct{}) error {
	if cfg == nil {
		return nil
	}
	absPath := path
	if !filepath.IsAbs(absPath) {
		var err error
		absPath, err = filepath.Abs(path)
		if err != nil {
			absPath = filepath.Clean(path)
		}
	}
	if seen == nil {
		seen = make(map[string]struct{})
	}
	if _, ok := seen[absPath]; ok {
		return NewErrorDetail(ErrorCyclicInclude, absPath)
	}
	seen[absPath] = struct{}{}
	defer delete(seen, absPath)

	var configIncludes []string
	var buildIncludes []string
	for _, raw := range cfg.Include {
		includePath := strings.TrimSpace(raw)
		if includePath == "" {
			continue
		}
		if isConfigFilePath(includePath) {
			configIncludes = append(configIncludes, includePath)
			continue
		}
		buildIncludes = append(buildIncludes, includePath)
	}
	if len(configIncludes) == 0 {
		return nil
	}

	merged := Config{}
	for _, includePath := range configIncludes {
		childPath := includePath
		if !filepath.IsAbs(childPath) {
			childPath = filepath.Join(filepath.Dir(absPath), includePath)
		}
		data, err := os.ReadFile(childPath)
		if err != nil {
			return NewErrorDetail(ErrorIncludeRead, childPath+": "+err.Error())
		}
		var childCfg Config
		if err := loadConfigData(childPath, data, &childCfg, seen); err != nil {
			return err
		}
		merged.Merge(&childCfg)
	}

	raw := *cfg
	raw.Include = buildIncludes
	merged.Merge(&raw)
	*cfg = merged
	return nil
}

func (c *Config) Validate() error {
	if c == nil {
		return nil
	}
	c.fillDefaults()
	if c.SourceDir != "" && len(c.SourceDirs) > 0 {
		return NewError(ErrorInvalidSourceConfig)
	}
	if c.SourceFile != "" && len(c.SourceFiles) > 0 {
		return NewError(ErrorInvalidSourceConfig)
	}
	if c.Mode != "auto" && c.Mode != "c" && c.Mode != "raw" {
		return NewErrorDetail(ErrorInvalidMode, c.Mode)
	}
	c.Profile = strings.TrimSpace(strings.ToLower(c.Profile))
	if c.Profile != "balanced" && c.Profile != "powered" && c.Profile != "performance" {
		return NewErrorDetail(ErrorInvalidProfile, c.Profile)
	}
	c.Toolchain = strings.TrimSpace(strings.ToLower(c.Toolchain))
	if _, ok := supportedToolchains[c.Toolchain]; !ok {
		return NewErrorDetail(ErrorInvalidToolchain, c.Toolchain)
	}
	c.Target = strings.TrimSpace(c.Target)
	switch c.Isolation {
	case IsolationNone, IsolationStandard, IsolationStrict:
	default:
		return NewErrorDetail(ErrorInvalidIsolation, string(c.Isolation))
	}
	switch c.CacheMode {
	case CacheModeDisk, CacheModeRAM, CacheModeOff, "":
	default:
		return NewErrorDetail(ErrorInvalidCacheMode, string(c.CacheMode))
	}
	if c.NoCache {
		c.CacheMode = CacheModeOff
	}
	if c.IgnoreFile == "" {
		c.IgnoreFile = ".fzignore"
	}
	if len(c.BuildRules) > 0 {
		outputs := make(map[string]struct{}, len(c.BuildRules)*2)
		for _, rule := range c.BuildRules {
			if rule.Action == "" {
				return NewError(ErrorBuildRuleActionRequired)
			}
			if len(rule.Outputs) == 0 {
				return NewError(ErrorBuildRuleOutputsRequired)
			}
			for _, out := range rule.Outputs {
				key := filepath.Clean(out)
				if _, ok := outputs[key]; ok {
					return NewErrorDetail(ErrorDuplicateBuildRuleOutput, out)
				}
				outputs[key] = struct{}{}
			}
		}
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
	if c == nil || other == nil {
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
	if other.SourceFile != "" {
		c.SourceFile = other.SourceFile
		c.SourceDir = ""
		c.SourceDirs = nil
		c.SourceFiles = nil
	}
	if len(other.SourceFiles) > 0 {
		if c.SourceFile != "" {
			c.SourceFiles = mergeStrings(c.SourceFiles, []string{c.SourceFile})
		}
		c.SourceDir = ""
		c.SourceFile = ""
		c.SourceFiles = mergeStrings(c.SourceFiles, other.SourceFiles)
	}
	if other.SourceDir != "" {
		c.SourceDir = other.SourceDir
		c.SourceDirs = nil
		c.SourceFiles = nil
		c.SourceFile = ""
	}
	if len(other.SourceDirs) > 0 {
		if c.SourceDir != "" {
			c.SourceDirs = append([]string{c.SourceDir}, c.SourceDirs...)
		}
		c.SourceDir = ""
		c.SourceFile = ""
		c.SourceDirs = mergeStrings(c.SourceDirs, other.SourceDirs)
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
		c.Debug = true
	}
	if other.Verbose {
		c.Verbose = true
	}
	if other.KeepObj {
		c.KeepObj = true
	}
	if other.NoCache {
		c.NoCache = true
		c.CacheMode = CacheModeOff
	}
	if other.CacheMode != "" {
		c.CacheMode = other.CacheMode
	}
	if other.Isolation != "" {
		c.Isolation = other.Isolation
	}
	if len(other.Exclude) > 0 {
		c.Exclude = mergeStrings(c.Exclude, other.Exclude)
	}
	if len(other.Include) > 0 {
		c.Include = mergeStrings(c.Include, other.Include)
	}
	if len(other.Libs) > 0 {
		c.Libs = mergeStrings(c.Libs, other.Libs)
	}
	if other.IgnoreFile != "" {
		c.IgnoreFile = other.IgnoreFile
	}
	if len(other.AuditIgnore) > 0 {
		c.AuditIgnore = mergeStrings(c.AuditIgnore, other.AuditIgnore)
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
		c.Flags.Asm = mergeStrings(c.Flags.Asm, other.Flags.Asm)
	}
	if len(other.Flags.Cc) > 0 {
		c.Flags.Cc = mergeStrings(c.Flags.Cc, other.Flags.Cc)
	}
	if len(other.Flags.Ld) > 0 {
		c.Flags.Ld = mergeStrings(c.Flags.Ld, other.Flags.Ld)
	}
	if other.OptimizationLevel > 0 {
		c.OptimizationLevel = other.OptimizationLevel
	}
	if len(other.Scripts) > 0 {
		c.Scripts = mergeStrings(c.Scripts, other.Scripts)
	}
	if len(other.Variables) > 0 {
		c.Variables = mergeStringMap(c.Variables, other.Variables)
	}
	if len(other.ToolchainSettings.SearchPriority) > 0 {
		c.ToolchainSettings.SearchPriority = mergeStrings(c.ToolchainSettings.SearchPriority, other.ToolchainSettings.SearchPriority)
	}
	if len(other.ToolchainSettings.EnvAllow) > 0 {
		c.ToolchainSettings.EnvAllow = mergeStrings(c.ToolchainSettings.EnvAllow, other.ToolchainSettings.EnvAllow)
	}
	if len(other.ToolchainSettings.ToolPaths) > 0 {
		if c.ToolchainSettings.ToolPaths == nil {
			c.ToolchainSettings.ToolPaths = make(map[string]string, len(other.ToolchainSettings.ToolPaths))
		}
		for k, v := range other.ToolchainSettings.ToolPaths {
			if v == "" {
				continue
			}
			c.ToolchainSettings.ToolPaths[k] = v
		}
	}
	if len(other.Hooks.PreBuild) > 0 {
		c.Hooks.PreBuild = append(c.Hooks.PreBuild, other.Hooks.PreBuild...)
	}
	if len(other.BuildRules) > 0 {
		c.BuildRules = append(c.BuildRules, other.BuildRules...)
	}
	if other.Hooks.OnFailure != "" {
		c.Hooks.OnFailure = other.Hooks.OnFailure
	}
	if other.ISO.Enabled {
		c.ISO.Enabled = true
	}
	c.mergeISO(&other.ISO)
}

func (c *Config) mergeISO(other *ISOConfig) {
	if other.Enabled {
		c.ISO.Enabled = true
	}
	if other.SourceDir != "" {
		c.ISO.SourceDir = other.SourceDir
	}
	if other.Output != "" {
		c.ISO.Output = other.Output
	}
	if other.VolumeID != "" {
		c.ISO.VolumeID = other.VolumeID
	}
	if other.BootImage != "" {
		c.ISO.BootImage = other.BootImage
	}
	if other.BootCatalog != "" {
		c.ISO.BootCatalog = other.BootCatalog
	}
	if other.BootLoadSize != "" {
		c.ISO.BootLoadSize = other.BootLoadSize
	}
	if other.NoEmulBoot {
		c.ISO.NoEmulBoot = true
	}
	if other.BootInfoTable {
		c.ISO.BootInfoTable = true
	}
	if other.Joliet {
		c.ISO.Joliet = true
	}
	if other.RockRidge {
		c.ISO.RockRidge = true
	}
	if other.Hybrid {
		c.ISO.Hybrid = true
	}
	if len(other.CustomArgs) > 0 {
		c.ISO.CustomArgs = mergeStrings(c.ISO.CustomArgs, other.CustomArgs)
	}
}

func FindConfigs() (system, user, local string) {
	systemPaths := []string{"/etc/github.com/forgezero-cli/ForgeZero/config.toml", "/etc/fz.toml", "/etc/github.com/forgezero-cli/ForgeZero/config.yaml", "/etc/fz.yaml"}
	for _, p := range systemPaths {
		if _, err := os.Stat(p); err == nil {
			system = p
			break
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		userPaths := []string{
			filepath.Join(home, ".config", "fz", "config.toml"),
			filepath.Join(home, ".fz.toml"),
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
	localPaths := []string{".fz.toml", "fz.toml", ".fz.yaml", "fz.yaml", ".fz.yml", "fz.yml"}
	for _, p := range localPaths {
		if _, err := os.Stat(p); err == nil {
			local = p
			break
		}
	}
	return
}

func loadConfigPath(path string, idx int, out chan<- loadResult, wg *sync.WaitGroup) {
	defer wg.Done()
	cfg, err := Load(path)
	out <- loadResult{idx: idx, cfg: cfg, err: err}
}

type loadResult struct {
	idx int
	cfg *Config
	err error
}

func LoadMerged(explicitPath string) (*Config, error) {
	if explicitPath == "" {
		if env := os.Getenv("FZ_CONFIG_PATH"); env != "" {
			explicitPath = env
		} else if env := os.Getenv("FZ_CONFIG"); env != "" {
			explicitPath = env
		}
	}
	var cfg Config
	if explicitPath != "" {
		explicitCfg, err := Load(explicitPath)
		if err != nil {
			return nil, err
		}
		cfg.Merge(explicitCfg)
		cfg.expand()
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return &cfg, nil
	}
	systemPath, userPath, localPath := FindConfigs()
	paths := make([]string, 0, 3)
	for _, path := range []string{systemPath, userPath, localPath} {
		if path != "" {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		cfg.expand()
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	results := make([]*Config, len(paths))
	out := make(chan loadResult, len(paths))
	var wg sync.WaitGroup
	wg.Add(len(paths))
	for idx, path := range paths {
		go loadConfigPath(path, idx, out, &wg)
	}
	wg.Wait()
	close(out)

	var loadErr error
	for result := range out {
		if result.err != nil {
			loadErr = errors.Join(loadErr, result.err)
			continue
		}
		results[result.idx] = result.cfg
	}
	if loadErr != nil {
		return nil, loadErr
	}

	for _, loaded := range results {
		if loaded != nil {
			cfg.Merge(loaded)
		}
	}
	cfg.expand()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func DefaultConfigPath() string {
	if env := os.Getenv("FZ_CONFIG_PATH"); env != "" {
		return env
	}
	if env := os.Getenv("FZ_CONFIG"); env != "" {
		return env
	}
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
		return nil, NewErrorDetail(ErrorMissingSource, "scan found no source files")
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
