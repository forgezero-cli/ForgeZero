package utils

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"flag"
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
	"syscall"
	"time"
	"unsafe"

	"fz/internal/config"
	fzvfs "fz/internal/fs"
	"fz/internal/seal"

	"github.com/zeebo/blake3"
)

var (
	hashKey        = [32]byte{0x9d, 0x74, 0x31, 0x6f, 0xd5, 0x23, 0x1b, 0xe4, 0xa1, 0x8f, 0x03, 0x71, 0x42, 0x5d, 0x6b, 0x9a, 0x3c, 0xf4, 0x75, 0x28, 0x0d, 0x62, 0x8a, 0x19, 0xbf, 0x4e, 0x50, 0x33, 0x13, 0x21, 0x97, 0x6c}
	bufferPool     = sync.Pool{New: func() any { return new(bytes.Buffer) }}
	copyBufferPool = sync.Pool{New: func() any { b := make([]byte, 32*1024); return &b }}
	hasherPool     = sync.Pool{New: func() any { h, _ := blake3.NewKeyed(hashKey[:]); return h }}
	hashSep        = []byte{0}
)

var limitedMode atomic.Bool

func IsLimitedMode() bool { return limitedMode.Load() }

func SelfAttest() error {
	if os.Getenv("FZ_STAGING") == "1" {
		return nil
	}
	if os.Getenv("FZ_SELF_ATTEST_DISABLE") == "1" {
		return nil
	}
	if strings.HasSuffix(filepath.Base(os.Args[0]), ".test") || flag.Lookup("test.v") != nil {
		return nil
	}
	ok, err := seal.Verify()
	if err != nil {
		limitedMode.Store(true)
		return nil
	}
	if !ok {
		limitedMode.Store(true)
		return nil
	}
	return nil
}

func BuildMerkleRoot(root string) ([32]byte, error) {
	var out [32]byte
	if root == "" {
		return out, fmt.Errorf("invalid merkle root")
	}
	files, err := collectRootFiles(root)
	if err != nil {
		return out, err
	}
	var reg [256][32]byte
	count := 0
	for i := range files {
		if count >= len(reg) {
			return out, fmt.Errorf("merkle registry overflow")
		}
		h, err := HashFileDigest(files[i])
		if err != nil {
			return out, err
		}
		reg[count] = h
		count++
	}
	if count == 0 {
		return hashEmptyDigest()
	}
	for count > 1 {
		next := 0
		for i := 0; i < count; i += 2 {
			left := reg[i]
			right := left
			if i+1 < count {
				right = reg[i+1]
			}
			reg[next] = hashDataPair(left, right)
			next++
		}
		count = next
	}
	return reg[0], nil
}

func hashDataPair(left, right [32]byte) [32]byte {
	var out [32]byte
	h, err := HashDataDigest(append(left[:], right[:]...))
	if err != nil {
		return out
	}
	out = h
	return out
}

func collectRootFiles(root string) ([]string, error) {
	rootAbs, err := resolveOrAbs(root)
	if err != nil {
		return nil, err
	}
	var files []string
	walkErr := filepath.Walk(rootAbs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("invalid path outside root: %s", path)
		}
		files = append(files, path)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.Strings(files)
	return files, nil
}

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
	return string(out[:])
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

func fnv1aHexUint64(h uint64) string {
	var out [16]byte
	const hextable = "0123456789abcdef"
	for i := 0; i < 8; i++ {
		b := byte(h >> ((7 - i) * 8))
		out[i*2] = hextable[b>>4]
		out[i*2+1] = hextable[b&0x0f]
	}
	return string(out[:])
}

func fnv1aHex(data []byte) string {
	return fnv1aHexUint64(fnv1aHash(data))
}

