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

package helpers

const helpBody1 = `
fz – assembly & C build tool (ForgeZero `

const helpBody2 = `)

ForgeZero includes built-in NASM/FASM backends. External dependencies: NONE.

Usage:
  fz [options] (-asm <file> | -cc <file> | -dir <dir> | (no args with config))
  fz audit [options]
  fz sbom [options]
  fz doctor [options]
  fz verify [options]
  fz bench [options]
  fz pm <subcommand> [args]
  fz contribute

Options:
  -asm <file>            Assembler source (.asm, .s, .S, .fasm)
  -cc <file>             C source (strict warnings enabled)
  -dir <dir>             Build all supported files recursively
  -out <name>            Output binary name
  -out-obj <name>        Object file name (single file only)
  -mode <auto|c|raw>     Linking mode (default: auto)
  -debug                 Emit debug symbols (-g)
  -verbose               Print every command executed
  -keep-obj              Keep temporary object files (when using -dir)
  -no-cache              Disable incremental cache
  -no-symbol-check       Skip duplicate symbol pre‑check
  -sanitize              Enable sanitizers for C (default: true)
  -no-sanitize           Disable sanitizers
  -strict                Enable aggressive sanitizers (use-after-return, use-after-scope) – prefers clang
  -toolchain <auto|zig>  Select toolchain: auto or zig
  -isolation <none|standard|strict>  File system isolation mode for build inputs
  -clean                 Remove all build artifacts (.fz_objs, .fz_cache, binaries)
  -watch                 Watch files and auto‑rebuild
  -json                  Output build report in JSON (for CI/CD)
  -config <file>         Config file (default: .fz.yaml, fz.yaml, .fz.yml, fz.yml)
  -man                   Generate roff man page and exit
  -format <elf32|elf64|bin> Output format: elf64 (default), elf32, bin (flat binary / bare-metal bootloader mode)
  -T <file>              Linker script (passed to ld)
  -Ttext <addr>          Set text segment address
  -j <n>                 Number of parallel jobs (0 = auto = CPU cores)
  -target <triple>       Target triple (default: x86_64-linux-gnu, experimental: wasm)
  -type <executable|static> Build type: executable (default) or static (library)
  -lib                   Shortcut for -type static
  -compile-commands      Generate compile_commands.json for LSP and exit
  -init                  Initialize project: create .fz.yaml and .fzignore
  -shell                 Run interactive shell
  -build                 Auto build project with auto backend (c/c++, asm)
  -update                Update fz to the latest version
  -profile <name>        Build profile: performance, balanced, power-saver (default: balanced)
  -p <name>              Shorthand for -profile
  -contribute            Generate CONTRIBUTING_USER.md with contribution guidance
  -musl                  Compile with static musl toolchain (x86_64, RISC-V)
  -h, --help             Show this help
  -v, --version          Show version

  For testing project: fz -alex 

Examples:
  fz -asm boot.asm
  fz -cc main.c -strict -verbose
  fz -dir ./src -out myapp -watch
  fz -json -cc test.c
  fz -dir . -clean
  fz -asm boot.asm -format bin -out boot.bin
  fz -target arm-linux-gnueabihf -cc test.c -out test_arm
  fz -profile performance -cc main.c
  fz -p power-saver -dir ./src
  fz contribute
  fz sbom -out sbom.json
  fz doctor -root .
  fz doctor -json
  fz verify --update
  fz bench -dir ./src -json

Supported extensions: .asm, .s, .S, .fasm, .c, .cpp, .cc, .cxx, .m (Objective-C)

Aegis Security & Integrity (v3.1.0):
  doctor [options]        Self-audit: toolchain reachability, R/W permissions, platform
                          -root <dir>   project root (default: cwd)
                          -json         machine-readable report; exit 1 if unhealthy
  audit [options]         SAST scan: secrets, license risks, vendor keyword matches
                          -config -vendor -verbose -json
  sbom [options]          Supply Chain (SBOM): CycloneDX JSON, BLAKE3 per component
                          -config -vendor -target -out <path> -json
  verify [options]        Source tree BLAKE3 manifest integrity
                          -root <dir> -manifest <file> -update -json
  bench [options]         Nanosecond build phase profiler
                          -asm|-cc|-dir -out -mode -target -toolchain -n -json -verbose
  contribute              Generate contributor guide with environment checks

Aegis technical (internal architecture):
  FileSystem VFS          internal/fs: Unix or Windows backend via build tags
                          OpenVerified: Lstat + SameFile TOCTOU hardening on reads
                          SecureWriteFile: temp 0600, close, atomic rename
  RunCommand              All subprocesses (git, ar, zig, fasm, gcc, ld, nasm, …)
                          exec.LookPath resolution, ValidateCLIArg per token,
                          deterministicEnv (LC_ALL=C, TZ=UTC, SOURCE_DATE_EPOCH)

Package Manager (fz pm):
  add <repo> [version]    Clone and add package to project
  remove <name>           Remove installed package
  list                    Show installed packages
  update                  Update all installed packages
  catalog                 List available packages from catalog
  search <keyword>        Search catalog
  install <name>          Install package from catalog (with hash verification)
`

func HelpText() string {
	return helpBody1 + VersionCore + helpBody2
}

func PrintHelp() {
	WriteStderr(HelpText())
}