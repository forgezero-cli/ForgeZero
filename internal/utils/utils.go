package utils

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/zeebo/blake3"
)

var (
	bufferPool     = sync.Pool{New: func() any { return new(bytes.Buffer) }}
	copyBufferPool = sync.Pool{New: func() any { return make([]byte, 32*1024) }}
	hasherPool     = sync.Pool{New: func() any { return blake3.New() }}
	hashSep        = []byte{0}
)

var (
	ErrHashOpen     = errors.New("hash: open")
	ErrHashMmap     = errors.New("hash: mmap")
	ErrHashSize     = errors.New("hash: size")
	ErrHashRead     = errors.New("hash: read")
	ErrScanOpen     = errors.New("scan: open")
	ErrScanMmap     = errors.New("scan: mmap")
	ErrScanResolve  = errors.New("scan: resolve")
	includeBytes    = [7]byte{'i', 'n', 'c', 'l', 'u', 'd', 'e'}
	warnOutsideHead = []byte("WARNING: include outside root ignored: ")
)

var globalScratchPad = func() []byte {
	b := make([]byte, 1024*1024+64)
	base := uintptr(unsafe.Pointer(&b[0]))
	off := int((64 - (base % 64)) % 64)
	return b[off:]
}()

func alignedSlice(n int) []byte {
	if n <= len(globalScratchPad) {
		return globalScratchPad[:n]
	}
	b := make([]byte, n+64)
	base := uintptr(unsafe.Pointer(&b[0]))
	off := int((64 - (base % 64)) % 64)
	return b[off : off+n]
}

var (
	executionRoot atomic.Value
	ToolChecksums sync.Map
	CheckToolFunc func(name string) error = checkToolInternal
)

func SetExecutionRoot(v string) {
	if v != "" {
		v = filepath.Clean(v)
	}
	executionRoot.Store(v)
}

func GetExecutionRoot() string {
	v := executionRoot.Load()
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

type mutexBufferWriter struct {
	mu  sync.Mutex
	buf *bytes.Buffer
}

func (w *mutexBufferWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	n, err := w.buf.Write(p)
	w.mu.Unlock()
	return n, err
}

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func blake3HexDigestToString(d [32]byte) string {
	var out [64]byte
	const hextable = "0123456789abcdef"
	for i := 0; i < 32; i++ {
		b := d[i]
		out[i*2] = hextable[b>>4]
		out[i*2+1] = hextable[b&0x0f]
	}
	return *(*string)(unsafe.Pointer(&struct {
		data *[64]byte
		len  int
	}{&out, 64}))
}

func fnv1aHash(data []byte) uint64 {
	const (
		offset uint64 = 1469598103934665603
		prime  uint64 = 1099511628211
	)
	h := offset
	for _, b := range data {
		h ^= uint64(b)
		h *= prime
	}
	return h
}

func fnv1aHex(data []byte) string {
	return fmt.Sprintf("%016x", fnv1aHash(data))
}

func ShadowCacheRoot() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "aegis", "shadow")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "aegis", "shadow")
}

func ShadowCachePath(key string) string {
	return filepath.Join(ShadowCacheRoot(), key+".o")
}

func ShadowCacheKey(src string, flags []string) (string, error) {
	resolved, err := ResolveSecurePath(src)
	if err != nil {
		return "", err
	}
	files, err := ScanDependencies(resolved)
	if err != nil {
		files = nil
	}
	sort.Strings(flags)
	sort.Strings(files)
	h := fnv1aHash(nil)
	for _, f := range flags {
		h = fnv1aHashAppend(h, []byte(f))
		h = fnv1aHashAppend(h, []byte{0})
	}
	for _, file := range files {
		hv, err := HashFile(file)
		if err != nil {
			return "", err
		}
		h = fnv1aHashAppend(h, []byte(hv))
		h = fnv1aHashAppend(h, []byte{0})
	}
	return fmt.Sprintf("%016x", h), nil
}

func fnv1aHashAppend(h uint64, data []byte) uint64 {
	const prime uint64 = 1099511628211
	for _, b := range data {
		h ^= uint64(b)
		h *= prime
	}
	return h
}

