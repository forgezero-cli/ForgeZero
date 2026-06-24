package linker

import (
	"errors"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/assembler"
)

var validFormats = []string{"elf32", "elf64", "bin"}

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
	for _, f := range validFormats {
		if f == format {
			assembler.OutputFormat = format
			return nil
		}
	}
	return errors.New("invalid output format: " + format + " (supported: elf32, elf64, bin)")
}