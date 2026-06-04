package builder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"fz/internal/assembler"
	"fz/internal/config"
	"fz/internal/linker"
	"fz/internal/seal"
	"fz/internal/utils"
)

type BuildResult struct {
	ObjectFiles []string
	Binary      string
	ObjDir      string
	CacheDir    string
}

var targetArch string

type pathBuffer struct {
	buf [2048]byte
	n   int
}

func (p *pathBuffer) appendString(s string) {
	copy(p.buf[p.n:], s)
	p.n += len(s)
}

func (p *pathBuffer) appendByte(b byte) {
	p.buf[p.n] = b
	p.n++
}

func (p *pathBuffer) appendBytes(b []byte) {
	copy(p.buf[p.n:], b)
	p.n += len(b)
}

func (p *pathBuffer) String() string {
	return unsafe.String((*byte)(unsafe.Pointer(&p.buf[0])), p.n)
}

type pair struct {
	src string
	obj string
}

type resultError struct {
	err error
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

func joinPath(base, name string) string {
	var pb pathBuffer
	pb.appendString(base)
	if len(base) > 0 && base[len(base)-1] != byte(os.PathSeparator) {
		pb.appendByte(byte(os.PathSeparator))
	}
	pb.appendString(name)
	return pb.String()
}

func buildCacheKey(hash string, debug bool, mode string) string {
	var pb pathBuffer
	pb.appendString(hash)
	pb.appendByte('_')
	if debug {
		pb.appendByte('1')
	} else {
		pb.appendByte('0')
	}
	pb.appendByte('_')
	pb.appendString(mode)
	return pb.String()
}

func cacheEntryPath(dir, key string) string {
	var pb pathBuffer
	pb.appendString(dir)
	if len(dir) > 0 && dir[len(dir)-1] != byte(os.PathSeparator) {
		pb.appendByte(byte(os.PathSeparator))
	}
	pb.appendString(key)
	return pb.String()
}

func RunHooks(ctx context.Context, hooks []config.Hook) error {
	for _, h := range hooks {
		out, err := utils.RunCommand(ctx, false, nil, nil, "sh", "-c", h.Cmd)
		_ = out
		if err != nil {
			if h.Critical {
				return fmt.Errorf("hook failed (critical): %w", err)
			}
			return fmt.Errorf("hook failed: %w", err)
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
				_, _ = utils.RunCommand(context.Background(), false, nil, nil, "sh", "-c", cfg.Hooks.OnFailure)
			}
		}()
	}
	res, err = buildDirInner(ctx, dirs, outBin, debug, verbose, mode, keepObj, noCache, noSymbolCheck, sanitize, strict, exclude, sourceFiles, ignoreMatcher, includes, libs, jobs, buildType)
	return res, err
}

