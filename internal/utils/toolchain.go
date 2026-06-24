package utils

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/config"
)

var strictToolchain string

var buildToolNames = map[string]struct{}{
	"zig": {}, "fasm": {}, "nasm": {}, "as": {}, "gcc": {}, "clang": {}, "ld": {},
	"emcc": {}, "em++": {}, "g++": {}, "clang++": {},
	"arm-linux-gnueabihf-gcc": {}, "arm-linux-gnueabihf-g++": {},
	"arm-linux-gnueabihf-as": {}, "arm-linux-gnueabihf-ld": {},
	"riscv64-unknown-elf-gcc": {}, "riscv64-unknown-elf-g++": {},
	"riscv64-unknown-elf-as": {}, "riscv64-unknown-elf-ld": {},
	"wasm-ld": {}, "ld.lld": {}, "lld": {},
}

func SetToolchainPolicy(toolchain string) {
	strictToolchain = strings.TrimSpace(strings.ToLower(toolchain))
}

func toolEnvVar(name string) string {
	name = strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	return "FZ_TOOLCHAIN_" + name
}

func lookupPathOverride(cfg *config.Config, name string) string {
	if cfg != nil && cfg.ToolchainSettings.ToolPaths != nil {
		if p, ok := cfg.ToolchainSettings.ToolPaths[name]; ok && p != "" {
			return p
		}
	}
	if env := os.Getenv(toolEnvVar(name)); env != "" {
		return env
	}
	return ""
}

func isBuildTool(name string) bool {
	name = filepath.Base(strings.TrimSpace(strings.ToLower(name)))
	_, ok := buildToolNames[name]
	return ok
}

func securityPanic(reason string) {
	os.Stderr.WriteString("SECURITY PANIC: ")
	os.Stderr.WriteString(reason)
	os.Stderr.WriteString("\n")
	os.Exit(2)
}

func enforceStrictToolchain(name string) {
	if strictToolchain == "zig" && isBuildTool(name) && filepath.Base(name) != "zig" {
		securityPanic("Zig-only toolchain policy violated: " + name)
	}
}

func toolchainName(raw string) string {
	return strings.TrimSpace(strings.ToLower(raw))
}

func toolCandidates(tool, target string) []string {
	tool = toolchainName(tool)
	switch tool {
	case "auto":
		return append(toolCandidates("zig", target), append(toolCandidates("gcc", target), toolCandidates("clang", target)...)...)
	case "zig":
		return []string{"zig"}
	case "fasm":
		return []string{"fasm"}
	case "nasm":
		if strings.Contains(target, "wasm") {
			return []string{"clang", "nasm"}
		}
		if strings.Contains(target, "arm") {
			return []string{"arm-linux-gnueabihf-nasm", "nasm"}
		}
		if strings.Contains(target, "riscv") {
			return []string{"riscv64-unknown-elf-nasm", "nasm"}
		}
		return []string{"nasm"}
	case "gas":
		if strings.Contains(target, "wasm") {
			return []string{"clang", "as"}
		}
		if strings.Contains(target, "arm") {
			return []string{"arm-linux-gnueabihf-as", "as"}
		}
		if strings.Contains(target, "riscv") {
			return []string{"riscv64-unknown-elf-as", "as"}
		}
		return []string{"as"}
	case "gcc":
		if strings.Contains(target, "riscv") && flag.Lookup("toolchain") != nil && flag.Lookup("toolchain").Value.String() == "zig" {
			return []string{"zig", "gcc"}
		}
		if strings.Contains(target, "wasm") {
			return []string{"emcc", "gcc"}
		}
		if strings.Contains(target, "arm") {
			return []string{"arm-linux-gnueabihf-gcc", "gcc"}
		}
		if strings.Contains(target, "riscv") {
			return []string{"riscv64-unknown-elf-gcc", "gcc"}
		}
		return []string{"gcc"}
	case "clang":
		return []string{"clang"}
	case "ld":
		if strings.Contains(target, "wasm") {
			return []string{"wasm-ld", "ld.lld", "lld", "ld"}
		}
		if strings.Contains(target, "arm") {
			return []string{"arm-linux-gnueabihf-ld", "ld"}
		}
		if strings.Contains(target, "riscv") {
			return []string{"riscv64-unknown-elf-ld", "ld"}
		}
		return []string{"ld"}
	default:
		return []string{tool}
	}
}

func localToolCandidates(name string) []string {
	root := GetExecutionRoot()
	if root == "" {
		return nil
	}
	return []string{
		filepath.Join(root, "toolchain", "bin", name),
		filepath.Join(root, "bin", name),
	}
}

func findTool(ctx context.Context, cfg *config.Config, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("tool name required")
	}
	enforceStrictToolchain(name)
	if filepath.IsAbs(name) {
		return filepath.Clean(name), nil
	}
	if cfg != nil {
		if override := lookupPathOverride(cfg, name); override != "" {
			return filepath.Clean(override), nil
		}
		for _, p := range cfg.ToolchainSettings.SearchPriority {
			switch p {
			case "local":
				for _, cand := range localToolCandidates(name) {
					if _, err := fileSystem().Stat(cand); err == nil {
						return filepath.Abs(cand)
					}
				}
			case "system":
				if pth, err := lookExecutable(name); err == nil {
					return filepath.Abs(pth)
				}
			}
		}
	}
	if override := lookupPathOverride(nil, name); override != "" {
		return filepath.Clean(override), nil
	}
	if pth, err := lookExecutable(name); err == nil {
		return filepath.Abs(pth)
	}
	return "", fmt.Errorf("toolchain executable not found: %s", name)
}

func ResolveToolPath(ctx context.Context, cfg *config.Config, tool, target string) (string, error) {
	if tool == "" {
		return "", fmt.Errorf("tool name required")
	}
	if cfg != nil && cfg.Toolchain == "zig" {
		tool = "zig"
	}
	for _, cand := range toolCandidates(tool, target) {
		if pth, err := findTool(ctx, cfg, cand); err == nil {
			return pth, nil
		}
	}
	return "", fmt.Errorf("toolchain executable not found: %s", tool)
}