func checkToolInternal(name string) error {
	path, err := lookExecutable(name)
	if err != nil {
		return fmt.Errorf("required tool not found in PATH: %s", name)
	}
	v, ok := ToolChecksums.Load(name)
	if !ok {
		return nil
	}
	expected, ok2 := v.(string)
	if !ok2 || expected == "" {
		return nil
	}
	actual, err := HashFile(path)
	if err != nil {
		return fmt.Errorf("cannot verify checksum for %s: %w", path, err)
	}
	if !constantTimeEqual(actual, expected) {
		return fmt.Errorf("tool checksum mismatch for %s", name)
	}
	return nil
}

func CheckTool(name string) error {
	if CheckToolFunc == nil {
		return checkToolInternal(name)
	}
	return CheckToolFunc(name)
}

func ensureEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func deterministicEnv() []string {
	env := os.Environ()
	env = ensureEnv(env, "LC_ALL", "C")
	env = ensureEnv(env, "LANG", "C")
	env = ensureEnv(env, "TZ", "UTC")
	env = ensureEnv(env, "SOURCE_DATE_EPOCH", "1600000000")
	return env
}

func EnsureInsideRoot(root, path string) error {
	rootEval, err := ResolveSecurePath(root)
	if err != nil {
		return err
	}
	targetEval, err := ResolveSecurePath(path)
	if err != nil {
		return err
	}
	if pathWithinRoot(rootEval, targetEval) {
		return nil
	}
	return fmt.Errorf("path %s outside project root %s", path, root)
}

func ValidateCLIArg(value string) error {
	if value == "" {
		return nil
	}
	if strings.ContainsAny(value, forbiddenArgChars()) {
		return fmt.Errorf("invalid CLI argument: %s", value)
	}
	if strings.ContainsAny(value, "\x00\n\r") {
		return fmt.Errorf("invalid CLI argument: %s", value)
	}
	return nil
}

func ValidateCLIPath(value string) error {
	if value == "" {
		return nil
	}
	if strings.ContainsAny(value, forbiddenPathChars()) {
		return fmt.Errorf("invalid path: %s", value)
	}
	sep := string(os.PathSeparator)
	if strings.Contains(value, ".."+sep) || strings.Contains(value, sep+"..") {
		return fmt.Errorf("path traversal not permitted: %s", value)
	}
	if runtime.GOOS == "windows" {
		if strings.Contains(value, "..\\") || strings.Contains(value, "\\..") {
			return fmt.Errorf("path traversal not permitted: %s", value)
		}
		if isUnsafeUNC(value) {
			return fmt.Errorf("invalid UNC path: %s", value)
		}
	}
	return nil
}

func ValidateFlagTokens(flagData []byte) ([]string, error) {
	if len(flagData) == 0 {
		return nil, nil
	}
	tokens := strings.Fields(string(flagData))
	for _, token := range tokens {
		if err := ValidateCLIArg(token); err != nil {
			return nil, err
		}
	}
	return tokens, nil
}

func ZeroizeBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

func DeriveNames(src, outFlag, outObjFlag string) (bin, obj string) {
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	objDefault := base + ".o"
	binDefault := base
	if runtime.GOOS == "windows" && filepath.Ext(binDefault) == "" {
		binDefault += ".exe"
	}
	if outObjFlag != "" {
		obj = outObjFlag
	} else {
		obj = objDefault
	}
	if outFlag != "" {
		bin = outFlag
	} else {
		bin = binDefault
	}
	return
}

func CheckFileExists(path string) error {
	info, err := LstatPath(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", path)
		}
		return fmt.Errorf("stat file %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink not permitted: %s", path)
	}
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		return err
	}
	f, err := openVerified(resolved)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

func SecureMkdirAll(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	resolved, err := ResolveSecurePath(dir)
	if err != nil {
		resolved, err = resolveOrAbs(dir)
		if err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return fileSystem().MkdirAll(resolved, DirPerm)
}

func EnsureDir(path string) error {
	return SecureMkdirAll(path)
}

func SecureWriteFile(path string, data []byte) error {
	if err := SecureMkdirAll(path); err != nil {
		return err
	}
	resolved, err := resolveDest(path)
	if err != nil {
		return err
	}
	return atomicWrite(resolved, data)
}

func SupportedExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".asm", ".s", ".fasm", ".c", ".cpp", ".cc", ".cxx":
		return true
	case ".S":
		return true
	}
	return false
}

