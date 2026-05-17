package man

import (
	"fmt"
	"time"
)

func GenerateManPage(version string) string {
	return fmt.Sprintf(`.TH fz 1 "%s" "fz %s" "User Commands"
.SH NAME
fz \- assemble projects with a single command
.SH SYNOPSIS
fz [OPTIONS] (\-asm <file> | \-cc <file> | \-dir <dir>)
.SH DESCRIPTION
fz is a build tool for assembly (NASM, GAS, FASM) and C projects.
It automates assembling, compiling, linking, caching, and provides features like watch mode,
JSON output, strict sanitizers, and duplicate symbol detection.
.SH OPTIONS
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
Config file path (default: .fz.yaml, fz.yaml, .fz.yml, fz.yml)
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
.SH AUTHORS
Alex Voste <alexvoste@proton.me>
.SH SEE ALSO
nasm(1), gcc(1), ld(1), clang(1)
`, time.Now().Format("Jan 2006"), version)
}