func ShadowCacheRoot() string {
	root := ""
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		root = filepath.Join(dir, "aegis", "shadow")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		root = filepath.Join(home, ".cache", "aegis", "shadow")
	}
	if mid, err := seal.MachineID(); err == nil && mid != "" {
		root = filepath.Join(root, fnv1aHex([]byte(mid)))
	}
	return root
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
	if err != nil || len(files) == 0 {
		files = []string{resolved}
	}
	sort.Strings(flags)
	sort.Strings(files)
	h := fnv1aHash(nil)
	for _, f := range flags {
		h = fnv1aHashAppend(h, []byte(f))
		h = fnv1aHashAppend(h, []byte{0})
	}
	if mid, err := seal.MachineID(); err == nil && mid != "" {
		h = fnv1aHashAppend(h, []byte(mid))
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
	return fnv1aHexUint64(h), nil
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
	resolved, err := FindExecutable(ctx, name)
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
	if cfg := ConfigFromContext(ctx); cfg != nil && cfg.Isolation != config.IsolationNone {
		cmd.Env = SafeEnv(cfg)
	} else {
		cmd.Env = deterministicEnv()
	}
	return cmd, nil
}

type ctxCfgKey struct{}

func ContextWithConfig(ctx context.Context, cfg *config.Config) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxCfgKey{}, cfg)
}

func ConfigFromContext(ctx context.Context) *config.Config {
	if ctx == nil {
		return nil
	}
	v := ctx.Value(ctxCfgKey{})
	if v == nil {
		return nil
	}
	if cfg, ok := v.(*config.Config); ok {
		return cfg
	}
	return nil
}

func SafeEnv(cfg *config.Config) []string {
	env := deterministicEnv()
	if cfg == nil {
		return env
	}
	if len(cfg.ToolchainSettings.EnvAllow) == 0 {
		return env
	}
	base := map[string]string{}
	for _, e := range os.Environ() {
		for _, k := range cfg.ToolchainSettings.EnvAllow {
			if strings.HasPrefix(e, k+"=") {
				parts := strings.SplitN(e, "=", 2)
				base[parts[0]] = parts[1]
			}
		}
	}
	out := append([]string(nil), env...)
	for k, v := range base {
		out = ensureEnv(out, k, v)
	}
	if len(cfg.ToolchainSettings.SearchPriority) > 0 {
		for _, p := range cfg.ToolchainSettings.SearchPriority {
			if p == "local" {
				root := GetExecutionRoot()
				if root != "" {
					localBin := filepath.Join(root, "toolchain", "bin")
					out = ensureEnv(out, "PATH", localBin+string(os.PathListSeparator)+os.Getenv("PATH"))
				}
				break
			}
		}
	}
	return out
}

func FindExecutable(ctx context.Context, name string) (string, error) {
	if filepath.IsAbs(name) {
		return filepath.Clean(name), nil
	}
	if cfg := ConfigFromContext(ctx); cfg != nil {
		for _, p := range cfg.ToolchainSettings.SearchPriority {
			switch p {
			case "local":
				root := GetExecutionRoot()
				if root != "" {
					cand := filepath.Join(root, "toolchain", "bin", name)
					if _, err := fileSystem().Stat(cand); err == nil {
						return filepath.Abs(cand)
					}
					cand2 := filepath.Join(root, "bin", name)
					if _, err := fileSystem().Stat(cand2); err == nil {
						return filepath.Abs(cand2)
					}
				}
			case "system":
				if pth, err := lookExecutable(name); err == nil {
					return filepath.Abs(pth)
				}
			}
		}
	}
	pth, err := lookExecutable(name)
	if err != nil {
		return "", err
	}
	return filepath.Abs(pth)
}