func buildDirInner(ctx context.Context, dirs []string, outBin string, debug, verbose bool, mode string, keepObj, noCache, noSymbolCheck, sanitize, strict bool, exclude, sourceFiles []string, ignoreMatcher interface{}, includes, libs []string, jobs int, buildType string) (*BuildResult, error) {
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
		return nil, fmt.Errorf("output path %s is a directory, cannot write binary", outBin)
	}
	if err := utils.EnsureDir(outBin); err != nil {
		return nil, fmt.Errorf("cannot create output directory: %w", err)
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
							fmt.Printf("Skipping directory tree: %s\n", path)
						}
						return filepath.SkipDir
					}
					return nil
				}
				if matchExclude(path, exclude) {
					if verbose {
						fmt.Printf("Excluding file: %s\n", path)
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
				return nil, fmt.Errorf("walk error in %s: %w", dir, err)
			}
		}
	}
	if len(srcFiles) == 0 {
		return nil, fmt.Errorf("no supported files found")
	}
	sort.Strings(srcFiles)

	objDir := joinPath(filepath.Dir(outBin), ".fz_objs")
	cacheDir := joinPath(filepath.Dir(outBin), ".fz_cache")
	if err := utils.SecureMkdirAll(joinPath(objDir, ".keep")); err != nil {
		return nil, fmt.Errorf("cannot create object temp dir: %w", err)
	}
	if !noCache {
		if err := utils.SecureMkdirAll(joinPath(cacheDir, ".keep")); err != nil {
			return nil, fmt.Errorf("cannot create cache dir: %w", err)
		}
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
			return nil, fmt.Errorf("cannot create subdir for object: %w", err)
		}
		pairs[i] = pair{src: src, obj: objPath}
	}

	sort.Slice(pairs, func(i, j int) bool { return pairs[i].obj < pairs[j].obj })

	if jobs <= 0 {
		jobs = 1
	}
	var nextIndex uint32
	var stopFlag uint32
	var firstErr atomic.Pointer[resultError]
	var wg sync.WaitGroup

	recordError := func(err error) {
		entry := &resultError{err: err}
		if firstErr.CompareAndSwap(nil, entry) {
			atomic.StoreUint32(&stopFlag, 1)
		}
	}

	worker := func() {
		defer wg.Done()
		for {
			if atomic.LoadUint32(&stopFlag) == 1 {
				return
			}
			idx := int(atomic.AddUint32(&nextIndex, 1) - 1)
			if idx >= len(pairs) {
				return
			}
			p := pairs[idx]
			needAssemble := true
			if !noCache {
				restored, err := restoreShadowCache(p.src, p.obj, debug, mode)
				if err != nil {
					recordError(fmt.Errorf("shadow cache %s: %w", p.src, err))
					return
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
							fmt.Printf("Cache hit for %s\n", p.src)
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
			if needAssemble {
				if verbose {
					fmt.Printf("Assembling %s -> %s\n", p.src, p.obj)
				}
				var mbuf [512]byte
				n := copy(mbuf[:], "assemble:")
				n += copy(mbuf[n:], p.src)
				seal.UpdateGlobalState(mbuf[:n])
				if err := assembler.Assemble(ctx, p.src, p.obj, debug, verbose, mode); err != nil {
					recordError(fmt.Errorf("assemble %s: %w", p.src, err))
					return
				}
				if !noCache {
					if err := storeCache(p.src, p.obj, cacheDir, debug, verbose, mode); err != nil {
						recordError(fmt.Errorf("cache %s: %w", p.src, err))
						return
					}
					if err := storeShadowCache(p.src, p.obj, debug, mode); err != nil {
						recordError(fmt.Errorf("shadow cache %s: %w", p.src, err))
						return
					}
					var mbuf2 [512]byte
					m := copy(mbuf2[:], "cache:store:")
					m += copy(mbuf2[m:], p.src)
					seal.UpdateGlobalState(mbuf2[:m])
				}
			}
		}
	}

	for w := 0; w < jobs; w++ {
		wg.Add(1)
		go worker()
	}
	wg.Wait()

	if entry := firstErr.Load(); entry != nil {
		if cleanupObjDir {
			os.RemoveAll(objDir)
		}
		return nil, entry.err
	}

	objFiles := make([]string, len(pairs))
	for i, p := range pairs {
		objFiles[i] = p.obj
	}

	if buildType == "obj" {
		cleanupObjDir = false
	} else if buildType == "static" {
		if verbose {
			_, _ = os.Stdout.WriteString("Creating static library " + outBin + "\n")
		}

		if err := createArchive(ctx, objFiles, outBin, verbose); err != nil {
			if cleanupObjDir {
				os.RemoveAll(objDir)
			}
			return nil, fmt.Errorf("Archive creation failed: %w", err)
		}
	} else {
		if verbose {
			_, _ = os.Stdout.WriteString("Linking object files -> " + outBin + "\n")
		}
		if err := linker.LinkMultiple(ctx, objFiles, outBin, verbose, mode, noSymbolCheck, sanitize, strict, libs); err != nil {
			if cleanupObjDir {
				os.RemoveAll(objDir)
			}
			return nil, fmt.Errorf("link failed: %w", err)
		}
	}

	return &BuildResult{
		ObjectFiles: objFiles,
		Binary:      outBin,
		ObjDir:      objDir,
		CacheDir:    cacheDir,
	}, nil
}

func checkCache(src, cacheDir string, debug, verbose bool, mode string) (string, error) {
	h, err := utils.HashFile(src)
	if err != nil {
		return "", err
	}
	key := buildCacheKey(h, debug, mode)
	cacheObj := cacheEntryPath(cacheDir, key+".o")
	info, err := os.Stat(cacheObj)
	if err != nil {
		return "", err
	}
	if info.Size() == 0 {
		return "", errors.New("cached file is empty")
	}
	return cacheObj, nil
}

func restoreShadowCache(src, obj string, debug bool, mode string) (bool, error) {
	flags := []string{"debug=" + strconv.FormatBool(debug), "mode=" + mode}
	key, err := utils.ShadowCacheKey(src, flags)
	if err != nil {
		return false, err
	}
	shadowObj := utils.ShadowCachePath(key)
	if _, err := os.Stat(shadowObj); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if err := utils.EnsureDir(obj); err != nil {
		return false, err
	}
	if err := utils.LinkOrClone(shadowObj, obj); err != nil {
		return false, err
	}
	if err := os.Chmod(obj, utils.FilePerm); err != nil {
		return false, err
	}
	if debug {
		fmt.Printf("Shadow cache restored %s -> %s\n", shadowObj, obj)
	}
	return true, nil
}

func storeCache(src, obj, cacheDir string, debug, verbose bool, mode string) error {
	h, err := utils.HashFile(src)
	if err != nil {
		return err
	}
	key := buildCacheKey(h, debug, mode)
	cacheObj := cacheEntryPath(cacheDir, key+".o")
	return utils.CopyFile(obj, cacheObj)
}

func storeShadowCache(src, obj string, debug bool, mode string) error {
	flags := []string{"debug=" + strconv.FormatBool(debug), "mode=" + mode}
	key, err := utils.ShadowCacheKey(src, flags)
	if err != nil {
		return err
	}
	shadowObj := utils.ShadowCachePath(key)
	if err := os.MkdirAll(filepath.Dir(shadowObj), 0o755); err != nil {
		return err
	}
	if err := utils.LinkOrClone(obj, shadowObj); err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func createArchive(ctx context.Context, objFiles []string, outBin string, verbose bool) error {
	args := append([]string{"rcs", outBin}, objFiles...)
	if verbose {
		fmt.Printf("Running: ar %s\n", strings.Join(args, " "))
	}
	_, err := utils.RunCommand(ctx, verbose, os.Stdout, os.Stderr, "ar", args...)
	return err
}

func removeIfExists(path string, isDir bool, verbose bool) error {
	if _, err := os.Stat(path); err == nil {
		if verbose {
			fmt.Printf("Removing %s\n", path)
		}
		if isDir {
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("failed to remove %s: %w", path, err)
			}
		} else {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove %s: %w", path, err)
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
		return fmt.Errorf("cannot read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := joinPath(dir, name)

		if strings.HasSuffix(name, ".o") {
			if verbose {
				fmt.Printf("Removing object file %s\n", path)
			}
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove %s: %w", path, err)
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
					fmt.Printf("Removing executable %s\n", path)
				}
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove %s: %w", path, err)
				}
			} else if ext == "" {
				if verbose {
					fmt.Printf("Removing executable (no extension) %s\n", path)
				}
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove %s: %w", path, err)
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
