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

package builder

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/fzp"

	"github.com/forgezero-cli/ForgeZero/internal/assembler"
	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/drivers/scheduler"
	"github.com/forgezero-cli/ForgeZero/internal/ignore"
	"github.com/forgezero-cli/ForgeZero/internal/linker"
	"github.com/forgezero-cli/ForgeZero/internal/seal"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

type BuildResult struct {
	ObjectFiles []string
	Binary      string
	ObjDir      string
	CacheDir    string
}

type pair struct {
	src string
	obj string
}

func matchExclude(path string, excludes []string) bool {
	for _, pattern := range excludes {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}
	return false
}

func supportedSourceInclude(ext string) bool {
	switch strings.ToLower(ext) {
	case ".c", ".cpp", ".cc", ".cxx", ".m", ".mm", ".s", ".S", ".asm", ".fasm":
		return true
	}
	return false
}

func targetOSFromTriple(target string) string {
	t := strings.ToLower(strings.TrimSpace(target))
	if t == "" {
		return ""
	}
	switch {
	case strings.Contains(t, "windows"):
		return "windows"
	case strings.Contains(t, "darwin") || strings.Contains(t, "apple"):
		return "darwin"
	case strings.Contains(t, "freebsd"):
		return "freebsd"
	case strings.Contains(t, "netbsd"):
		return "netbsd"
	case strings.Contains(t, "openbsd"):
		return "openbsd"
	case strings.Contains(t, "android"):
		return "android"
	case strings.Contains(t, "solaris") || strings.Contains(t, "illumos") || strings.Contains(t, "sunos"):
		return "solaris"
	case strings.Contains(t, "linux"):
		return "linux"
	case strings.Contains(t, "wasi") || strings.Contains(t, "wasm"):
		return "wasm"
	default:
		return ""
	}
}

func configSpecifiesBuild(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	if cfg.ConfigOnly || cfg.ParseMakefile {
		return true
	}
	if cfg.SourceFile != "" || cfg.SourceDir != "" || len(cfg.SourceDirs) > 0 {
		return true
	}
	if len(cfg.SourceFiles) > 0 || len(cfg.BuildRules) > 0 {
		return true
	}
	return false
}

func skipPlatformSource(path, targetOS string) bool {
	normalized := strings.ToLower(filepath.ToSlash(path))

	if strings.Contains(normalized, "ngx_linux_") {
		return targetOS != "linux"
	}
	if strings.Contains(normalized, "ngx_darwin_") {
		return targetOS != "darwin"
	}
	if strings.Contains(normalized, "ngx_freebsd_") {
		return targetOS != "freebsd"
	}
	if strings.Contains(normalized, "ngx_openbsd_") {
		return targetOS != "openbsd"
	}
	if strings.Contains(normalized, "ngx_netbsd_") {
		return targetOS != "netbsd"
	}
	if strings.Contains(normalized, "ngx_solaris_") || strings.Contains(normalized, "ngx_sunos_") {
		return targetOS != "solaris"
	}
	if strings.Contains(normalized, "/os/win32/") || strings.Contains(normalized, "/win32/") || strings.Contains(normalized, "/windows/") || strings.Contains(normalized, "ngx_win32_") {
		return targetOS != "windows"
	}
	if strings.Contains(normalized, "/os/unix/") || strings.Contains(normalized, "/unix/") || strings.Contains(normalized, "/posix/") {
		return targetOS == "windows"
	}
	if strings.Contains(normalized, "/os/darwin/") || strings.Contains(normalized, "/darwin/") || strings.Contains(normalized, "/macos/") || strings.Contains(normalized, "/osx/") {
		return targetOS != "darwin"
	}
	if strings.Contains(normalized, "/freebsd/") {
		return targetOS != "freebsd"
	}
	if strings.Contains(normalized, "/netbsd/") {
		return targetOS != "netbsd"
	}
	if strings.Contains(normalized, "/openbsd/") {
		return targetOS != "openbsd"
	}
	if strings.Contains(normalized, "/android/") {
		return targetOS != "android"
	}
	if strings.Contains(normalized, "/solaris/") || strings.Contains(normalized, "/sunos/") || strings.Contains(normalized, "/illumos/") {
		return targetOS != "solaris"
	}
	if strings.Contains(normalized, "/wasi/") {
		return targetOS != "wasm"
	}
	return false
}

func filterPlatformSpecificSources(srcFiles []string, target string) []string {
	targetOS := targetOSFromTriple(target)
	if targetOS == "" {
		return srcFiles
	}
	filtered := make([]string, 0, len(srcFiles))
	for _, src := range srcFiles {
		if skipPlatformSource(src, targetOS) {
			continue
		}
		filtered = append(filtered, src)
	}
	return filtered
}

func findIncludedSourceFiles(srcFiles []string) map[string]struct{} {
	included := make(map[string]struct{})
	for _, src := range srcFiles {
		ext := strings.ToLower(filepath.Ext(src))
		if !supportedSourceInclude(ext) {
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		currentDir := filepath.Dir(src)
		pos := 0
		for {
			idx := strings.Index(string(data[pos:]), "include")
			if idx == -1 {
				break
			}
			start := pos + idx
			if start == 0 || data[start-1] != '#' {
				pos = start + len("include")
				continue
			}
			i := start + len("include")
			for i < len(data) && (data[i] == ' ' || data[i] == '\t') {
				i++
			}

			if i >= len(data) || data[i] != '"' {
				pos = start + len("include")
				continue
			}
			begin := i + 1
			end := begin
			for end < len(data) && data[end] != '"' {
				end++
			}
			if end >= len(data) {
				break
			}
			includePath := string(data[begin:end])
			if supportedSourceInclude(strings.ToLower(filepath.Ext(includePath))) {
				resolved := filepath.Clean(filepath.Join(currentDir, includePath))
				if abs, err := filepath.Abs(resolved); err == nil {
					resolved = abs
				}
				if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
					included[resolved] = struct{}{}
				}
			}
			pos = end + 1
		}
	}
	return included
}

