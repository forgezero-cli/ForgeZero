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
	ErrHashOpen  = errors.New("hash: open")
	ErrHashMmap  = errors.New("hash: mmap")
	ErrHashSize  = errors.New("hash: size")
	ErrHashRead  = errors.New("hash: read")
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
	return b[off:off+n]
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
	defer w.mu.Unlock()
	return w.buf.Write(p)
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
	defer bufferPool.Put(buf)
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
	if stdout == mbw {
		return buf.String(), runErr
	}
	return "", runErr
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
	defer in.Close()
	tmp, err := fileSystem().CreateTemp(filepath.Dir(dstResolved), "fz_copy_*.tmp")
	if err != nil {
		return fmt.Errorf("create temp for %s: %w", dst, err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		tmp.Close()
		if cleanup {
			_ = fileSystem().Remove(tmpName)
		}
	}()
	if err := fileSystem().Chmod(tmpName, FilePerm); err != nil {
		return fmt.Errorf("chmod temp %s: %w", tmpName, err)
	}
	buf := copyBufferPool.Get().([]byte)
	defer func() {
		ZeroizeBytes(buf)
		copyBufferPool.Put(buf)
	}()
	if _, err := io.CopyBuffer(tmp, in, buf); err != nil {
		return fmt.Errorf("copy data to %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp %s: %w", tmpName, err)
	}
	if err := renameResolved(tmpName, dstResolved); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpName, dstResolved, err)
	}
	cleanup = false
	return fileSystem().Chmod(dstResolved, FilePerm)
}

func hashStream(r io.Reader) (string, error) {
	hasher := hasherPool.Get().(*blake3.Hasher)
	buf := copyBufferPool.Get().([]byte)
	defer func() {
		hasher.Reset()
		hasherPool.Put(hasher)
		ZeroizeBytes(buf)
		copyBufferPool.Put(buf)
	}()
	if _, err := io.CopyBuffer(hasher, r, buf); err != nil {
		return "", err
	}
	digest := hasher.Digest()
	var out [32]byte
	digest.Read(out[:])
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
	defer f.Close()
	
	if of, ok := f.(interface{ Stat() (os.FileInfo, error); Fd() uintptr }); ok {
		fi, err := of.Stat()
		if err == nil && fi.Size() > 0 {
			size := fi.Size()
			data, merr := mmapFile(getFileDescriptor(of), size)
			if merr == nil {
				sum := blake3.Sum256(data)
				_ = unmapFile(data)
				return blake3HexDigestToString(sum), nil
			}
		}
	}
	
	h, err := hashStream(f)
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
	defer func() {
		hasher.Reset()
		hasherPool.Put(hasher)
	}()
	
	buf := alignedSlice(32 * 1024)
	for _, rel := range files {
		hasher.Write([]byte(rel))
		hasher.Write(hashSep)
		fullPath := filepath.Join(dirAbs, rel)
		
		f, err := openVerified(fullPath)
		if err != nil {
			return "", ErrHashOpen
		}
		
		mmapped := false
		if of, ok := f.(interface{ Stat() (os.FileInfo, error); Fd() uintptr }); ok {
			fi, _ := of.Stat()
			if fi != nil && fi.Size() > 0 {
				size := fi.Size()
				data, merr := mmapFile(getFileDescriptor(of), size)
				if merr == nil {
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
	return blake3HexDigestToString(out), nil
}
