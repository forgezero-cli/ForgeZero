package utils

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/hex"
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

	"github.com/zeebo/blake3"
)

const (
	DirPerm  os.FileMode = 0o700
	FilePerm os.FileMode = 0o600
)

var (
	bufferPool     = sync.Pool{New: func() any { return new(bytes.Buffer) }}
	copyBufferPool = sync.Pool{New: func() any { return make([]byte, 32*1024) }}
)

var (
	executionRoot atomic.Value
	ToolChecksums sync.Map
	CheckToolFunc func(name string) error = checkToolInternal
)

func SetExecutionRoot(v string) {
	executionRoot.Store(v)
}

func GetExecutionRoot() string {
	v := executionRoot.Load()
	if v == nil {
		return ""
	}
	return v.(string)
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

func checkToolInternal(name string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("required tool not found in PATH: %s", name)
	}
	if v, ok := ToolChecksums.Load(name); ok {
		if expected, ok2 := v.(string); ok2 && expected != "" {
			actual, err := HashFile(path)
			if err != nil {
				return fmt.Errorf("cannot verify checksum for %s: %w", path, err)
			}
			if !constantTimeEqual(actual, expected) {
				return fmt.Errorf("tool checksum mismatch for %s", name)
			}
		}
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

func ResolveSecurePath(path string) (string, error) {
	if err := ValidateCLIPath(path); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path %s: %w", path, err)
	}
	abs = filepath.Clean(abs)
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return abs, nil
		}
		return "", fmt.Errorf("eval symlinks %s: %w", abs, err)
	}
	return eval, nil
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
	if targetEval == rootEval || strings.HasPrefix(targetEval, rootEval+string(os.PathSeparator)) {
		return nil
	}
	return fmt.Errorf("path %s outside project root %s", path, root)
}

func ValidateCLIArg(value string) error {
	if value == "" {
		return nil
	}
	if strings.ContainsAny(value, "`$&|;><*?[]{}()\"'\\") {
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
	if strings.ContainsAny(value, "`$&|;><*?[]{}()\"'\\\x00\n\r") {
		return fmt.Errorf("invalid path: %s", value)
	}
	if strings.Contains(value, ".."+string(os.PathSeparator)) || strings.Contains(value, string(os.PathSeparator)+"..") {
		return fmt.Errorf("path traversal not permitted: %s", value)
	}
	return nil
}

func ValidateFlagTokens(flagData []byte) ([]string, error) {
	if len(flagData) == 0 {
		return nil, nil
	}
	s := string(flagData)
	tokens := strings.Fields(s)
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
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(resolved)
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}
	if err != nil {
		return fmt.Errorf("cannot stat file %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}
	return nil
}

func SecureMkdirAll(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	resolved, err := ResolveSecurePath(dir)
	if err != nil {
		abs, absErr := filepath.Abs(dir)
		if absErr != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
		resolved = filepath.Clean(abs)
	}
	return os.MkdirAll(resolved, DirPerm)
}

func EnsureDir(path string) error {
	return SecureMkdirAll(path)
}

func SecureWriteFile(path string, data []byte) error {
	if err := SecureMkdirAll(path); err != nil {
		return err
	}
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		abs, absErr := filepath.Abs(path)
		if absErr != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		resolved = filepath.Clean(abs)
	}
	tmp, err := os.CreateTemp(filepath.Dir(resolved), ".fz_write_*.tmp")
	if err != nil {
		return fmt.Errorf("create temp for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(FilePerm); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temp %s: %w", tmpName, err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, resolved); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpName, resolved, err)
	}
	return os.Chmod(resolved, FilePerm)
}

func SupportedExtension(ext string) bool {
	ext = strings.ToLower(ext)
	switch ext {
	case ".asm", ".s", ".S", ".fasm", ".c", ".cpp", ".cc", ".cxx":
		return true
	}
	return false
}

func IsWindows() bool {
	return runtime.GOOS == "windows"
}

func buildCommand(ctx context.Context, name string, args ...string) (*exec.Cmd, string, error) {
	if name == "" {
		return nil, "", errors.New("command name required")
	}
	if err := ValidateCLIArg(name); err != nil {
		return nil, "", fmt.Errorf("invalid command name: %w", err)
	}
	resolved, err := exec.LookPath(name)
	if err != nil {
		return nil, "", fmt.Errorf("executable not found: %s", name)
	}
	base := filepath.Base(resolved)
	if base == "sh" || base == "bash" {
		if len(args) >= 1 {
			if err := ValidateCLIArg(args[0]); err != nil {
				return nil, "", fmt.Errorf("invalid arg: %w", err)
			}
		}
	}
	for i, a := range args {
		if (base == "sh" || base == "bash") && i == 1 && args[0] == "-c" {
			continue
		}
		if err := ValidateCLIArg(a); err != nil {
			return nil, "", fmt.Errorf("invalid arg: %w", err)
		}
	}
	cmd := exec.CommandContext(ctx, resolved, args...)
	if dir := GetExecutionRoot(); dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = deterministicEnv()
	return cmd, base, nil
}