func discoverDependencyIncludeDirs(rootDir string) []string {
	var result []string
	parent := filepath.Dir(rootDir)
	if parent == rootDir {
		return nil
	}
	addDir := func(path string) {
		if path == "" {
			return
		}
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			for _, existing := range result {
				if existing == path {
					return
				}
			}
			result = append(result, path)
		}
	}
	addIfHasHeaders := func(path string) {
		if path == "" {
			return
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".h" || ext == ".hpp" {
				addDir(path)
				return
			}
		}
	}

	addIfHasHeaders(parent)
	depsDir := filepath.Join(parent, "deps")
	if info, err := os.Stat(depsDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(depsDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				child := filepath.Join(depsDir, entry.Name())
				addIfHasHeaders(child)
				addIfHasHeaders(filepath.Join(child, "include"))
				addIfHasHeaders(filepath.Join(child, "src"))
			}
		}
	}
	return result
}

func isHeaderFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".h" || ext == ".hpp"
}

func discoverSourceIncludeDirs(rootDir string) []string {
	srcDir := filepath.Join(rootDir, "src")
	if info, err := os.Stat(srcDir); err != nil || !info.IsDir() {
		return nil
	}

	var result []string
	addDir := func(path string) {
		if path == "" {
			return
		}
		for _, existing := range result {
			if existing == path {
				return
			}
		}
		result = append(result, path)
	}

	_ = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if isHeaderFile(entry.Name()) {
				addDir(path)
				break
			}
		}
		return nil
	})
	return result
}

func RunHooks(ctx context.Context, hooks []config.Hook) error {
	for _, h := range hooks {
		if h.Cmd == "" {
			continue
		}
		name, args := utils.ShellCommand(h.Cmd)
		_, err := utils.RunCommand(ctx, false, nil, nil, name, args...)
		if err != nil {
			if h.Critical {
				return errors.New("hook failed (critical): " + err.Error())
			}
			return errors.New("hook failed: " + err.Error())
		}
	}
	return nil
}

func BuildDir(ctx context.Context, dirs []string, outBin string, debug, verbose bool, mode string, keepObj, noCache, noSymbolCheck, sanitize, strict bool, exclude, sourceFiles []string, ignoreMatcher interface{}, includes, libs []string, jobs int, buildType string) (*BuildResult, error) {
	cfg := utils.ConfigFromContext(ctx)
	if len(dirs) > 0 {
		localPaths := []string{filepath.Join(dirs[0], "fz.toml"), filepath.Join(dirs[0], ".fz.toml")}
		for _, lp := range localPaths {
			if info, err := os.Stat(lp); err == nil && !info.IsDir() {
				if localCfg, err := config.Load(lp); err == nil {
					if cfg == nil {
						cfg = localCfg
					} else {
						merged := *cfg
						merged.Merge(localCfg)
						cfg = &merged
					}
					break
				}
			}
		}
	}

	if cfg != nil && len(cfg.Hooks.PreBuild) > 0 {
		if err := RunHooks(ctx, cfg.Hooks.PreBuild); err != nil {
			return nil, err
		}
	}
	var res *BuildResult
	var err error
	if cfg != nil && cfg.Hooks.OnFailure != "" {
		defer func() {
			if err != nil {
				name, args := utils.ShellCommand(cfg.Hooks.OnFailure)
				_, _ = utils.RunCommand(context.Background(), false, nil, nil, name, args...)
			}
		}()
	}
	res, err = buildDirInner(ctx, cfg, dirs, outBin, debug, verbose, mode, keepObj, noCache, noSymbolCheck, sanitize, strict, exclude, sourceFiles, ignoreMatcher, includes, libs, jobs, buildType)
	return res, err
}

