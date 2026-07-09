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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/fzp"

	"github.com/forgezero-cli/ForgeZero/internal/assembler"
	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/drivers/scheduler"
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
	jobs = AdjustJobs(jobs)

	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	rootDir, err := filepath.Abs(dirs[0])
	if err != nil {
		return nil, err
	}
	rootDir = filepath.Clean(rootDir)
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

	if cfg != nil && len(cfg.BuildRules) > 0 {
		return runBuildRules(ctx, cfg, verbose, jobs)
	}

	var srcFiles []string
	if len(sourceFiles) > 0 {
		srcFiles = append(srcFiles, sourceFiles...)
	} else {
		for _, dir := range dirs {
			err := utils.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					name := info.Name()
					if name == ".git" || name == ".svn" || name == "node_modules" || matchExclude(path, exclude) {
						if verbose {
							os.Stdout.WriteString("Skipping directory tree: " + path + "\n")
						}
						return filepath.SkipDir
					}
					return nil
				}
				if matchExclude(path, exclude) {
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
	objDir := joinPath(filepath.Dir(outBin), ".fz_objs")
	generatedIncludeDir := joinPath(objDir, "include")
	if err := utils.SecureMkdirAll(generatedIncludeDir); err != nil {
		return nil, errors.New("cannot create generated include dir: " + err.Error())
	}
	includeDirs := []string{generatedIncludeDir}
	if cfg != nil {
		includeDirs = append(includeDirs, cfg.Include...)
	}
	includeDirs = append(includeDirs, includes...)
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

	depGraph, err := buildDependencyGraph(pairs)
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
				if len(matches) == 0 {
					continue
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
		for _, input := range cfg.Preprocess.Inputs {
			if input == "" {
				continue
			}
			inputPath := input
			if !filepath.IsAbs(inputPath) {
				inputPath = filepath.Join(dir, inputPath)
			}
			outputPath := inputPath
			if len(cfg.Preprocess.Outputs) > 0 {
				if len(cfg.Preprocess.Outputs) == 1 {
					outputPath = cfg.Preprocess.Outputs[0]
				} else if len(cfg.Preprocess.Outputs) > 0 {
					outputPath = cfg.Preprocess.Outputs[0]
				}
			}
			if !filepath.IsAbs(outputPath) {
				outputPath = filepath.Join(dir, outputPath)
			}
			if verbose {
				os.Stdout.WriteString("Generating preprocessed output " + outputPath + " from " + inputPath + "\n")
			}
			data, err := os.ReadFile(inputPath)
			if err != nil {
				return err
			}
			proc := fzp.NewProcessor(fzp.Options{RootDir: filepath.Dir(inputPath), Macros: cfg.Preprocess.Defines})
			processed, err := proc.Process(inputPath, fzp.Options{RootDir: filepath.Dir(inputPath)})
			if err != nil {
				return err
			}
			if processed == "" {
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