func ScrubHostPaths(path string, hostRoot string) (string, error) {
	if hostRoot == "" {
		return "", nil
	}
	data, err := fileSystem().ReadFile(path)
	if err != nil {
		return "", err
	}
	root := []byte(hostRoot)
	base := []byte("./" + filepath.Base(hostRoot))
	changed := false
	for i := 0; i+len(root) <= len(data); i++ {
		if bytes.Equal(data[i:i+len(root)], root) {
			copy(data[i:i+len(base)], base)
			for j := i + len(base); j < i+len(root); j++ {
				data[j] = 0
			}
			changed = true
			i += len(root) - 1
		}
	}
	if !changed {
		h := hex.EncodeToString(nil)
		return h, nil
	}
	if err := fileSystem().WriteFile(path, data, 0o755); err != nil {
		return "", err
	}
	return fmt.Sprintf("scrubbed:%d", len(root)), nil
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
	bufp := copyBufferPool.Get().(*[]byte)
	buf := *bufp
	if _, err := io.CopyBuffer(tmp, in, buf); err != nil {
		ZeroizeBytes(buf)
		copyBufferPool.Put(bufp)
		in.Close()
		tmp.Close()
		_ = fileSystem().Remove(tmpName)
		return fmt.Errorf("copy data to %s: %w", tmpName, err)
	}
	ZeroizeBytes(buf)
	copyBufferPool.Put(bufp)
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

func hashEmptyDigest() ([32]byte, error) {
	var out [32]byte
	hasher := hasherPool.Get().(*blake3.Hasher)
	digest := hasher.Digest()
	if _, err := digest.Read(out[:]); err != nil {
		hasher.Reset()
		hasherPool.Put(hasher)
		return out, err
	}
	hasher.Reset()
	hasherPool.Put(hasher)
	return out, nil
}

func fileIsStable(of interface {
	Stat() (os.FileInfo, error)
}, delay time.Duration,
) bool {
	fi1, err := of.Stat()
	if err != nil {
		return false
	}
	if delay > 0 {
		time.Sleep(delay)
	}
	fi2, err := of.Stat()
	if err != nil {
		return false
	}
	return fi1.Size() == fi2.Size() && fi1.ModTime() == fi2.ModTime()
}

func HashDataDigest(data []byte) ([32]byte, error) {
	var out [32]byte
	hasher := hasherPool.Get().(*blake3.Hasher)
	if _, err := hasher.Write(data); err != nil {
		hasher.Reset()
		hasherPool.Put(hasher)
		return out, err
	}
	digest := hasher.Digest()
	if _, err := digest.Read(out[:]); err != nil {
		hasher.Reset()
		hasherPool.Put(hasher)
		return out, err
	}
	hasher.Reset()
	hasherPool.Put(hasher)
	return out, nil
}

func openRawPath(path string) (int, error) {
	const atFDCWD = ^uintptr(0) - 99
	var buf [4096]byte
	if len(path) >= len(buf) {
		return -1, syscall.ENAMETOOLONG
	}
	n := copy(buf[:], path)
	buf[n] = 0
	r0, _, errno := syscall.Syscall(syscall.SYS_OPENAT, atFDCWD, uintptr(unsafe.Pointer(&buf[0])), uintptr(syscall.O_RDONLY|syscall.O_CLOEXEC))
	if errno != 0 {
		return -1, errno
	}
	return int(r0), nil
}

func hashRawFileDigest(path string) ([32]byte, error) {
	var out [32]byte
	if fileSystem() != fzvfs.Default {
		f, err := openVerified(path)
		if err != nil {
			return out, ErrHashOpen
		}
		hasher := hasherPool.Get().(*blake3.Hasher)
		var buf [65536]byte
		if _, err := io.CopyBuffer(hasher, f, buf[:]); err != nil {
			hasher.Reset()
			hasherPool.Put(hasher)
			f.Close()
			return out, err
		}
		if cerr := f.Close(); cerr != nil {
			hasher.Reset()
			hasherPool.Put(hasher)
			return out, cerr
		}
		digest := hasher.Digest()
		if _, err := digest.Read(out[:]); err != nil {
			hasher.Reset()
			hasherPool.Put(hasher)
			return out, err
		}
		hasher.Reset()
		hasherPool.Put(hasher)
		return out, nil
	}

	fd, err := openRawPath(path)
	if err != nil {
		return out, ErrHashOpen
	}
	hasher := hasherPool.Get().(*blake3.Hasher)
	var buf [65536]byte
	for {
		n, readErr := syscall.Read(fd, buf[:])
		if n > 0 {
			if _, err := hasher.Write(buf[:n]); err != nil {
				syscall.Close(fd)
				hasher.Reset()
				hasherPool.Put(hasher)
				return out, err
			}
		}
		if readErr != nil {
			syscall.Close(fd)
			hasher.Reset()
			hasherPool.Put(hasher)
			return out, ErrHashRead
		}
		if n == 0 {
			break
		}
	}
	syscall.Close(fd)
	digest := hasher.Digest()
	if _, err := digest.Read(out[:]); err != nil {
		hasher.Reset()
		hasherPool.Put(hasher)
		return out, err
	}
	hasher.Reset()
	hasherPool.Put(hasher)
	return out, nil
}

func HashFileDigest(path string) ([32]byte, error) {
	var out [32]byte
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		return out, ErrHashOpen
	}
	return hashRawFileDigest(resolved)
}

