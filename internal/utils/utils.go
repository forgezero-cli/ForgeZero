package utils

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/zeebo/blake3"
)

var bufferPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}
var copyBufferPool = sync.Pool{New: func() any { return make([]byte, 32*1024) }}

var ExecutionRoot string
var ToolChecksums = map[string]string{}
var CheckToolFunc func(name string) error = checkToolInternal

func checkToolInternal(name string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("required tool not found in PATH: %s", name)
	}
	if expected, ok := ToolChecksums[name]; ok && expected != "" {
		actual, err := HashFile(path)
		if err != nil {
			return fmt.Errorf("cannot verify checksum for %s: %w", path, err)
		}
		if actual != expected {
			return fmt.Errorf("tool checksum mismatch for %s: expected %s got %s", name, expected, actual)
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

func EnsureInsideRoot(root, path string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	rootAbs = filepath.Clean(rootAbs)
	rootEval, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		rootEval = rootAbs
	}
	targetAbs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	targetAbs = filepath.Clean(targetAbs)
	targetEval, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		targetEval = targetAbs
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
	info, err := os.Stat(path)
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

func EnsureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
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

func RunCommandSilent(ctx context.Context, verbose bool, name string, args ...string) (output string, err error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if ExecutionRoot != "" {
		cmd.Dir = ExecutionRoot
	}
	cmd.Env = deterministicEnv()
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)
	if verbose {
		cmd.Stdout = io.MultiWriter(os.Stdout, buf)
		cmd.Stderr = io.MultiWriter(os.Stderr, buf)
	} else {
		cmd.Stdout = buf
		cmd.Stderr = buf
	}
	err = cmd.Run()
	return buf.String(), err
}

func CopyFile(src, dst string) error {
	if err := EnsureDir(dst); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), "fz_copy_*.tmp")
	if err != nil {
		return err
	}
	buf := copyBufferPool.Get().([]byte)
	defer func() {
		ZeroizeBytes(buf)
		copyBufferPool.Put(buf)
	}()
	if _, err := io.CopyBuffer(tmp, in, buf); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), dst)
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hasher := blake3.New()
	buf := copyBufferPool.Get().([]byte)
	defer func() {
		ZeroizeBytes(buf)
		copyBufferPool.Put(buf)
	}()
	if _, err := io.CopyBuffer(hasher, f, buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func HashDir(root string) (string, error) {
	root = filepath.Clean(root)
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return HashDirWithRoot(rootAbs, rootAbs)
}

func HashDirWithRoot(rootAbs, dir string) (string, error) {
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	dirAbs = filepath.Clean(dirAbs)

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

			rootAbsClean := filepath.Clean(rootAbs)
			// Allow symlink only if its resolved target is within the project root.
			if targetEval != rootAbsClean && !strings.HasPrefix(targetEval, rootAbsClean+string(os.PathSeparator)) {
				// Security warning, skip (never fail build).
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
			return "", err
		}
		buf := copyBufferPool.Get().([]byte)
		if _, err := io.CopyBuffer(hasher, f, buf); err != nil {
			f.Close()
			ZeroizeBytes(buf)
			copyBufferPool.Put(buf)
			return "", err
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

