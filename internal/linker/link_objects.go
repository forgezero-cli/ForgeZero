package linker

import (
	"context"
	"errors"
	"fmt"
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

func dedupObjects(elements []string) []string {
	encountered := make(map[string]struct{}, len(elements))
	result := make([]string, 0, len(elements))
	for _, v := range elements {
		if _, ok := encountered[v]; !ok {
			encountered[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
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
	return strings.Contains(output, "undefined reference") || strings.Contains(output, "undefined symbol") || strings.Contains(output, "unresolved symbol")
}

func newLinkError(cmd string, verbose bool, err error, output string) error {
	if verbose {
		return fmt.Errorf("%s failed: %w\n%s", cmd, err, output)
	}
	return fmt.Errorf("%s link failed (use -verbose for details)", cmd)
}
