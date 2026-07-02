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
	"sync"

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
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "", "disk":
		*m = CacheModeDisk
	case "ram":
		*m = CacheModeRAM
	case "off":
		*m = CacheModeOff
	default:
		return errors.New("invalid cache_mode: " + s)
	}
	return nil
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

type ISOConfig struct {
	Enabled       bool     `yaml:"enabled"`
	SourceDir     string   `yaml:"source_dir"`
	Output        string   `yaml:"output"`
	VolumeID      string   `yaml:"volume_id"`
	BootImage     string   `yaml:"boot_image"`
	BootCatalog   string   `yaml:"boot_catalog"`
	BootLoadSize  string   `yaml:"boot_load_size"`
	NoEmulBoot    bool     `yaml:"no_emul_boot"`
	BootInfoTable bool     `yaml:"boot_info_table"`
	Joliet        bool     `yaml:"joliet"`
	RockRidge     bool     `yaml:"rock_ridge"`
	Hybrid        bool     `yaml:"hybrid"`
	CustomArgs    []string `yaml:"custom_args"`
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
	CacheMode          CacheMode         `yaml:"cache_mode"`
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
	Hooks Hooks     `yaml:"hooks"`
	ISO   ISOConfig `yaml:"iso"`
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("cannot read config file " + path + ": " + err.Error())
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.New("cannot parse YAML: " + err.Error())
	}
	cfg.expand()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c == nil {
		return nil
	}
	c.fillDefaults()
	if c.SourceDir != "" && len(c.SourceDirs) > 0 {
		return errors.New("cannot set both source_dir and source_dirs")
	}
	if c.SourceFile != "" && len(c.SourceFiles) > 0 {
		return errors.New("cannot set both source_file and source_files")
	}
	if c.Mode != "auto" && c.Mode != "c" && c.Mode != "raw" {
		return errors.New("invalid mode: " + c.Mode)
	}
	c.Profile = strings.TrimSpace(strings.ToLower(c.Profile))
	if c.Profile != "balanced" && c.Profile != "powered" && c.Profile != "performance" {
		return errors.New("invalid profile: " + c.Profile)
	}
	c.Toolchain = strings.TrimSpace(strings.ToLower(c.Toolchain))
	if _, ok := supportedToolchains[c.Toolchain]; !ok {
		return errors.New("invalid toolchain: " + c.Toolchain)
	}
	c.Target = strings.TrimSpace(c.Target)
	switch c.Isolation {
	case IsolationNone, IsolationStandard, IsolationStrict:
	default:
		return errors.New("invalid isolation: " + string(c.Isolation))
	}
	switch c.CacheMode {
	case CacheModeDisk, CacheModeRAM, CacheModeOff, "":
	default:
		return errors.New("invalid cache_mode: " + string(c.CacheMode))
	}
	if c.NoCache {
		c.CacheMode = CacheModeOff
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