func HashFile(path string) (string, error) {
	out, err := HashFileDigest(path)
	if err != nil {
		return "", err
	}
	return blake3HexDigestToString(out), nil
}

func HashDir(root string) (string, error) {
	rootAbs, err := resolveOrAbs(root)
	if err != nil {
		return "", fmt.Errorf("hash dir %s: %w", root, err)
	}
	return HashDirWithRoot(rootAbs, rootAbs)
}

func HashDirWithRoot(rootAbs, dir string) (string, error) {
	digest, err := HashDirDigest(rootAbs, dir)
	if err != nil {
		return "", err
	}
	return blake3HexDigestToString(digest), nil
}

func HashDirDigest(rootAbs, dir string) ([32]byte, error) {
	var out [32]byte
	dirAbs, err := resolveOrAbs(dir)
	if err != nil {
		return out, ErrHashRead
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
		return out, ErrHashRead
	}
	sort.Strings(files)
	hasher := hasherPool.Get().(*blake3.Hasher)
	buf := alignedSlice(32 * 1024)
	var f io.ReadCloser
	for _, rel := range files {
		n := copy(buf, rel)
		if _, err := hasher.Write(buf[:n]); err != nil {
			if f != nil {
				f.Close()
			}
			hasher.Reset()
			hasherPool.Put(hasher)
			return out, ErrHashRead
		}
		if _, err := hasher.Write(hashSep); err != nil {
			if f != nil {
				f.Close()
			}
			hasher.Reset()
			hasherPool.Put(hasher)
			return out, ErrHashRead
		}
		fullPath := dirAbs + string(os.PathSeparator) + rel
		f, err = openVerified(fullPath)
		if err != nil {
			hasher.Reset()
			hasherPool.Put(hasher)
			return out, ErrHashOpen
		}
		mmapped := false
		if of, ok := f.(interface {
			Stat() (os.FileInfo, error)
			Fd() uintptr
		}); ok {
			fi, _ := of.Stat()
			if fi != nil && fi.Size() > 0 {
				size := fi.Size()
				fd := getFileDescriptor(of)
				if err := lockFileShared(fd); err == nil {
					if !fileIsStable(of, 1*time.Millisecond) {
						_ = unlockFile(fd)
					} else {
						data, merr := mmapFile(fd, size)
						if merr == nil {
							if fi2, err := of.Stat(); err == nil && fi2.Size() == size {
								madviseNormal(data)
								if _, err := hasher.Write(data); err == nil {
									_ = unmapFile(data)
									mmapped = true
								} else {
									_ = unmapFile(data)
								}
							} else {
								_ = unmapFile(data)
							}
						}
						_ = unlockFile(fd)
					}
				}
			}
		}
		if !mmapped {
			if _, err := io.CopyBuffer(hasher, f, buf); err != nil {
				f.Close()
				hasher.Reset()
				hasherPool.Put(hasher)
				return out, ErrHashRead
			}
		}
		f.Close()
		if _, err := hasher.Write(hashSep); err != nil {
			hasher.Reset()
			hasherPool.Put(hasher)
			return out, ErrHashRead
		}
	}
	digest := hasher.Digest()
	if _, err := digest.Read(out[:]); err != nil {
		hasher.Reset()
		hasherPool.Put(hasher)
		return out, err
	}
	hasher.Reset()
	hasherPool.Put(hasher)
	return out, nil
}

func hashPath(path string) uint64 {
	h := uint64(1469598103934665603)
	for i := 0; i < len(path); i++ {
		h ^= uint64(path[i])
		h *= 1099511628211
	}
	return h
}

func resolveIncludePathBytes(currentDir string, include []byte) (string, error) {
	var tmp [4096]byte
	n := copy(tmp[:], currentDir)
	if n > 0 && tmp[n-1] != os.PathSeparator {
		tmp[n] = os.PathSeparator
		n++
	}
	m := copy(tmp[n:], include)
	resolved := filepath.Clean(string(tmp[:n+m]))
	return ResolveSecurePath(resolved)
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
	defer func() { _ = unmapFile(data) }()
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
			resolved, err := resolveIncludePathBytes(currentDir, data[start:k])
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
			resolved, err := resolveIncludePathBytes(currentDir, data[start:k])
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
