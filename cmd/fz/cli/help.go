/*
 * Copyright (c) 2026 ForgeZero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package cli

import "github.com/forgezero-cli/ForgeZero/cmd/fz/stdio"

const helpBody1 = `
fz – assembly & C build tool (ForgeZero `

const helpBody2 = `) · GPLv3 · (c) ForgeZero-cli

Usage:
  fz [options] (-asm <file> | -cc <file> | -dir <dir> | (no args with config))
  fz audit|sbom|doctor|verify|bench|pm|contribute [options]

Options:
  -asm <file>          Assembler source (.asm, .s, .S, .fasm)
  -cc <file>           C/C++ source (.c, .cpp, .cc, .cxx, .m)
  -dir <dir>           Build all supported files recursively
  -out <name>          Output binary name
  -out-obj <name>      Object file name
  -mode <auto|c|raw>   Linking mode (default: auto)
  -debug               Emit debug symbols
  -verbose             Print commands executed
  -keep-obj            Keep temporary object files
  -no-cache            Disable incremental cache
  -no-symbol-check     Skip duplicate symbol pre-check
  -sanitize            Enable sanitizers for C (default)
  -no-sanitize         Disable sanitizers
  -strict              Enable aggressive sanitizers (prefers clang)
  -toolchain <auto|zig>  Select toolchain
  -isolation <none|standard|strict>  FS isolation for build inputs
  -clean               Remove build artifacts
  -watch               Watch files and auto-rebuild
  -json                Output build report in JSON
  -config <file>       Config file (default: .fz.yaml)
  -man                 Generate roff man page and exit
  -format <elf32|elf64|bin>  Output format (default: elf64)
  -T <file>            Linker script
  -Ttext <addr>        Set text segment address
  -j <n>               Parallel jobs (0 = auto)
  -target <triple>     Target triple (default: x86_64-linux-gnu)
  -type <executable|static>  Build type
  -lib                 Shortcut for -type static
  -compile-commands    Generate compile_commands.json for LSP
  -init                Initialize project (.fz.yaml, .fzignore)
  -shell               Run interactive shell
  -autoBuild           Auto build project with auto backend
  -update              Update fz to latest version
  -profile <name>      Build profile: performance, balanced, power-saver
  -p <name>            Shorthand for -profile
  -contribute          Generate CONTRIBUTING_USER.md
  -musl                Compile with static musl toolchain
  -h, --help           Show this help
  -v, --version        Show version

Aegis Security & Integrity:
  doctor [options]     Self-audit: toolchain, permissions, platform (--root, --json)
  audit [options]      SAST scan: secrets, licenses, vendor keywords (--config, --vendor, --verbose, --json)
  sbom [options]       Generate SBOM (CycloneDX JSON, BLAKE3) (--config, --vendor, --target, --out, --json)
  verify [options]     Verify source tree BLAKE3 manifest (--root, --manifest, --update, --json)
  bench [options]      Nanosecond build phase profiler (--asm|--cc|--dir, --n, --json, --verbose)
  contribute           Generate contributor guide with environment checks

Package Manager (fz pm):
  add <repo> [version]   Clone and add package
  remove <name>          Remove package
  list                   Show installed packages
  update                 Update all packages
  catalog                List available packages
  search <keyword>       Search catalog
  install <name>         Install package from catalog (with hash verification)

Supported extensions: .asm, .s, .S, .fasm, .c, .cpp, .cc, .cxx, .m (Objective-C)

Examples:
  fz -asm boot.asm -format bin -out boot.bin
  fz -cc main.c -strict -verbose
  fz -dir ./src -out myapp -watch
  fz -json -cc test.c
  fz -dir . -clean
  fz -target arm-linux-gnueabihf -cc test.c -out test_arm
  fz -profile performance -cc main.c
  fz -p power-saver -dir ./src
  fz contribute
  fz sbom -out sbom.json
  fz doctor -json
  fz verify --update
  fz bench -dir ./src -json
  fz pm add github.com/user/repo
`

func HelpText() string {
	return helpBody1 + VersionCore + helpBody2
}

func PrintHelp() {
	stdio.WriteStderr(HelpText())
}