func buildDirInner(ctx context.Context, cfg *config.Config, dirs []string, outBin string, debug, verbose bool, mode string, keepObj, noCache, noSymbolCheck, sanitize, strict bool, exclude, sourceFiles []string, ignoreMatcher interface{}, includes, libs []string, jobs int, buildType string) (*BuildResult, error) {
	ApplyHostDetection(cfg)
	if cfg == nil {
		SetRAMCacheCapacityMB(0)
	} else {
		SetRAMCacheCapacityMB(cfg.CacheRAMMB)
	}
	localCfgLoaded := false
	if len(dirs) > 0 {
		localPaths := []string{filepath.Join(dirs[0], "fz.toml"), filepath.Join(dirs[0], ".fz.toml")}
		for _, lp := range localPaths {
			if info, err := os.Stat(lp); err == nil && !info.IsDir() {
				localCfgLoaded = true
				break
			}
		}
	}
	jobs = AdjustJobs(jobs)
	collectedDepLdFlags := make([]string, 0)

	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	rootDir, err := filepath.Abs(dirs[0])
	if err != nil {
		return nil, err
	}
	rootDir = filepath.Clean(rootDir)
	matcherPath := func(p string) string {
		for _, d := range dirs {
			if d == "" {
				continue
			}
			if rel, err := filepath.Rel(d, p); err == nil {
				if strings.HasPrefix(rel, "..") {
					continue
				}
				return filepath.ToSlash(rel)
			}
		}
		return filepath.ToSlash(p)
	}

	for _, dir := range dirs {
		if err := utils.EnsureInsideRoot(rootDir, dir); err != nil {
			return nil, err
		}
	}
	if outBin == "" && cfg != nil && cfg.Output != "" {
		outBin = cfg.Output
	}
	if outBin == "" {
		if len(dirs) == 1 {
			base := filepath.Base(dirs[0])
			if utils.IsWindows() {
				outBin = base + ".exe"
			} else {
				outBin = base + ".out"
			}
		} else {
			outBin = "fz_build"
			if utils.IsWindows() {
				outBin += ".exe"
			}
		}
	}
	if info, err := os.Stat(outBin); err == nil && info.IsDir() {
		return nil, errors.New("output path is a directory: " + outBin)
	}
	if err := utils.EnsureDir(outBin); err != nil {
		return nil, errors.New("cannot create output directory: " + err.Error())
	}

	if len(dirs) > 0 {
		cfgPath := filepath.Join(dirs[0], "configure.fz")
		if info, err := os.Stat(cfgPath); err == nil && !info.IsDir() {
			data, _ := os.ReadFile(cfgPath)
			proc := fzp.NewProcessor(fzp.Options{RootDir: dirs[0]})
			if defs, err := proc.ParseDefinitions(string(data)); err == nil {
				if cfg == nil {
					cfg = &config.Config{}
				}
				output := cfg.Output
				if v, ok := defs["OUTPUT"]; ok && v != "" {
					output = v
				}
				if output == "" {
					output = outBin
				}
				action := "make"
				if v, ok := defs["ACTION"]; ok && strings.TrimSpace(v) != "" {
					action = v
				} else if v, ok := defs["MAKECMD"]; ok && strings.TrimSpace(v) != "" {
					action = v
				}
				rule := config.BuildRule{
					Name:    "configure.fz:build",
					Action:  action,
					Inputs:  nil,
					Outputs: []string{output},
				}
				cfg.BuildRules = append([]config.BuildRule{rule}, cfg.BuildRules...)
			}
		}
	}

	if cfg != nil && len(cfg.BuildRules) > 0 {
		return runBuildRules(ctx, cfg, verbose, jobs)
	}

	if !localCfgLoaded {
		defaultEx := []string{"t/*", "test/*", "tests/*", "unit-tests/*", "tools/*", "examples/*"}
		exclude = append(exclude, defaultEx...)
	}

	var srcFiles []string
	autoDiscoveredSources := false

	if len(sourceFiles) > 0 {
		srcFiles = append(srcFiles, sourceFiles...)
	} else if len(dirs) > 0 {
		if localCfgLoaded && cfg != nil && len(cfg.SourceFiles) > 0 {
			for _, sf := range cfg.SourceFiles {
				if !filepath.IsAbs(sf) {
					srcFiles = append(srcFiles, filepath.Join(dirs[0], sf))
				} else {
					srcFiles = append(srcFiles, sf)
				}
			}
		}

		if cfg != nil && cfg.ConfigOnly && len(srcFiles) == 0 {
			return nil, errors.New("config-only mode requires SourceFiles or BuildRules in fz.toml or configure.fz")
		}

		if cfg != nil && cfg.ParseMakefile && !cfg.ConfigOnly {
			if discoveredSources, err := discoverMakefileSources(dirs[0]); err == nil && len(discoveredSources) > 0 {
				srcFiles = append(srcFiles, discoveredSources...)
				autoDiscoveredSources = true
			}
		}

		if len(srcFiles) == 0 {
			autoDiscoveredSources = true
			walkRoots := dirs
			if !localCfgLoaded {
				walkRoots = []string{}
				for _, d := range dirs {
					walkRoots = append(walkRoots, d)
					cands := []string{"src", "include", "lib", "cmd", filepath.Join("src", "core")}
					for _, s := range cands {
						p := filepath.Join(d, s)
						if info, err := os.Stat(p); err == nil && info.IsDir() {
							walkRoots = append(walkRoots, p)
						}
					}
				}
			}
			for _, dir := range walkRoots {
				err := utils.Walk(dir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if info.IsDir() {
						name := info.Name()
						shouldSkip := name == ".git" || name == ".svn" || name == "node_modules" || matchExclude(path, exclude)
						if !localCfgLoaded {
							if name == "t" || name == "test" || name == "tests" || name == "unit-tests" || name == "tools" || name == "examples" || name == "po" || name == "trace2" {
								shouldSkip = true
							}
						}
						if !shouldSkip && ignoreMatcher != nil {
							if m, ok := ignoreMatcher.(*ignore.IgnoreMatcher); ok && m != nil {
								shouldSkip = m.Match(matcherPath(path))
							}
						}
						if shouldSkip {
							if verbose {
								os.Stdout.WriteString("Skipping directory tree: " + path + "\n")
							}
							return filepath.SkipDir
						}
						return nil
					}
					shouldExclude := matchExclude(path, exclude)
					if !shouldExclude && ignoreMatcher != nil {
						if m, ok := ignoreMatcher.(*ignore.IgnoreMatcher); ok && m != nil {
							p := matcherPath(path)
							shouldExclude = m.Match(p)
							if verbose && shouldExclude {
								os.Stdout.WriteString("Ignored by matcher: " + path + "\n")
							} else if verbose && strings.Contains(filepath.Base(path), "cli_commands") {
								os.Stdout.WriteString("NOT matched by ignore: " + path + "\n")
							}
						}
					}
					if shouldExclude {
						if verbose {
							os.Stdout.WriteString("Excluding file: " + path + "\n")
						}
						return nil
					}
					ext := strings.ToLower(filepath.Ext(path))
					if utils.SupportedExtension(ext) {
						srcFiles = append(srcFiles, path)
					}
					return nil
				})
				if err != nil {
					return nil, errors.New("walk error in " + dir + ": " + err.Error())
				}
			}
		}
	}

	if autoDiscoveredSources && len(srcFiles) > 0 {
		if cfg != nil {
			target := cfg.Target
			if target == "" {
				target = assembler.Target
			}
			filtered := filterPlatformSpecificSources(srcFiles, target)
			if len(filtered) != len(srcFiles) {
				srcFiles = filtered
			}
		}
		included := findIncludedSourceFiles(srcFiles)
		if len(included) > 0 {
			filtered := make([]string, 0, len(srcFiles))
			for _, src := range srcFiles {
				abs, err := filepath.Abs(src)
				if err != nil {
					filtered = append(filtered, src)
					continue
				}
				if _, ok := included[filepath.Clean(abs)]; ok {
					if verbose {
						os.Stdout.WriteString("Skipping included source file: " + src + "\n")
					}
					continue
				}
				filtered = append(filtered, src)
			}
			srcFiles = filtered
		}
		if ignoreMatcher != nil {
			if m, ok := ignoreMatcher.(*ignore.IgnoreMatcher); ok && m != nil {
				filtered := make([]string, 0, len(srcFiles))
				for _, src := range srcFiles {
					if !m.Match(matcherPath(src)) {
						filtered = append(filtered, src)
					} else if verbose {
						os.Stdout.WriteString("Ignoring file (from .fzignore): " + src + "\n")
					}
				}
				srcFiles = filtered
			}
		}
	}

	objDir := joinPath(filepath.Dir(outBin), ".fz_objs")
	generatedIncludeDir := joinPath(objDir, "include")
	if err := utils.SecureMkdirAll(objDir); err != nil {
		return nil, errors.New("cannot create obj dir: " + err.Error())
	}
	if err := utils.SecureMkdirAll(generatedIncludeDir); err != nil {
		return nil, errors.New("cannot create generated include dir: " + err.Error())
	}
	includeDirs := []string{generatedIncludeDir}
	if cfg != nil {
		includeDirs = append(includeDirs, cfg.Include...)
	}
	includeDirs = append(includeDirs, includes...)
	if cfg != nil && len(cfg.Libs) > 0 {
		seen := make(map[string]struct{}, len(libs)+len(cfg.Libs))
		out := make([]string, 0, len(libs)+len(cfg.Libs))
		for _, l := range libs {
			if l == "" {
				continue
			}
			seen[l] = struct{}{}
			out = append(out, l)
		}
		for _, l := range cfg.Libs {
			if l == "" {
				continue
			}
			if _, ok := seen[l]; ok {
				continue
			}
			seen[l] = struct{}{}
			out = append(out, l)
		}
		libs = out
	}
	if len(dirs) > 0 {
		includeDirs = append(includeDirs, discoverDependencyIncludeDirs(dirs[0])...)
	}

	if len(dirs) > 0 {
		cand := []string{filepath.Join(dirs[0], "objs"), filepath.Join(dirs[0], "auto")}
		for _, c := range cand {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				if a, err := filepath.Abs(c); err == nil {
					includeDirs = append(includeDirs, a)
				} else {
					includeDirs = append(includeDirs, c)
				}
			}
		}
	}

	if len(dirs) > 0 {
		cands := []string{dirs[0], filepath.Join(dirs[0], "src"), filepath.Join(dirs[0], "src", "core")}
		for _, c := range cands {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				if a, err := filepath.Abs(c); err == nil {
					includeDirs = append(includeDirs, a)
				} else {
					includeDirs = append(includeDirs, c)
				}
			}
		}
	}

	if len(dirs) > 0 && cfg != nil && cfg.ParseMakefile {
		discoveredIncludes, discoveredCflags, discoveredLdflags := discoverMakefileSettings(dirs[0])
		if len(discoveredIncludes) > 0 {
			includeDirs = append(includeDirs, discoveredIncludes...)
		}
		if len(discoverSourceIncludeDirs(dirs[0])) > 0 {
			includeDirs = append(includeDirs, discoverSourceIncludeDirs(dirs[0])...)
		}
		if strings.TrimSpace(discoveredCflags) != "" {
			if assembler.CcFlags == "" {
				assembler.CcFlags = discoveredCflags
			} else {
				assembler.CcFlags = joinFlags(assembler.CcFlags, discoveredCflags)
			}
			assembler.CcFLagsParsed = strings.Fields(assembler.CcFlags)
		}
		if strings.TrimSpace(discoveredLdflags) != "" {
			if linker.LdFlags == "" {
				linker.LdFlags = discoveredLdflags
			} else {
				linker.LdFlags = joinFlags(linker.LdFlags, discoveredLdflags)
			}
		}
	}

	var globalAutoBuild *config.AutoBuildConfig
	if len(dirs) > 0 {
		cfgPath := filepath.Join(filepath.Dir(dirs[0]), "configure.fz")
		if info, err := os.Stat(cfgPath); err == nil && !info.IsDir() {
			if autoCfg, err := config.Load(cfgPath); err == nil && autoCfg != nil {
				globalAutoBuild = &autoCfg.AutoBuild
				if verbose {
					os.Stdout.WriteString("Loaded configure.fz for auto-build settings\n")
				}
			}
		}
	}

	var depsArchives []string
	if cfg != nil && cfg.AutoBuildDeps {
		depsDir := filepath.Join(filepath.Dir(dirs[0]), "deps")
		if info, err := os.Stat(depsDir); err == nil && info.IsDir() {
			depsObjDir := joinPath(objDir, "deps")
			if err := utils.SecureMkdirAll(depsObjDir); err != nil {
				return nil, errors.New("cannot create deps obj dir: " + err.Error())
			}

			buildOrder := make(map[string]int)
			if globalAutoBuild != nil && len(globalAutoBuild.BuildOrder) > 0 {
				for i, name := range globalAutoBuild.BuildOrder {
					buildOrder[name] = i
				}
			}

			entries, err := os.ReadDir(depsDir)
			if err == nil {
				type depEntry struct {
					name  string
					path  string
					order int
				}
				var deps []depEntry
				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}
					depName := entry.Name()
					depPath := filepath.Join(depsDir, depName)
					order := 1000
					if o, ok := buildOrder[depName]; ok {
						order = o
					}
					deps = append(deps, depEntry{name: depName, path: depPath, order: order})
				}

				for i := 0; i < len(deps); i++ {
					for j := i + 1; j < len(deps); j++ {
						if deps[j].order < deps[i].order {
							deps[i], deps[j] = deps[j], deps[i]
						}
					}
				}

				for _, dep := range deps {
					depPath := dep.path
					depName := dep.name
					outArchive := filepath.Join(depsObjDir, depName+".a")

					fzCfgPath := filepath.Join(depPath, "fz.toml")
					var localCfg *config.Config
					if info, err := os.Stat(fzCfgPath); err == nil && !info.IsDir() {
						var lerr error
						localCfg, lerr = config.Load(fzCfgPath)
						if lerr != nil {
							if verbose {
								os.Stdout.WriteString("Warning: failed to load " + fzCfgPath + ": " + lerr.Error() + "\n")
							}
							continue
						}
						if verbose {
							enabledStr := "true"
							if !localCfg.DepBuild.Enabled {
								enabledStr = "false"
							}
							os.Stdout.WriteString("DEBUG: Loaded fz.toml for dep " + depName + ", DepBuild.Enabled=" + enabledStr + "\n")
						}
					}

					if localCfg != nil {
						builder := NewDepBuilder(ctx, depPath, depName, localCfg, globalAutoBuild, verbose)

						if !localCfg.DepBuild.Enabled {
							builder.logf("warn", "Dependency disabled in fz.toml")
							continue
						}

						excludePatterns := []string{"test", "tests", "test/*", "tests/*", "*/test/*", "*/tests/*"}
						if !localCfg.DepBuild.SkipTests {
							excludePatterns = nil
						}

						depIncludes := make([]string, 0)
						if len(localCfg.Include) > 0 {
							depIncludes = append(depIncludes, localCfg.Include...)
						}
						if len(localCfg.DepBuild.Include) > 0 {
							depIncludes = append(depIncludes, localCfg.DepBuild.Include...)
						}
						for i, inc := range depIncludes {
							if inc == "" {
								continue
							}
							if !filepath.IsAbs(inc) {
								abs := filepath.Join(depPath, inc)
								if a, err := filepath.Abs(abs); err == nil {
									depIncludes[i] = a
								} else {
									depIncludes[i] = abs
								}
							} else {
								if a, err := filepath.Abs(inc); err == nil {
									depIncludes[i] = a
								}
							}
						}

						var depSourceFiles []string
						if len(localCfg.SourceFiles) > 0 {
							depSourceFiles = make([]string, len(localCfg.SourceFiles))
							for i, sf := range localCfg.SourceFiles {
								depSourceFiles[i] = filepath.Join(depPath, sf)
							}
						}

						oldCcFlags := assembler.CcFlags
						oldLdFlags := linker.LdFlags
						envMap := make(map[string]string)
						if globalAutoBuild != nil && len(globalAutoBuild.DefaultEnvironment) > 0 {
							for k, v := range globalAutoBuild.DefaultEnvironment {
								envMap[k] = v
							}
						}
						if localCfg != nil && len(localCfg.DepBuild.Environment) > 0 {
							for k, v := range localCfg.DepBuild.Environment {
								envMap[k] = v
							}
						}
						if v, ok := envMap["CFLAGS"]; ok {
							assembler.CcFlags = v
							if strings.TrimSpace(v) == "" {
								assembler.CcFLagsParsed = nil
							} else {
								assembler.CcFLagsParsed = strings.Fields(v)
							}
						}
						if v, ok := envMap["LDFLAGS"]; ok {
							linker.LdFlags = v
							if strings.TrimSpace(v) != "" {
								collectedDepLdFlags = append(collectedDepLdFlags, v)
							}
						}
						if verbose {
							os.Stdout.WriteString("DEBUG: depIncludes for " + depName + ":\n")
							for _, d := range depIncludes {
								os.Stdout.WriteString("  " + d + "\n")
							}
						}
						_, err := BuildDir(ctx, []string{depPath}, outArchive, debug, verbose, mode, false, noCache, noSymbolCheck, sanitize, strict, excludePatterns, depSourceFiles, nil, depIncludes, nil, jobs, "static")
						assembler.CcFlags = oldCcFlags
						if strings.TrimSpace(oldCcFlags) == "" {
							assembler.CcFLagsParsed = nil
						} else {
							assembler.CcFLagsParsed = strings.Fields(oldCcFlags)
						}
						linker.LdFlags = oldLdFlags
						if err != nil {
							builder.logf("error", "Build failed")
							if globalAutoBuild == nil || !globalAutoBuild.ContinueOnError {
								return nil, errors.New("failed to build dependency " + depPath + ": " + err.Error())
							}
							continue
						}

						if len(localCfg.DepBuild.Outputs) > 0 {
							for _, o := range localCfg.DepBuild.Outputs {
								outPath := o
								if !filepath.IsAbs(outPath) {
									outPath = filepath.Join(depPath, outPath)
								}
								if info, err := os.Stat(outPath); err == nil && !info.IsDir() {
									depsArchives = append(depsArchives, outPath)
								}
							}
							continue
						}
						depsArchives = append(depsArchives, outArchive)
						continue
					}

					if verbose {
						os.Stdout.WriteString("Skipping " + depName + ": no fz.toml found (create " + fzCfgPath + " to enable)\n")
					}
				}
			}
		}
	}
	assembler.SetAdditionalIncludeDirs(includeDirs)
	defer assembler.SetAdditionalIncludeDirs(nil)
	if err := runPreprocessStep(cfg, dirs, generatedIncludeDir, verbose); err != nil {
		return nil, err
	}

	if len(srcFiles) == 0 {
		return nil, errors.New("no supported files found")
	}
	sort.Strings(srcFiles)

	cacheDir := joinPath(filepath.Dir(outBin), ".fz_cache")

	effectiveCache := determineCacheMode(cfg, noCache)
	var hashCache map[string][32]byte

	if cacheDir != "" {

		assembler.SetPCHCacheDir(filepath.Join(cacheDir, "pch"))

	}

	if effectiveCache != cacheOff {
		var err error
		hashCache, err = loadHashCache(cacheDir)
		if err != nil {
			if verbose {
				os.Stdout.WriteString("Warning: failed to load hash cache: " + err.Error() + "\n")
			}
			hashCache = nil
		}
	}
	if effectiveCache == cacheDisk {
		_ = PreloadCache(ctx, cacheDir)
	}

	if err := refreshSourceHashes(dirs); err != nil {
		return nil, errors.New("failed to refresh source hashes: " + err.Error())
	}

	if err := utils.SecureMkdirAll(joinPath(objDir, ".keep")); err != nil {
		return nil, errors.New("cannot create object temp dir: " + err.Error())
	}
	if effectiveCache == cacheDisk {
		if err := utils.SecureMkdirAll(joinPath(cacheDir, ".keep")); err != nil {
			return nil, errors.New("cannot create cache dir: " + err.Error())
		}
	} else {
		cacheDir = ""
	}
	cleanupObjDir := !keepObj

	pairs := make([]pair, len(srcFiles))
	for i, src := range srcFiles {
		srcAbs, err := filepath.Abs(src)
		if err != nil {
			return nil, err
		}
		if err := utils.EnsureInsideRoot(dirs[0], srcAbs); err != nil {
			return nil, err
		}
		var rel string
		rel, err = filepath.Rel(rootDir, srcAbs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			rel = filepath.Base(srcAbs)
		}
		var rep pathBuffer
		sep := byte(os.PathSeparator)
		for j := 0; j < len(rel); j++ {
			c := rel[j]
			if c == sep {
				rep.appendByte('_')
			} else {
				rep.appendByte(c)
			}
		}
		lastDot := -1
		for j := rep.n - 1; j >= 0; j-- {
			if rep.buf[j] == '.' {
				lastDot = j
				break
			}
		}
		var pb pathBuffer
		pb.appendString(objDir)
		pb.appendByte(byte(os.PathSeparator))
		if lastDot >= 0 {
			pb.appendBytes(rep.buf[:lastDot])
			pb.appendByte('_')
			pb.appendBytes(rep.buf[lastDot+1 : rep.n])
		} else {
			pb.appendBytes(rep.buf[:rep.n])
			pb.appendByte('_')
		}
		pb.appendString(".o")
		objPath := pb.String()
		if err := utils.SecureMkdirAll(objPath); err != nil {
			return nil, errors.New("cannot create subdir for object: " + err.Error())
		}
		pairs[i] = pair{src: src, obj: objPath}
	}

	sort.Slice(pairs, func(i, j int) bool { return pairs[i].obj < pairs[j].obj })

	depGraph, err := buildDependencyGraph(pairs, rootDir)
	if err != nil && verbose {
		os.Stdout.WriteString("Warning: could not build dependency graph: " + err.Error() + "; falling back to flat build\n")
	}
	useDAG := (err == nil && depGraph != nil && len(depGraph) == len(pairs))

	if jobs <= 0 {
		jobs = 1
	}

	buildOne := func(p pair) error {
		needAssemble := true
		if effectiveCache != cacheOff && hashCache != nil {
			if oldHash, ok := hashCache[p.src]; ok && oldHash == sourceHashes[p.src] {
				if effectiveCache == cacheRAM {
					restored, err := restoreRAMCache(p.src, p.obj, debug, mode)
					if err != nil {
						return errors.New("ram cache " + p.src + ": " + err.Error())
					}
					if restored {
						needAssemble = false
						if verbose {
							os.Stdout.WriteString("RAM cache hit for " + p.src + "\n")
						}
						var mbuf [512]byte
						n := copy(mbuf[:], "cache:hit:")
						n += copy(mbuf[n:], p.src)
						seal.UpdateGlobalState(mbuf[:n])
					}
				} else {
					restored, err := restoreShadowCache(p.src, p.obj, debug, mode)
					if err != nil {
						return errors.New("shadow cache " + p.src + ": " + err.Error())
					}
					if restored {
						needAssemble = false
						var mbuf [512]byte
						n := copy(mbuf[:], "shadow:restore:")
						n += copy(mbuf[n:], p.src)
						seal.UpdateGlobalState(mbuf[:n])
					} else {
						cachedObj, err := checkCache(p.src, cacheDir, debug, verbose, mode)
						if err == nil && cachedObj != "" {
							if verbose {
								os.Stdout.WriteString("Cache hit for " + p.src + "\n")
							}
							if err := utils.CopyFile(cachedObj, p.obj); err == nil {
								cachedSyms := strings.TrimSuffix(cachedObj, ".o") + ".syms"
								_ = utils.CopyFile(cachedSyms, p.obj+".syms")
								needAssemble = false
								var mbuf [512]byte
								n := copy(mbuf[:], "cache:hit:")
								n += copy(mbuf[n:], p.src)
								seal.UpdateGlobalState(mbuf[:n])
							}
						}
					}
				}
			}
		}
		if needAssemble {
			if verbose {
				os.Stdout.WriteString("Assembling " + p.src + " -> " + p.obj + "\n")
			}
			var mbuf [512]byte
			n := copy(mbuf[:], "assemble:")
			n += copy(mbuf[n:], p.src)
			seal.UpdateGlobalState(mbuf[:n])
			if err := assembler.Assemble(ctx, p.src, p.obj, debug, verbose, mode); err != nil {
				return errors.New("assemble " + p.src + ": " + err.Error())
			}
			if effectiveCache != cacheOff {
				if effectiveCache == cacheRAM {
					if err := storeRAMCache(p.src, p.obj, debug, mode); err != nil {
						return errors.New("ram cache " + p.src + ": " + err.Error())
					}
				} else {
					if err := AsyncStoreCache(p.src, p.obj, cacheDir, debug, verbose, mode); err != nil {
						return errors.New("cache " + p.src + ": " + err.Error())
					}
					if err := AsyncStoreShadowCache(p.src, p.obj, debug, mode); err != nil {
						return errors.New("shadow cache " + p.src + ": " + err.Error())
					}
				}
				var mbuf2 [512]byte
				m := copy(mbuf2[:], "cache:store:")
				m += copy(mbuf2[m:], p.src)
				seal.UpdateGlobalState(mbuf2[:m])
			}
		}
		return nil
	}

	if useDAG {
		dag := scheduler.NewDAGScheduler(jobs, len(pairs))
		for i := range pairs {
			idx := i
			p := pairs[i]
			_, err := dag.Submit(scheduler.AcquireTask(func(arg uintptr, extra uintptr) error {
				return buildOne(p)
			}, 0, 0), depGraph[idx])
			if err != nil {
				if cleanupObjDir {
					os.RemoveAll(objDir)
				}
				return nil, errors.New("failed to submit task: " + err.Error())
			}
		}
		if err := dag.Run(ctx); err != nil {
			if cleanupObjDir {
				os.RemoveAll(objDir)
			}
			return nil, err
		}
	} else {
		sched := scheduler.NewScheduler(jobs, len(pairs)*2)
		for i := range pairs {
			p := pairs[i]
			sched.SubmitBlocking(scheduler.AcquireTask(func(arg uintptr, extra uintptr) error {
				return buildOne(p)
			}, 0, 0), 0)
		}
		if err := sched.Run(ctx); err != nil {
			if cleanupObjDir {
				os.RemoveAll(objDir)
			}
			return nil, err
		}
	}

	objFiles := make([]string, len(pairs))
	for i, p := range pairs {
		objFiles[i] = p.obj
	}

	if len(depsArchives) > 0 {
		if verbose {
			os.Stdout.WriteString("Appending " + strconv.Itoa(len(depsArchives)) + " dependency archives: " + strings.Join(depsArchives, ", ") + "\n")
		}
		depsObjDir := filepath.Join(objDir, "deps")
		_ = os.MkdirAll(depsObjDir, 0o755)
		prepared := make([]string, 0, len(depsArchives))
		for _, a := range depsArchives {
			info, err := os.Lstat(a)
			if err != nil {
				prepared = append(prepared, a)
				continue
			}
			if info.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(a)
				if err != nil {
					prepared = append(prepared, a)
					continue
				}
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(a), target)
				}
				src := target
				dst := filepath.Join(depsObjDir, filepath.Base(a))
				if _, err := os.Stat(dst); err == nil {
					for i := 1; ; i++ {
						try := dst + "." + strconv.Itoa(i)
						if _, err := os.Stat(try); os.IsNotExist(err) {
							dst = try
							break
						}
					}
				}
				if in, err := os.Open(src); err == nil {
					out, err := os.Create(dst)
					if err == nil {
						_, _ = io.Copy(out, in)
						_ = out.Close()
					}
					_ = in.Close()
					prepared = append(prepared, dst)
					continue
				}
				prepared = append(prepared, a)
				continue
			}
			prepared = append(prepared, a)
		}

		objFiles = append(objFiles, prepared...)
	}

	oldGlobalLdFlags := linker.LdFlags
	if len(collectedDepLdFlags) > 0 {
		linker.LdFlags = strings.TrimSpace(oldGlobalLdFlags + " " + strings.Join(collectedDepLdFlags, " "))
	}

	if effectiveCache != cacheOff {
		if err := saveHashCache(cacheDir, sourceHashes); err != nil {
			if verbose {
				os.Stdout.WriteString("Warning: failed to save hash cache: " + err.Error() + "\n")
			}
		}
	}

	if buildType == "obj" {
	} else if buildType == "static" {
		if verbose {
			os.Stdout.WriteString("Creating static library " + outBin + "\n")
		}
		if err := createArchive(ctx, objFiles, outBin, verbose); err != nil {
			if cleanupObjDir {
				os.RemoveAll(objDir)
			}
			return nil, errors.New("Archive creation failed: " + err.Error())
		}
	} else {
		if verbose {
			os.Stdout.WriteString("Linking object files -> " + outBin + "\n")
		}
		if err := linker.LinkMultiple(ctx, objFiles, outBin, verbose, mode, noSymbolCheck, sanitize, strict, libs); err != nil {
			if cleanupObjDir {
				os.RemoveAll(objDir)
			}
			return nil, errors.New("link failed: " + err.Error())
		}
	}

	return &BuildResult{
		ObjectFiles: objFiles,
		Binary:      outBin,
		ObjDir:      objDir,
		CacheDir:    cacheDir,
	}, nil
}

