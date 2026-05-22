package linker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"fz/internal/config"
	"fz/internal/utils"
	"fz/internal/zig"
)

func LinkObjects(ctx context.Context, target string, objs []string, cfg *config.Config) error {
	if err := validateLinkCall(ctx, target); err != nil {
		return err
	}
	if len(objs) == 0 {
		return errors.New("no object files to link")
	}
	unique := dedupObjects(objs)
	if len(unique) == 0 {
		return errors.New("no object files to link")
	}
	if err := utils.EnsureDir(target); err != nil {
		return err
	}

	cmd, args, err := buildLinkCommand(unique, target, cfg)
	if err != nil {
		return err
	}
	if err := utils.CheckTool(cmd); err != nil {
		return err
	}

	verbose := cfg != nil && cfg.Verbose
	if verbose {
		fmt.Printf("Running: %s %s\n", cmd, strings.Join(args, " "))
	}
	output, err := runLinkerCommand(ctx, verbose, cmd, args)
	if err != nil {
		if hasUndefinedSymbol(output) {
			return fmt.Errorf("link failed: undefined symbols\n%s", output)
		}
		return newLinkError(cmd, verbose, err, output)
	}
	return nil
}

func dedupObjects(objs []string) []string {
	seen := make(map[uint64][]string, len(objs))
	unique := make([]string, 0, len(objs))
	for _, obj := range objs {
		if obj == "" {
			continue
		}
		h := fnv1aString(obj)
		bucket := seen[h]
		match := false
		for _, existing := range bucket {
			if existing == obj {
				match = true
				break
			}
		}
		if match {
			continue
		}
		seen[h] = append(bucket, obj)
		unique = append(unique, obj)
	}
	sort.Strings(unique)
	return unique
}

func fnv1aString(s string) uint64 {
	const (
		offset uint64 = 1469598103934665603
		prime  uint64 = 1099511628211
	)
	h := offset
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	return h
}

func buildLinkCommand(objs []string, target string, cfg *config.Config) (string, []string, error) {
	mode := "auto"
	toolchain := "auto"
	if cfg != nil {
		mode = strings.TrimSpace(strings.ToLower(cfg.Mode))
		toolchain = strings.TrimSpace(strings.ToLower(cfg.Toolchain))
	}
	if mode == "" {
		mode = "auto"
	}
	if toolchain == "" {
		toolchain = "auto"
	}
	if mode == "raw" {
		cmd := ldForTarget()
		args := append([]string{}, objs...)
		args = append(args, "-o", target)
		if cfg != nil {
			args = append(args, cfg.Flags.Ld...)
			if cfg.OptimizationLevel > 2 {
				args = append(args, "--gc-sections")
			}
		}
		return cmd, args, nil
	}

	if toolchain == "zig" || (toolchain == "auto" && useZig()) {
		if !zig.IsAvailable() {
			if toolchain == "zig" {
				return "", nil, errors.New("zig toolchain requested but not available")
			}
		} else {
			return buildZigLinkCommand(objs, target, cfg)
		}
	}

	return gccForTarget(), buildGccLinkCommand(objs, target, cfg), nil
}

func buildZigLinkCommand(objs []string, target string, cfg *config.Config) (string, []string, error) {
	cmd := "zig"
	args := make([]string, 0, len(objs)+10)
	args = append(args, "c++", "-target", Target)
	if cfg != nil && cfg.OptimizationLevel > 2 {
		args = append(args, "-flto", "-fuse-linker-plugin", "-Wl,--gc-sections")
	}
	args = append(args, objs...)
	args = append(args, "-o", target)
	if cfg != nil {
		args = append(args, cfg.Flags.Ld...)
	}
	return cmd, args, nil
}

func buildGccLinkCommand(objs []string, target string, cfg *config.Config) []string {
	args := make([]string, 0, len(objs)+10)
	args = append(args, objs...)
	if cfg != nil && cfg.OptimizationLevel > 2 {
		args = append(args, "-flto", "-fuse-linker-plugin", "-Wl,--gc-sections")
	}
	args = append(args, "-o", target)
	if cfg != nil {
		args = append(args, cfg.Flags.Ld...)
	}
	return args
}

func hasUndefinedSymbol(output string) bool {
	if output == "" {
		return false
	}
	raw := []byte(output)
	return bytes.Contains(raw, []byte("undefined reference")) || bytes.Contains(raw, []byte("undefined symbol")) || bytes.Contains(raw, []byte("unresolved symbol"))
}

func newLinkError(cmd string, verbose bool, err error, output string) error {
	if verbose {
		return fmt.Errorf("%s failed: %w\n%s", cmd, err, output)
	}
	return fmt.Errorf("%s link failed (use -verbose for details)", cmd)
}

func createMmapResponseFile(args []string) (string, error) {
	f, err := os.CreateTemp("", "fz_link_args_*.rsp")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := f.Chmod(utils.FilePerm); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	total := 0
	for _, arg := range args {
		if strings.ContainsAny(arg, "\n\r\x00") {
			f.Close()
			os.Remove(name)
			return "", fmt.Errorf("invalid argument for response file")
		}
		if err := utils.ValidateCLIArg(arg); err != nil {
			f.Close()
			os.Remove(name)
			return "", fmt.Errorf("invalid argument for response file: %w", err)
		}
		total += len(arg) + 1
	}
	if err := f.Truncate(int64(total)); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	data, err := mmapWritableFile(f, total)
	if err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	off := 0
	for _, arg := range args {
		n := copy(data[off:], arg)
		off += n
		data[off] = '\n'
		off++
	}
	if err := unmapWritableFile(data); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}