func IsWindows() bool {
	return runtime.GOOS == "windows"
}

func buildCommand(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	if name == "" {
		return nil, errors.New("command name required")
	}
	if err := ValidateCLIArg(name); err != nil {
		return nil, fmt.Errorf("invalid command name: %w", err)
	}
	resolved, err := lookExecutable(name)
	if err != nil {
		return nil, fmt.Errorf("executable not found: %s", name)
	}
	base := filepath.Base(resolved)
	if base == "sh" || base == "bash" {
		if len(args) >= 1 {
			if err := ValidateCLIArg(args[0]); err != nil {
				return nil, fmt.Errorf("invalid arg: %w", err)
			}
		}
	}
	for i, a := range args {
		if (base == "sh" || base == "bash") && i == 1 && args[0] == "-c" {
			continue
		}
		if err := ValidateCLIArg(a); err != nil {
			return nil, fmt.Errorf("invalid arg: %w", err)
		}
	}
	cmd := exec.CommandContext(ctx, resolved, args...)
	if dir := GetExecutionRoot(); dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = deterministicEnv()
	return cmd, nil
}

func RunCommand(ctx context.Context, verbose bool, stdout, stderr io.Writer, name string, args ...string) (string, error) {
	cmd, err := buildCommand(ctx, name, args...)
	if err != nil {
		return "", err
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	mbw := &mutexBufferWriter{buf: buf}
	if stdout == nil {
		stdout = mbw
	}
	if stderr == nil {
		stderr = mbw
	}
	if verbose {
		cmd.Stdout = io.MultiWriter(os.Stdout, stdout)
		cmd.Stderr = io.MultiWriter(os.Stderr, stderr)
	} else {
		cmd.Stdout = stdout
		cmd.Stderr = stderr
	}
	runErr := cmd.Run()
	out := ""
	if stdout == mbw {
		out = buf.String()
	}
	bufferPool.Put(buf)
	return out, runErr
}

func RunCommandSilent(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
	return RunCommand(ctx, verbose, nil, nil, name, args...)
}

func RunCommandOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := RunCommandSilent(ctx, false, name, args...)
	return []byte(out), err
}

func CopyFile(src, dst string) error {
	srcResolved, err := ResolveSecurePath(src)
	if err != nil {
		return fmt.Errorf("copy src %s: %w", src, err)
	}
	if err := SecureMkdirAll(dst); err != nil {
		return err
	}
	dstResolved, err := resolveDest(dst)
	if err != nil {
		return fmt.Errorf("copy dst %s: %w", dst, err)
	}
	in, err := openVerified(srcResolved)
	if err != nil {
		return fmt.Errorf("open src %s: %w", src, err)
	}
	tmp, err := fileSystem().CreateTemp(filepath.Dir(dstResolved), "fz_copy_*.tmp")
	if err != nil {
		return fmt.Errorf("create temp for %s: %w", dst, err)
	}
	tmpName := tmp.Name()
	if err := fileSystem().Chmod(tmpName, FilePerm); err != nil {
		tmp.Close()
		_ = fileSystem().Remove(tmpName)
		return fmt.Errorf("chmod temp %s: %w", tmpName, err)
	}
	buf := copyBufferPool.Get().([]byte)
	if _, err := io.CopyBuffer(tmp, in, buf); err != nil {
		ZeroizeBytes(buf)
		copyBufferPool.Put(buf)
		in.Close()
		tmp.Close()
		_ = fileSystem().Remove(tmpName)
		return fmt.Errorf("copy data to %s: %w", tmpName, err)
	}
	ZeroizeBytes(buf)
	copyBufferPool.Put(buf)
	if err := in.Close(); err != nil {
		tmp.Close()
		_ = fileSystem().Remove(tmpName)
		return fmt.Errorf("close src %s: %w", srcResolved, err)
	}
	if err := tmp.Close(); err != nil {
		_ = fileSystem().Remove(tmpName)
		return fmt.Errorf("close temp %s: %w", tmpName, err)
	}
	if err := renameResolved(tmpName, dstResolved); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpName, dstResolved, err)
	}
	return fileSystem().Chmod(dstResolved, FilePerm)
}