func runPreprocessStep(cfg *config.Config, dirs []string, outputRoot string, verbose bool) error {
	if cfg == nil {
		return nil
	}
	if !cfg.Preprocess.Enabled {
		if len(cfg.Preprocess.Inputs) == 0 && len(cfg.Preprocess.Outputs) == 0 {
			for _, dir := range dirs {
				matches, err := filepath.Glob(filepath.Join(dir, "*.h.in"))
				if err != nil {
					return err
				}
				for _, templatePath := range matches {
					base := strings.TrimSuffix(filepath.Base(templatePath), ".in")
					outputPath := filepath.Join(outputRoot, base)
					if verbose {
						os.Stdout.WriteString("Generating header " + outputPath + " from " + templatePath + "\n")
					}
					if err := config.GenerateConfigH(templatePath, outputPath, cfg); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}

	for _, dir := range dirs {
		for i, input := range cfg.Preprocess.Inputs {
			if input == "" {
				continue
			}
			inputPath := input
			if !filepath.IsAbs(inputPath) {
				inputPath = filepath.Join(dir, inputPath)
			}
			var outputPath string
			if len(cfg.Preprocess.Outputs) > i {
				outputPath = cfg.Preprocess.Outputs[i]
			} else if len(cfg.Preprocess.Outputs) == 1 {
				outputPath = cfg.Preprocess.Outputs[0]
			} else {
				outputPath = strings.TrimSuffix(filepath.Base(inputPath), ".in")
			}
			if !filepath.IsAbs(outputPath) {
				outputPath = filepath.Join(outputRoot, filepath.Base(outputPath))
			}
			if verbose {
				os.Stdout.WriteString("Generating preprocessed output " + outputPath + " from " + inputPath + "\n")
			}
			proc := fzp.NewProcessor(fzp.Options{RootDir: filepath.Dir(inputPath), Macros: cfg.Preprocess.Defines})
			processed, err := proc.Process(inputPath, fzp.Options{RootDir: filepath.Dir(inputPath)})
			if err != nil {
				return err
			}
			if processed == "" {
				data, err := os.ReadFile(inputPath)
				if err != nil {
					return err
				}
				processed = string(data)
			}
			if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(outputPath, []byte(processed), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func createArchive(ctx context.Context, objFiles []string, outBin string, verbose bool) error {
	args := make([]string, 0, 2+len(objFiles))
	args = append(args, "rcs", outBin)
	args = append(args, objFiles...)
	if verbose {
		os.Stdout.WriteString("Running: ar " + strings.Join(args, " ") + "\n")
	}
	_, err := utils.RunCommand(ctx, verbose, os.Stdout, os.Stderr, "ar", args...)
	return err
}

func removeIfExists(path string, isDir bool, verbose bool) error {
	if _, err := os.Stat(path); err == nil {
		if verbose {
			os.Stdout.WriteString("Removing " + path + "\n")
		}
		if isDir {
			if err := os.RemoveAll(path); err != nil {
				return errors.New("failed to remove " + path + ": " + err.Error())
			}
		} else {
			if err := os.Remove(path); err != nil {
				return errors.New("failed to remove " + path + ": " + err.Error())
			}
		}
	}
	return nil
}

func CleanDir(dir string, verbose bool) error {
	objDir := joinPath(dir, ".fz_objs")
	if err := removeIfExists(objDir, true, verbose); err != nil {
		return err
	}
	cacheDir := joinPath(dir, ".fz_cache")
	if err := removeIfExists(cacheDir, true, verbose); err != nil {
		return err
	}

	base := filepath.Base(dir)
	outPath := joinPath(dir, base+".out")
	if err := removeIfExists(outPath, false, verbose); err != nil {
		return err
	}
	exePath := joinPath(dir, base+".exe")
	if err := removeIfExists(exePath, false, verbose); err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return errors.New("cannot read directory " + dir + ": " + err.Error())
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := joinPath(dir, name)

		if strings.HasSuffix(name, ".o") {
			if verbose {
				os.Stdout.WriteString("Removing object file " + path + "\n")
			}
			if err := os.Remove(path); err != nil {
				return errors.New("failed to remove " + path + ": " + err.Error())
			}
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Mode()&0o111 != 0 {
			ext := strings.ToLower(filepath.Ext(name))
			if !utils.SupportedExtension(ext) && ext != "" {
				if verbose {
					os.Stdout.WriteString("Removing executable " + path + "\n")
				}
				if err := os.Remove(path); err != nil {
					return errors.New("failed to remove " + path + ": " + err.Error())
				}
			} else if ext == "" {
				if verbose {
					os.Stdout.WriteString("Removing executable (no extension) " + path + "\n")
				}
				if err := os.Remove(path); err != nil {
					return errors.New("failed to remove " + path + ": " + err.Error())
				}
			}
		}
	}
	return nil
}

func CollectSourceFiles(cfg *config.Config, dirs []string) ([]string, error) {
	var srcFiles []string
	if cfg != nil && len(cfg.SourceFiles) > 0 {
		return cfg.SourceFiles, nil
	}
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".git" || name == ".svn" || name == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if utils.SupportedExtension(ext) {
				srcFiles = append(srcFiles, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return srcFiles, nil
}
