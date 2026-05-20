package linker

import (
	"fmt"
	"strings"

	"fz/internal/assembler"
)

func ApplyGccLdFlags(args []string, ldScript, textAddr string) []string {
	if ldScript != "" {
		args = append(args, "-Wl,-T,"+ldScript)
	}
	if textAddr != "" {
		args = append(args, "-Wl,-Ttext="+textAddr)
	}
	if !strings.Contains(assembler.Target, "wasm") && !strings.Contains(assembler.Target, "wasm32") {
		args = append(args, "-Wl,--build-id=none")
	}
	return args
}

func ApplyLdFlags(args []string, ldScript, textAddr string) []string {
	if ldScript != "" {
		args = append(args, "-T", ldScript)
	}
	if textAddr != "" {
		args = append(args, "-Ttext", textAddr)
	}
	if !strings.Contains(assembler.Target, "wasm") && !strings.Contains(assembler.Target, "wasm32") {
		args = append(args, "--build-id=none")
	}
	return args
}

func SetOutputFormat(format string) error {
	valid := map[string]bool{"elf32": true, "elf64": true, "bin": true}
	if !valid[format] {
		return fmt.Errorf("invalid output format: %s (supported: elf32, elf64, bin)", format)
	}
	assembler.OutputFormat = format
	return nil
}
