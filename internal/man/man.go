/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package man

import "time"

func GenerateManPage(version string) string {
	head := ".TH fz 1 \"" + time.Now().Format("Jan 2006") + "\" \"fz " + version + "\" \"User Commands\"\n"
	body := `.SH NAME
fz \- assemble projects with a single command
.SH SYNOPSIS
fz [OPTIONS] (\-asm <file> | \-cc <file> | \-dir <dir>)
.SH DESCRIPTION
fz is a build tool with built-in NASM/FASM backends. External dependencies: NONE.
It automates assembling, compiling, linking, caching, and provides watch mode,
JSON output, strict sanitizers, and deterministic builds with Zig integration.
.SH OPTIONS
.TP
\fB\-\-format\fR <elf32|elf64|bin>
Output format: elf64 (default), elf32, bin (flat binary / bare-metal bootloader mode)
.TP
\fB\-\-asm\fR <file>
Assembler source file (.asm, .s, .S, .fasm)
.TP
\fB\-\-cc\fR <file>
C source file (compiled with -Wall -Wextra -Werror -Wpedantic -Wshadow -Wconversion)
.TP
\fB\-\-dir\fR <dir>
Build all supported files in directory (recursive)
.TP
\fB\-\-out\fR <name>
Output binary name
.TP
\fB\-\-out-obj\fR <name>
Object file name (single file only)
.TP
\fB\-\-mode\fR <auto|c|raw>
Linking mode (default: auto)
.TP
\fB\-\-debug\fR
Emit debug information (-g)
.TP
\fB\-\-verbose\fR
Print executed commands
.TP
\fB\-\-keep-obj\fR
Keep temporary object files when using -dir
.TP
\fB\-\-no-cache\fR
Disable incremental cache
.TP
\fB\-\-no-symbol-check\fR
Skip duplicate symbol pre‑check
.TP
\fB\-\-sanitize\fR
Enable sanitizers for C (default: true)
.TP
\fB\-\-no-sanitize\fR
Disable sanitizers
.TP
\fB\-\-strict\fR
Enable aggressive sanitizers (use-after-return, use-after-scope) – prefers clang
.TP
\fB\-\-toolchain\fR <auto|zig>
Select toolchain for C/C++ builds and linking
.TP
\fB\-\-clean\fR
Remove all build artifacts (.fz_objs, .fz_cache, binaries)
.TP
\fB\-\-watch\fR
Watch source files and rebuild automatically
.TP
\fB\-\-json\fR
Output build report in JSON format (CI/CD)
.TP
\fB\-\-config\fR <file>
Config file path (default: .fz.toml, .fz.yaml, fz.toml, fz.yaml, .fz.yml, fz.yml)
.TP
\fB\-\-man\fR
Generate roff man page and exit
.TP
\fB\-h, \-\-help\fR
Show this help
.TP
\fB\-v, \-\-version\fR
Show version
.SH EXAMPLES
fz -asm boot.asm
fz -cc main.c -strict -verbose
fz -dir ./src -out myapp -watch
fz -json -cc test.c
fz -dir . -clean
fz -asm boot.asm -format bin -out boot.bin
.SH AUTHORS
Alex Voste <alexvoste@proton.me>
.SH SEE ALSO
nasm(1), gcc(1), ld(1), clang(1)
`
	return head + body
}