func RunCommand(ctx context.Context, verbose bool, stdout, stderr io.Writer, name string, args ...string) (string, error) {
	cmd, _, err := buildCommand(ctx, name, args...)
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
	dstResolved, err := ResolveSecurePath(dst)
	if err != nil {
		abs, absErr := filepath.Abs(dst)
		if absErr != nil {
			return fmt.Errorf("copy dst %s: %w", dst, err)
		}
		dstResolved = filepath.Clean(abs)
	}
	in, err := os.Open(srcResolved)
	if err != nil {
		return fmt.Errorf("open src %s: %w", src, err)
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dstResolved), "fz_copy_*.tmp")
	if err != nil {
		return fmt.Errorf("create temp for %s: %w", dst, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(FilePerm); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temp %s: %w", tmpName, err)
	}
	buf := copyBufferPool.Get().([]byte)
	defer func() {
		ZeroizeBytes(buf)
		copyBufferPool.Put(buf)
	}()
	if _, err := io.CopyBuffer(tmp, in, buf); err != nil {
		tmp.Close()
		return fmt.Errorf("copy data to %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, dstResolved); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpName, dstResolved, err)
	}
	return os.Chmod(dstResolved, FilePerm)
}

func HashFile(path string) (string, error) {
	resolved, err := ResolveSecurePath(path)
	if err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	f, err := os.Open(resolved)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	hasher := blake3.New()
	buf := copyBufferPool.Get().([]byte)
	defer func() {
		ZeroizeBytes(buf)
		copyBufferPool.Put(buf)
	}()
	if _, err := io.CopyBuffer(hasher, f, buf); err != nil {
		return "", fmt.Errorf("hash read %s: %w", path, err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func HashDir(root string) (string, error) {
	root = filepath.Clean(root)
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("hash dir %s: %w", root, err)
	}
	return HashDirWithRoot(rootAbs, rootAbs)
}

func HashDirWithRoot(rootAbs, dir string) (string, error) {
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("hash dir abs %s: %w", dir, err)
	}
	dirAbs = filepath.Clean(dirAbs)
	rootEval, err := ResolveSecurePath(rootAbs)
	if err != nil {
		rootEval = filepath.Clean(rootAbs)
	}
	var files []string
	err = filepath.Walk(dirAbs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, readErr := os.Readlink(path)
			if readErr != nil {
				return fmt.Errorf("cannot read symlink %s: %w", path, readErr)
			}
			var targetAbs string
			if filepath.IsAbs(linkTarget) {
				targetAbs = filepath.Clean(linkTarget)
			} else {
				targetAbs = filepath.Clean(filepath.Join(filepath.Dir(path), linkTarget))
			}
			targetEval, evalErr := filepath.EvalSymlinks(targetAbs)
			if evalErr != nil {
				return fmt.Errorf("cannot resolve symlink %s target %s: %w", path, targetAbs, evalErr)
			}
			rootAbsClean := filepath.Clean(rootEval)
			if targetEval != rootAbsClean && !strings.HasPrefix(targetEval, rootAbsClean+string(os.PathSeparator)) {
				fmt.Fprintf(os.Stderr, "SECURITY WARNING: skipping symlink %s -> %s outside project root %s\n", path, targetAbs, rootAbsClean)
				return nil
			}
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dirAbs, path)
		if err != nil {
			return err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("invalid path outside root: %s", path)
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	hasher := blake3.New()
	sep := []byte{0}
	for _, rel := range files {
		if _, err := hasher.Write([]byte(rel)); err != nil {
			return "", err
		}
		if _, err := hasher.Write(sep); err != nil {
			return "", err
		}
		fullPath := filepath.Join(dirAbs, rel)
		f, err := os.Open(fullPath)
		if err != nil {
			return "", fmt.Errorf("open %s: %w", fullPath, err)
		}
		buf := copyBufferPool.Get().([]byte)
		if _, err := io.CopyBuffer(hasher, f, buf); err != nil {
			f.Close()
			ZeroizeBytes(buf)
			copyBufferPool.Put(buf)
			return "", fmt.Errorf("hash %s: %w", fullPath, err)
		}
		ZeroizeBytes(buf)
		copyBufferPool.Put(buf)
		if err := f.Close(); err != nil {
			return "", err
		}
		if _, err := hasher.Write(sep); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