func hashStream(r io.Reader) (string, error) {
	hasher := hasherPool.Get().(*blake3.Hasher)
	buf := copyBufferPool.Get().([]byte)
	if _, err := io.CopyBuffer(hasher, r, buf); err != nil {
		hasher.Reset()
		hasherPool.Put(hasher)
		ZeroizeBytes(buf)
		copyBufferPool.Put(buf)
		return "", err
	}
	digest := hasher.Digest()
	var out [32]byte
	digest.Read(out[:])
	hasher.Reset()
	hasherPool.Put(hasher)
	ZeroizeBytes(buf)
	copyBufferPool.Put(buf)
	return blake3HexDigestToString(out), nil
}

func HashFile(path string) (string, error) {
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		return "", ErrHashOpen
	}
	f, err := openVerified(resolved)
	if err != nil {
		return "", ErrHashOpen
	}
	var result string
	if of, ok := f.(interface {
		Stat() (os.FileInfo, error)
		Fd() uintptr
	}); ok {
		fi, err := of.Stat()
		if err == nil {
			if fi.Size() == 0 {
				result = blake3HexDigestToString(blake3.Sum256(nil))
			} else {
				size := fi.Size()
				data, merr := mmapFile(getFileDescriptor(of), size)
				if merr == nil {
					madviseNormal(data)
					result = blake3HexDigestToString(blake3.Sum256(data))
					_ = unmapFile(data)
				}
			}
		}
	}
	if result != "" {
		f.Close()
		return result, nil
	}
	h, err := hashStream(f)
	f.Close()
	if err != nil {
		return "", ErrHashRead
	}
	return h, nil
}

func HashDir(root string) (string, error) {
	rootAbs, err := resolveOrAbs(root)
	if err != nil {
		return "", fmt.Errorf("hash dir %s: %w", root, err)
	}
	return HashDirWithRoot(rootAbs, rootAbs)
}

func HashDirWithRoot(rootAbs, dir string) (string, error) {
	dirAbs, err := resolveOrAbs(dir)
	if err != nil {
		return "", ErrHashRead
	}
	rootEval, _ := ResolveSecurePath(rootAbs)
	if rootEval == "" {
		rootEval = filepath.Clean(rootAbs)
	}
	var files []string
	walkErr := filepath.Walk(dirAbs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			ok, serr := symlinkAllowed(rootEval, path, "")
			if serr != nil {
				return serr
			}
			if !ok {
				return nil
			}
			abs, aerr := resolveOrAbs(path)
			if aerr != nil {
				return aerr
			}
			target, aerr := fileSystem().EvalSymlinks(abs)
			if aerr != nil {
				return aerr
			}
			tinfo, aerr := fileSystem().Lstat(target)
			if aerr != nil {
				return aerr
			}
			if tinfo.IsDir() {
				return nil
			}
			path = target
			info = tinfo
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dirAbs, path)
		if err != nil {
			return err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return ErrHashRead
		}
		files = append(files, rel)
		return nil
	})
	if walkErr != nil {
		return "", ErrHashRead
	}
	sort.Strings(files)
	hasher := hasherPool.Get().(*blake3.Hasher)
	buf := alignedSlice(32 * 1024)
	for _, rel := range files {
		hasher.Write([]byte(rel))
		hasher.Write(hashSep)
		fullPath := dirAbs + string(os.PathSeparator) + rel
		f, err := openVerified(fullPath)
		if err != nil {
			hasher.Reset()
			hasherPool.Put(hasher)
			return "", ErrHashOpen
		}
		mmapped := false
		if of, ok := f.(interface {
			Stat() (os.FileInfo, error)
			Fd() uintptr
		}); ok {
			fi, _ := of.Stat()
			if fi != nil && fi.Size() > 0 {
				size := fi.Size()
				data, merr := mmapFile(getFileDescriptor(of), size)
				if merr == nil {
					madviseNormal(data)
					hasher.Write(data)
					_ = unmapFile(data)
					mmapped = true
				}
			}
		}
		if !mmapped {
			io.CopyBuffer(hasher, f, buf)
		}
		f.Close()
		hasher.Write(hashSep)
	}
	digest := hasher.Digest()
	var out [32]byte
	digest.Read(out[:])
	hasher.Reset()
	hasherPool.Put(hasher)
	return blake3HexDigestToString(out), nil
}

