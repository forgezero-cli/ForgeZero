package initpkg

import (
	"errors"
	"os"

	"fz/internal/utils"
)

var readmeTemplate = []byte(`# Your Project

This project was initialized with [ForgeZero](https://github.com/forgezero-cli/ForgeZero) – a build tool for assembly and C.

## How to build

1. Edit .fz.yaml to configure source directories, output name, etc.
2. Run:

    fz 

or with custom flags:

    fz -dir ./src -out myapp -verbose

## Build options

- ` + "`-asm <file>`" + ` – assemble a single file (NASM, GAS, FASM)
- ` + "`-cc <file>`" + ` – compile a single C file (strict warnings)
- ` + "`-dir <dir>`" + ` – build all supported files recursively
- ` + "`-out <name>`" + ` – set output binary name
- ` + "`-mode auto|c|raw`" + ` – linking mode (auto = gcc → gcc -no-pie → ld)
- ` + "`-debug`" + ` – emit debug symbols (-g)
- ` + "`-verbose`" + ` – show executed commands
- ` + "`-keep-obj`" + ` – keep intermediate .o files
- ` + "`-no-cache`" + ` – disable incremental cache
- ` + "`-strict`" + ` – enable advanced sanitizers (use-after-return, etc.)
- ` + "`-watch`" + ` – auto‑rebuild on source changes
- ` + "`-json`" + ` – machine‑readable output for CI/CD
- ` + "`-clean`" + ` – remove all build artifacts
- ` + "`-format bin`" + ` – build flat binary (e.g., bootloader)

## .fz.yaml configuration

See the generated .fz.yaml file for all options. It supports:
- Multiple source directories (` + "`source_dirs`" + `)
- Exact file list (` + "`source_files`" + `)
- Exclude patterns (` + "`exclude`" + `) and include patterns (` + "`include`" + `)
- Libraries (` + "`libs`" + `)
- Custom flags for assembler, C compiler, linker

## .fzignore

You can list files/directories to ignore (like .gitignore). Syntax: glob patterns, e.g., ` + "`*.o`, `temp/`" + `.

## Example

    fz -asm boot.asm -format bin -out boot.bin
    qemu-system-x86_64 -drive format=raw,file=boot.bin

## License

MIT

`)

var yamlTemplate = []byte(`# fz configuration file

# Copyright (c) 2026 ForgeZero

# Source directories to scan recursively (optional, default: current directory)
source_dirs:
  - src
  - lib

# Explicit source files (overrides source_dirs if set)
# source_files:
#   - boot.asm
#   - main.c

# Patterns to exclude (glob syntax)
exclude:
  - test_*
  - temp/
  - "*.bak"

# Only include files matching these patterns (empty means all)
# include:
#   - "*.asm"
#   - "*.c"

# Libraries to link (passed as -l<name>)
# libs:
#   - m
#   - c

# Output binary name (default: derived from source or directory)
output: myprogram

# Linking mode: auto (gcc -> gcc -no-pie -> ld), c (gcc only), raw (ld only)
mode: auto

# Emit debug information
debug: false

# Print executed commands
verbose: false

# Keep intermediate object files
keep_obj: false

# Disable incremental cache
no_cache: false

# Custom flags for assembler, C compiler, and linker
# flags:
#   asm: ["-felf64", "-Ox"]
#   cc: ["-O2"]
#   ld: ["-T", "linker.ld"]

# Path to .fzignore file (default: .fzignore)
ignore_file: .fzignore
`)

var ignoreTemplate = []byte(`# fz ignore file
# Copyright (c) 2026 ForgeZero

# Ignore object files
*.o

# Ignore temporary editor files
*~
*.swp

# Ignore build directories
build/
dist/

# Ignore specific files
test_*
*.bak

# Ignore hidden directories
.fz_objs/
.fz_cache/
`)

func Run() error {
	if _, err := os.Stat(".fz.yaml"); err == nil {
		return errors.New(".fz.yaml already exists (not overwritten)")
	}
	if _, err := os.Stat(".fzignore"); err == nil {
		return errors.New(".fzignore already exists (not overwritten)")
	}
	if _, err := os.Stat("README.md"); err != nil {
		if err := utils.SecureWriteFile("README.md", readmeTemplate); err != nil {
			return err
		}
	}
	if err := utils.SecureWriteFile(".fz.yaml", yamlTemplate); err != nil {
		return err
	}
	if err := utils.SecureWriteFile(".fzignore", ignoreTemplate); err != nil {
		return err
	}
	return nil
}