func hashPath(path string) uint64 {
	h := uint64(1469598103934665603)
	for i := 0; i < len(path); i++ {
		h ^= uint64(path[i])
		h *= 1099511628211
	}
	return h
}

func joinPath(base, file string) string {
	if len(base) == 0 {
		return file
	}
	sep := byte(os.PathSeparator)
	need := len(base) + 1 + len(file)
	buf := make([]byte, need)
	n := copy(buf, base)
	if base[len(base)-1] != sep {
		buf[n] = sep
		n++
	}
	n += copy(buf[n:], file)
	return unsafe.String(&buf[0], n)
}

func resolveIncludePath(currentDir, include string) (string, error) {
	if filepath.IsAbs(include) {
		return ResolveSecurePath(include)
	}
	return ResolveSecurePath(joinPath(currentDir, include))
}

func warnOutsideRoot(path string) {
	var tmp [4096]byte
	n := copy(tmp[:], warnOutsideHead)
	if n+len(path)+1 > len(tmp) {
		os.Stderr.Write(warnOutsideHead)
		os.Stderr.Write([]byte(path))
		os.Stderr.Write([]byte{'\n'})
		return
	}
	n += copy(tmp[n:], path)
	tmp[n] = '\n'
	os.Stderr.Write(tmp[:n+1])
}

func mmapPath(path string) ([]byte, error) {
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		return nil, ErrScanResolve
	}
	f, err := openVerified(resolved)
	if err != nil {
		return nil, ErrScanOpen
	}
	of, ok := f.(interface {
		Stat() (os.FileInfo, error)
		Fd() uintptr
	})
	if !ok {
		f.Close()
		return nil, ErrScanOpen
	}
	fi, err := of.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if fi.Size() == 0 {
		f.Close()
		return nil, nil
	}
	data, err := mmapFile(getFileDescriptor(of), fi.Size())
	f.Close()
	if err != nil {
		return nil, ErrScanMmap
	}
	return data, nil
}

func scanFileIncludes(path string) ([]string, error) {
	data, err := mmapPath(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	defer unmapFile(data)
	currentDir := filepath.Dir(path)
	list := make([]string, 0, 8)
	for i := 0; i < len(data); i++ {
		if data[i] != '#' {
			continue
		}
		j := i + 1
		for j < len(data) && (data[j] == ' ' || data[j] == '\t') {
			j++
		}
		if j+len(includeBytes) >= len(data) {
			continue
		}
		ok := true
		for k := 0; k < len(includeBytes); k++ {
			if data[j+k] != includeBytes[k] {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		j += len(includeBytes)
		for j < len(data) && (data[j] == ' ' || data[j] == '\t') {
			j++
		}
		if j >= len(data) {
			continue
		}
		switch data[j] {
		case '"':
			start := j + 1
			k := start
			for k < len(data) && data[k] != '"' {
				k++
			}
			if k >= len(data) {
				continue
			}
			inc := unsafe.String(&data[start], k-start)
			resolved, err := resolveIncludePath(currentDir, inc)
			if err == nil {
				list = append(list, resolved)
			}
		case '<':
			start := j + 1
			k := start
			for k < len(data) && data[k] != '>' {
				k++
			}
			if k >= len(data) {
				continue
			}
			inc := unsafe.String(&data[start], k-start)
			resolved, err := resolveIncludePath(currentDir, inc)
			if err == nil {
				list = append(list, resolved)
			}
		}
	}
	return list, nil
}

func ScanDependencies(path string) ([]string, error) {
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		return nil, err
	}
	rootDir := filepath.Dir(resolved)
	stack := []string{resolved}
	visited := make(map[uint64]struct{}, 64)
	deps := make([]string, 0, 64)
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		h := hashPath(cur)
		if _, ok := visited[h]; ok {
			continue
		}
		visited[h] = struct{}{}
		deps = append(deps, cur)
		includes, err := scanFileIncludes(cur)
		if err != nil {
			return nil, err
		}
		for _, inc := range includes {
			if !pathWithinRoot(rootDir, inc) {
				warnOutsideRoot(inc)
				continue
			}
			hi := hashPath(inc)
			if _, ok := visited[hi]; ok {
				continue
			}
			stack = append(stack, inc)
		}
	}
	return deps, nil
}
