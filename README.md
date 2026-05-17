# ☘️ ForgeZero (fz) — Complete Documentation

<div align="center">
  <img src="pictures/fz.jpg" alt="ForgeZero Logo" width="180" />
  
  <br />
  <br />
  
</div>

![Build Status](https://github.com/alexvoste/ForgeZero/actions/workflows/go.yml/badge.svg)
![Go Version](https://img.shields.io/github/go-mod/go-version/alexvoste/ForgeZero)
![License](https://img.shields.io/github/license/alexvoste/ForgeZero)
![Commits](https://img.shields.io/github/commits-since/alexvoste/ForgeZero/v1.3.0)

> **Version:** 1.4.0 &nbsp;·&nbsp; **Language:** Go &nbsp;·&nbsp; **License:** MIT &nbsp;·&nbsp; **Platform:** Linux · Windows · macOS

ForgeZero is a high-performance, zero-overhead build tool for assembly and C developers. It wraps NASM, GAS, FASM, GCC, Clang, and LD into a single unified command-line interface — no Makefiles, no build scripts, no configuration required to get started.

One command. Any assembler. Any platform.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Requirements](#2-requirements)
3. [Installation](#3-installation)
   - 3.1 [Linux — Debian / Ubuntu](#31-linux--debian--ubuntu)
   - 3.2 [Linux — Fedora / RHEL / CentOS](#32-linux--fedora--rhel--centos)
   - 3.3 [Linux — Arch Linux / Manjaro](#33-linux--arch-linux--manjaro)
   - 3.4 [Linux — openSUSE](#34-linux--opensuse)
   - 3.5 [macOS](#35-macos)
   - 3.6 [Windows](#36-windows)
   - 3.7 [Build from Source (All Platforms)](#37-build-from-source-all-platforms)
   - 3.8 [Go Install](#38-go-install)
4. [Quick Start](#4-quick-start)
5. [Supported Languages & Extensions](#5-supported-languages--extensions)
6. [Build Modes](#6-build-modes)
   - 6.1 [Single File Mode](#61-single-file-mode)
   - 6.2 [Directory Mode](#62-directory-mode)
   - 6.3 [Configuration File Mode](#63-configuration-file-mode)
7. [CLI Reference](#7-cli-reference)
8. [Linking Modes](#8-linking-modes)
9. [C Compilation](#9-c-compilation)
   - 9.1 [Strict Warning Flags](#91-strict-warning-flags)
   - 9.2 [Sanitizers](#92-sanitizers)
10. [Internal Mechanisms](#10-internal-mechanisms)
    - 10.1 [Build Cache](#101-build-cache)
    - 10.2 [Pre-link Symbol Check](#102-pre-link-symbol-check)
    - 10.3 [Watch Mode](#103-watch-mode)
    - 10.4 [JSON Output](#104-json-output)
    - 10.5 [Clean](#105-clean)
11. [Configuration File Reference](#11-configuration-file-reference)
12. [Assembler Backends](#12-assembler-backends)
    - 12.1 [NASM (.asm)](#121-nasm-asm)
    - 12.2 [GAS (.s / .S)](#122-gas-s--s)
    - 12.3 [FASM (.fasm)](#123-fasm-fasm)
13. [Examples](#13-examples)
14. [Exit Codes](#14-exit-codes)
15. [Troubleshooting](#15-troubleshooting)
16. [Roadmap](#16-roadmap)
17. [Contributing](#17-contributing)
18. [License](#18-license)

---

## 1. Overview

ForgeZero removes the friction between writing assembly (or C) code and running it. Instead of managing assembler flags, linker invocations, and object file paths by hand, you point `fz` at a source file or directory and it handles everything:

- Detects the file type and selects the correct assembler backend automatically.
- Compiles each source file into an object file with appropriate flags.
- Checks for duplicate global symbols across all objects before linking.
- Links everything into a single binary using the most appropriate linker.
- Caches compiled objects so unchanged files are never recompiled.
- Optionally watches the filesystem and rebuilds on every save.
- Emits structured JSON build reports for CI/CD integration.

ForgeZero is intentionally lightweight — a single statically compiled Go binary with no runtime dependencies beyond the standard assembler/compiler toolchain.

---

## 2. Requirements

### Assembler and compiler tools

| Source type   | Required tool        | Notes |
|---------------|----------------------|-------|
| `.asm`        | `nasm`               | x86/x86-64 Intel syntax |
| `.s` / `.S`   | `gcc` (drives `as`)  | AT&T syntax; `.S` files are C-preprocessed first |
| `.fasm`       | `fasm`               | Must be downloaded separately from flatassembler.net |
| `.c`          | `gcc` or `clang`     | `clang` preferred when `-strict` is used |

### Linker tools

| Linker  | Required for |
|---------|--------------|
| `gcc`   | Default linking, C runtime support |
| `ld`    | Raw linking (`-mode raw`) |
| `clang` | Strict sanitizer mode (`-strict`) |

### Optional tools (used internally)

| Tool       | Purpose |
|------------|---------|
| `nm`       | Pre-link duplicate symbol check (primary) |
| `objdump`  | Fallback for symbol check |
| `readelf`  | Second fallback for symbol check |

### Go version (build from source only)

Go **1.21** or later is required to build `fz` from source.

---

## 3. Installation

### 3.1 Linux — Debian / Ubuntu

**Install system dependencies:**

```bash
sudo apt update
sudo apt install -y nasm gcc binutils
```

**Install Clang (optional, for `-strict` mode):**

```bash
sudo apt install -y clang
```

**Install FASM (optional, for `.fasm` files):**

```bash
wget https://flatassembler.net/fasm-1.73.32.tgz
tar -xzf fasm-1.73.32.tgz
sudo cp fasm/fasm /usr/local/bin/
chmod +x /usr/local/bin/fasm
```

**Install ForgeZero via Go:**

```bash
go install github.com/alexvoste/ForgeZero/cmd/fz@latest
```

Ensure `~/go/bin` is on your `PATH`:

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

Verify:

```bash
fz -v
```

---

### 3.2 Linux — Fedora / RHEL / CentOS

**Install system dependencies:**

```bash
# Fedora
sudo dnf install -y nasm gcc binutils clang

# RHEL / CentOS — enable EPEL first for nasm
sudo dnf install -y epel-release
sudo dnf install -y nasm gcc binutils clang
```

**Install ForgeZero:**

```bash
go install github.com/alexvoste/ForgeZero/cmd/fz@latest
```

---

### 3.3 Linux — Arch Linux / Manjaro

**Install system dependencies:**

```bash
sudo pacman -S --noconfirm nasm gcc binutils clang

# FASM is available in the AUR
yay -S fasm
```

**Install ForgeZero:**

```bash
go install github.com/alexvoste/ForgeZero/cmd/fz@latest
```

---

### 3.4 Linux — openSUSE

**Install system dependencies:**

```bash
sudo zypper install -y nasm gcc binutils clang
```

**Install ForgeZero:**

```bash
go install github.com/alexvoste/ForgeZero/cmd/fz@latest
```

---

### 3.5 macOS

macOS support is in progress. The following setup works for most use cases today.

**Install Homebrew (if not already installed):**

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

**Install dependencies:**

```bash
brew install nasm gcc go
```

> **Note:** macOS ships `clang` as the system compiler under the `gcc` alias via Xcode Command Line Tools. For full GCC, `brew install gcc` installs it as `gcc-14` (or similar). ForgeZero uses whatever `gcc` resolves to on your `PATH`.

**Install ForgeZero:**

```bash
go install github.com/alexvoste/ForgeZero/cmd/fz@latest
```

Add Go's bin directory to your shell profile:

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

Verify:

```bash
fz -v
```

---

### 3.6 Windows

Windows support is in progress. The recommended approach is **WSL2** (Windows Subsystem for Linux), which provides a full Linux environment and the best compatibility with all ForgeZero features.

#### Option A — WSL2 (Recommended)

1. Open **PowerShell as Administrator** and run:

```powershell
wsl --install
```

1. Restart your machine. Open **Ubuntu** from the Start menu.

2. Inside the WSL2 terminal, follow the [Debian/Ubuntu instructions](#31-linux--debian--ubuntu).

3. Access your Windows files from WSL2 at `/mnt/c/Users/<YourName>/`.

#### Option B — Native Windows (Experimental)

Native Windows support requires manual toolchain setup via MSYS2.

**Step 1 — Install MSYS2:**

Download and run the installer from [msys2.org](https://www.msys2.org/). After installation, open the **MSYS2 MinGW 64-bit** terminal and run:

```bash
pacman -Syu
pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-binutils mingw-w64-x86_64-clang
```

**Step 2 — Install NASM for Windows:**

Download the Windows installer from [nasm.us](https://www.nasm.us/pub/nasm/releasebuilds/). Run it and note the installation path (e.g. `C:\Program Files\NASM`).

**Step 3 — Add tools to your PATH:**

Open **System Properties → Advanced → Environment Variables**. Add the following to the `Path` variable:

```
C:\msys64\mingw64\bin
C:\Program Files\NASM
C:\Users\<YourName>\go\bin
```

**Step 4 — Install Go for Windows:**

Download from [go.dev/dl](https://go.dev/dl/) and run the installer.

**Step 5 — Install ForgeZero:**

Open **Command Prompt** or **PowerShell**:

```powershell
go install github.com/alexvoste/ForgeZero/cmd/fz@latest
```

Or build from source:

```powershell
git clone https://github.com/alexvoste/ForgeZero.git
cd ForgeZero
go build -o fz.exe ./cmd/fz/main.go
```

Move `fz.exe` to a directory on your `PATH`.

> **Known limitation:** `-sanitize` and `-strict` require Clang with AddressSanitizer support compiled for Windows. This is available via the LLVM official Windows release but requires additional setup beyond MSYS2. Basic NASM assembly and GCC linking work without any extra configuration.

---

### 3.7 Build from Source (All Platforms)

```bash
git clone https://github.com/alexvoste/ForgeZero.git
cd ForgeZero
go build -o fz ./cmd/fz/main.go    # Linux / macOS
go build -o fz.exe ./cmd/fz/main.go  # Windows
```

Run tests:

```bash
go test ./...
```

Install to `PATH`:

```bash
# Linux / macOS
sudo mv fz /usr/local/bin/

# Windows (PowerShell, as Administrator)
Move-Item fz.exe C:\Windows\System32\fz.exe
```

---

### 3.8 Go Install

The simplest method if Go is already configured:

```bash
go install github.com/alexvoste/ForgeZero/cmd/fz@latest
```

The binary lands in `$GOPATH/bin`. Verify installation:

```bash
fz -v
```

---

## 4. Quick Start

**Assemble a NASM file:**

```bash
fz -asm hello.asm
./hello
```

**Compile a C file:**

```bash
fz -cc main.c
./main
```

**Build an entire directory:**

```bash
fz -dir ./src
./src
```

---

## 5. Supported Languages & Extensions

| Extension | Language | Backend | Notes |
|-----------|----------|---------|-------|
| `.asm` | Assembly | NASM | x86/x86-64, Intel syntax, ELF64 |
| `.s` | Assembly | GAS via `gcc -c` | AT&T syntax |
| `.S` | Assembly | GAS via `gcc -c` | AT&T syntax + C preprocessor |
| `.fasm` | Assembly | FASM | Requires separate install |
| `.c` | C | GCC or Clang | Strict flags + sanitizers by default |

---

## 6. Build Modes

### 6.1 Single File Mode

Compiles and links a single source file into a binary.

```bash
fz -asm program.asm
fz -cc main.c
```

- Output binary name is derived from the source filename (`program.asm` → `program`).
- A single object file is created and removed after linking unless `-keep-obj` is set.
- Override the binary name with `-out` and the object file name with `-out-obj`.

### 6.2 Directory Mode

Recursively scans a directory for all supported source files, compiles each to a uniquely named object file, then links everything into a single binary.

```bash
fz -dir ./src
```

**Object file naming** — names are generated to prevent collisions across subdirectories:

| Source file | Object file |
|-------------|-------------|
| `src/hello.asm` | `hello_asm.o` |
| `src/hello.s` | `hello_s.o` |
| `src/sub/hello.asm` | `sub_hello_asm.o` |

Object files live in `.fz_objs/` and are removed after linking unless `-keep-obj` is passed. The output binary is named after the directory (`src` → `src` on Linux/macOS, `src.exe` on Windows).

### 6.3 Configuration File Mode

ForgeZero automatically searches the working directory for a config file in this order:

1. `.fz.yaml`
2. `fz.yaml`
3. `.fz.yml`
4. `fz.yml`

Run without any flags to use the config:

```bash
fz
```

Use `-config` to specify a path explicitly:

```bash
fz -config ./configs/release.yaml
```

CLI flags always take precedence over config file values.

---

## 7. CLI Reference

### Synopsis

```
fz [options]
```

At least one of `-asm`, `-cc`, or `-dir` is required (or a valid config file must be present in the working directory).

### Full Flag Reference

| Flag | Argument | Default | Description |
|------|----------|---------|-------------|
| `-asm` | `<file>` | — | Assemble the given assembly source file. |
| `-cc` | `<file>` | — | Compile the given C source file. |
| `-dir` | `<dir>` | — | Recursively build all supported files in the directory. |
| `-out` | `<name>` | Derived from source | Name of the output binary. |
| `-out-obj` | `<name>` | `<basename>.o` | Object file name (single-file mode only). |
| `-mode` | `auto\|c\|raw` | `auto` | Linking mode. See [Linking Modes](#8-linking-modes). |
| `-debug` | — | off | Pass `-g` to the assembler/compiler to emit debug symbols. |
| `-verbose` | — | off | Print each external command to stdout before running it. |
| `-keep-obj` | — | off | Preserve object files after linking (directory mode). |
| `-no-cache` | — | off | Disable the build cache; always recompile every file. |
| `-no-symbol-check` | — | off | Skip the pre-link duplicate symbol check. |
| `-sanitize` | — | **on** | Enable `-fsanitize=address,undefined` for C. Disable with `-sanitize=false`. |
| `-strict` | — | off | Stricter sanitizers + use-after-return/scope checks. Prefers `clang`. |
| `-json` | — | off | Suppress normal output; emit a JSON build report to stdout. |
| `-watch` | — | off | Watch source files for changes and rebuild automatically. |
| `-clean` | — | off | Remove all build artifacts and exit. |
| `-config` | `<file>` | auto-detect | Path to a YAML configuration file. |
| `-timeout` | `<sec>` | `60` | Timeout in seconds for each sub-command. |
| `-h`, `--help` | — | — | Print help and exit. |
| `-v`, `--version` | — | — | Print version and exit. |

---

## 8. Linking Modes

The `-mode` flag controls how compiled object files are linked into a final binary.

### `auto` (default)

ForgeZero tries linkers in sequence until one succeeds:

1. `gcc` — standard linking with libc and C runtime.
2. `gcc -no-pie` — position-dependent executable; needed when code assumes fixed load addresses.
3. `ld` — raw system linker; last resort.

When `-strict` is active, `clang` with full sanitizer flags is tried first.

Use `auto` for the vast majority of projects — it handles the widest range of code without manual tuning.

### `c` — Force GCC / Clang

```bash
fz -asm program.asm -mode c
fz -cc main.c -mode c
```

Always links using `gcc` (or `clang` in strict mode). Required when code calls libc functions (`printf`, `malloc`, `exit`, etc.) or depends on C runtime initialization (`__libc_start_main`).

### `raw` — Force LD

```bash
fz -asm kernel.asm -mode raw -out kernel.bin
```

Bypasses GCC entirely and invokes `ld` directly. Produces minimal binaries with no C runtime overhead. Suitable for:

- OS kernels and bootloaders
- Bare-metal firmware
- Programs that define their own `_start` and use raw syscalls
- Embedded targets requiring full control over the binary layout

> **Warning:** Raw-linked binaries cannot reference any libc symbol. If your code calls `printf` or any standard library function, use `-mode c`.

---

## 9. C Compilation

### 9.1 Strict Warning Flags

Every `.c` file compiled by `fz` receives these flags unconditionally:

```
-Wall -Wextra -Werror -Wpedantic -Wshadow -Wconversion
```

Any warning is treated as an error and stops the build immediately.

| Flag | Effect |
|------|--------|
| `-Wall` | Enables most common warnings |
| `-Wextra` | Enables additional warnings beyond `-Wall` |
| `-Werror` | Promotes all warnings to errors |
| `-Wpedantic` | Enforces strict ISO C compliance |
| `-Wshadow` | Warns when a local variable shadows an outer one |
| `-Wconversion` | Warns on implicit type conversions that may lose precision |

This is intentional — ForgeZero enforces clean, portable C code by default and gives no option to silently ignore warnings.

### 9.2 Sanitizers

ForgeZero enables runtime sanitizers by default for all C compilation to catch memory errors and undefined behavior early.

**Standard mode (default — always enabled unless `-sanitize=false`):**

```
-fsanitize=address
-fsanitize=undefined
```

Detects heap and stack overflows, use-after-free, integer overflow, null dereference, and other undefined behavior at runtime.

**Strict mode (`-strict`):**

```
-fsanitize=address
-fsanitize=undefined
-fsanitize-address-use-after-return=always    (Clang only)
-fsanitize-address-use-after-scope
```

When `-strict` is active, `fz` prefers `clang` because it supports `-fsanitize-address-use-after-return=always`. If only `gcc` is available, `-fsanitize-address-use-after-scope` is applied but `use-after-return` is skipped with a warning.

**Disable sanitizers (release build / benchmarking):**

```bash
fz -cc main.c -sanitize=false
```

---

## 10. Internal Mechanisms

### 10.1 Build Cache

ForgeZero caches compiled object files in `.fz_cache/` to skip recompilation of unchanged sources.

**Cache key** is computed from:

- SHA-256 hash of the source file contents
- `-debug` flag state (`true`/`false`)
- `-mode` value (`auto`, `c`, or `raw`)

If all three match an existing cache entry, the stored object file is reused. The assembler/compiler is never invoked for that file.

**Disable caching:**

```bash
fz -dir ./src -no-cache
```

**Clear cache:**

```bash
fz -dir . -clean
```

The cache is stored as plain files under `.fz_cache/` and can be safely deleted at any time. The next build regenerates it.

---

### 10.2 Pre-link Symbol Check

Before invoking the linker, `fz` scans all compiled object files for duplicate global symbol definitions. This catches conflicts — such as `_start` or a global function defined in two files — before the linker emits a cryptic error.

**Tools used (in order of preference):** `nm` → `objdump` → `readelf`

If a conflict is found, `fz` reports which files define the duplicate symbol and exits with code `1`, without attempting to link.

**Disable the check:**

```bash
fz -dir ./src -no-symbol-check
```

Use this when intentionally relying on weak symbols or linker scripts that resolve conflicts at link time.

---

### 10.3 Watch Mode

Watch mode monitors source files (and the config file, if present) for filesystem changes and triggers a rebuild automatically.

```bash
fz -dir ./src -watch
fz -asm main.asm -watch
```

Uses [fsnotify](https://github.com/fsnotify/fsnotify) for cross-platform filesystem event detection. Rebuilds are debounced with a **500 ms** delay — multiple rapid saves within that window produce only one rebuild.

Press `Ctrl+C` to exit.

---

### 10.4 JSON Output

When `-json` is passed, all standard output is suppressed. On completion (or on error), a single JSON object is written to stdout:

```json
{
  "status": "success",
  "exit_code": 0,
  "duration_ms": 342,
  "binary": "./src",
  "source_files": ["src/main.asm", "src/utils.asm"],
  "object_files": ["main_asm.o", "utils_asm.o"],
  "error": null
}
```

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | `"success"` or `"error"` |
| `exit_code` | int | `0` on success, `1` on build error, `2` on argument error |
| `duration_ms` | int | Total build duration in milliseconds |
| `binary` | string | Path to the output binary (`null` on error) |
| `source_files` | array | All source files processed |
| `object_files` | array | All object files produced |
| `error` | string | Error message (`null` on success) |

**Example CI/CD integration (bash):**

```bash
result=$(fz -dir ./src -json)
status=$(echo "$result" | jq -r '.status')
duration=$(echo "$result" | jq -r '.duration_ms')

if [ "$status" != "success" ]; then
  echo "Build failed after ${duration}ms: $(echo "$result" | jq -r '.error')"
  exit 1
fi

echo "Build succeeded in ${duration}ms"
```

---

### 10.5 Clean

The `-clean` flag removes all artifacts produced by `fz`:

```bash
fz -dir . -clean
```

Deleted items:

- `.fz_objs/` — temporary object files from directory mode
- `.fz_cache/` — build cache
- Binaries named `{dirname}.out` or `{dirname}.exe`
- All `.o` files in the working tree
- All executable files (files with the `+x` bit) that are not recognized source files

> **Caution:** Clean identifies executables by the `+x` permission bit. Avoid running `-clean` in directories containing pre-built third-party binaries you did not intend to delete.

---

## 11. Configuration File Reference

ForgeZero accepts YAML configuration files. The file is searched automatically in this order: `.fz.yaml`, `fz.yaml`, `.fz.yml`, `fz.yml`. Use `-config <path>` to specify explicitly.

CLI flags always override config file values.

**Full annotated example:**

```yaml
# fz.yaml

# Source — choose one:
source_dir: ./src         # Build all supported files recursively
# source_file: main.asm   # Or build a single file

# Output
output: myprogram         # Name of the final binary

# Build options
mode: auto                # auto | c | raw
debug: false              # Include debug symbols (-g)
verbose: false            # Print all invoked commands
keep_obj: false           # Keep object files after linking
no_cache: false           # Disable build cache

# C-specific
sanitize: true            # Enable ASan + UBSan
strict: false             # Enable stricter sanitizers, prefer clang

# File filtering
exclude:                  # Glob patterns — matching files/dirs are skipped
  - vendor/
  - "*_test.asm"
  - legacy/

# Extra flags passed directly to assembler and linker
flags:
  asm:                    # Appended to nasm / gcc / fasm invocations
    - -DDEBUG_BUILD
    - -I./include
  ld:                     # Appended to the linker invocation
    - -lm
    - -lpthread
```

**Field reference:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `source_dir` | string | — | Directory to build recursively |
| `source_file` | string | — | Single source file to build |
| `output` | string | auto | Output binary name |
| `mode` | string | `auto` | Linking mode: `auto`, `c`, or `raw` |
| `debug` | bool | `false` | Emit debug symbols |
| `verbose` | bool | `false` | Verbose command output |
| `keep_obj` | bool | `false` | Preserve object files after linking |
| `no_cache` | bool | `false` | Disable build cache |
| `sanitize` | bool | `true` | Enable sanitizers for C |
| `strict` | bool | `false` | Strict sanitizer mode, prefers `clang` |
| `exclude` | list | — | Glob patterns for files/dirs to ignore |
| `flags.asm` | list | — | Extra flags appended to assembler invocations |
| `flags.ld` | list | — | Extra flags appended to linker invocations |

---

## 12. Assembler Backends

### 12.1 NASM (.asm)

**Command:** `nasm -felf64 <file> -o <output.o>`

NASM (Netwide Assembler) is the most widely-used x86/x86-64 assembler on Linux. It uses Intel syntax and outputs ELF64 object files.

```asm
; hello.asm — print "Hello, World!" to stdout and exit
section .data
    msg  db "Hello, World!", 0x0a
    len  equ $ - msg

section .text
    global _start

_start:
    mov rax, 1          ; sys_write
    mov rdi, 1          ; fd = stdout
    mov rsi, msg        ; buffer
    mov rdx, len        ; count
    syscall

    mov rax, 60         ; sys_exit
    xor rdi, rdi        ; exit code 0
    syscall
```

```bash
fz -asm hello.asm
./hello
```

---

### 12.2 GAS (.s / .S)

**Command:** `gcc -c <file> -o <output.o>`

GAS (GNU Assembler) uses AT&T syntax. Files with the `.S` extension are first passed through the C preprocessor, enabling `#include`, `#define`, and conditional compilation.

```asm
# hello.s — AT&T syntax
.section .data
    msg: .ascii "Hello, World!\n"
    len = . - msg

.section .text
    .global _start

_start:
    movq  $1,   %rax    # sys_write
    movq  $1,   %rdi    # stdout
    movq  $msg, %rsi
    movq  $len, %rdx
    syscall

    movq  $60,  %rax    # sys_exit
    xorq  %rdi, %rdi
    syscall
```

```bash
fz -asm hello.s
./hello
```

---

### 12.3 FASM (.fasm)

**Command:** `fasm <file> <output.o>`

FASM (Flat Assembler) is a self-hosting assembler with a powerful macro system. It must be downloaded separately from [flatassembler.net](https://flatassembler.net).

```asm
; hello.fasm
format ELF64 executable
entry _start

segment readable writeable
    msg db "Hello, World!", 10
    len = $ - msg

segment readable executable
_start:
    mov rax, 1
    mov rdi, 1
    mov rsi, msg
    mov rdx, len
    syscall

    mov rax, 60
    xor rdi, rdi
    syscall
```

```bash
fz -asm hello.fasm
./hello
```

---

## 13. Examples

### Minimal builds

```bash
fz -asm hello.asm       # NASM
fz -asm hello.s         # GAS
fz -asm hello.fasm      # FASM
fz -cc main.c           # C
```

---

### Debug build with verbose output

```bash
fz -asm hello.asm -debug -verbose
```

Passes `-g` to NASM and prints every invoked command. Attach GDB afterward:

```bash
gdb ./hello
```

---

### Bare-metal / bootloader binary

```bash
fz -asm boot.asm -mode raw -out boot.bin
```

Calls `ld` directly; no C runtime, no ELF overhead beyond the object format.

---

### C with strict sanitizers

```bash
fz -cc main.c -strict
```

Compiles with maximum warning flags and all sanitizer checks. Prefers `clang` for `use-after-return` detection.

---

### Build a full project directory

```bash
fz -dir ./src
```

All `.asm`, `.s`, `.S`, `.fasm`, and `.c` files under `./src/` are compiled and linked into a single binary.

---

### Directory build with JSON output (CI/CD)

```bash
fz -dir ./src -json | tee build_report.json
```

---

### Watch mode during development

```bash
fz -dir ./kernel -watch
```

Rebuilds automatically on every saved change. Debounced at 500 ms.

---

### Custom output and object file names

```bash
fz -asm code.s -out-obj build/code.o -out build/myprog
```

---

### Disable sanitizers for release

```bash
fz -cc main.c -sanitize=false -out main_release
```

---

### Keep object files for inspection

```bash
fz -dir ./src -keep-obj -verbose
ls .fz_objs/
```

---

### Using a configuration file

```bash
# Automatically detected fz.yaml in current directory
fz

# Explicit config path
fz -config ./configs/release.yaml
```

---

### Clean all build artifacts

```bash
fz -dir . -clean
```

---

## 14. Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success — binary was produced without errors. |
| `1` | Build error — assembler, compiler, or linker failed; or a duplicate global symbol was detected. Check stderr for details. |
| `2` | Argument error — invalid or missing flags, source file not found, or unreadable configuration file. |

---

## 15. Troubleshooting

### `fz: command not found`

Ensure Go's binary directory is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Add to `~/.bashrc` or `~/.zshrc` to persist across sessions.

---

### `nasm: command not found`

```bash
sudo apt install nasm          # Debian / Ubuntu
sudo dnf install nasm          # Fedora
sudo pacman -S nasm            # Arch
brew install nasm              # macOS
pacman -S mingw-w64-x86_64-nasm  # Windows MSYS2
```

---

### `fasm: command not found`

FASM is not in standard repositories. Download from [flatassembler.net](https://flatassembler.net):

```bash
wget https://flatassembler.net/fasm-1.73.32.tgz
tar -xzf fasm-1.73.32.tgz
sudo cp fasm/fasm /usr/local/bin/
chmod +x /usr/local/bin/fasm
```

---

### `undefined reference to _start`

Your source is missing an entry point. Define `_start` explicitly:

```asm
; NASM
global _start
_start:
    ; ...

# GAS
.global _start
_start:
    # ...
```

Or, if you want a `main` function with C runtime initialization:

```bash
fz -asm program.asm -mode c
```

---

### Binary crashes immediately (segfault on startup)

Likely cause: `-mode raw` was used, but the code references libc symbols. Switch to:

```bash
fz -asm program.asm -mode c
```

---

### Pre-link duplicate symbol error

`fz` detected that two or more object files define the same global symbol. Check your sources for conflicting `global` declarations (NASM) or `.global` directives (GAS). Rename one of them.

To skip the check when using weak symbols intentionally:

```bash
fz -dir ./src -no-symbol-check
```

---

### Sanitizer error at runtime

AddressSanitizer or UBSan detected a bug. The output includes the exact source file and line number. Fix the issue, then rerun. To temporarily disable sanitizers:

```bash
fz -cc main.c -sanitize=false
```

---

### Build hangs / times out

The default per-command timeout is 60 seconds. Increase for large projects:

```bash
fz -asm big_program.asm -timeout 300
```

---

### Cache returns stale results

If you suspect the cache is wrong (e.g. after changing assembler flags that `fz` does not track in the cache key), clear it and do a clean rebuild:

```bash
fz -dir . -clean
fz -dir ./src
```

Or do a single one-off build without the cache:

```bash
fz -dir ./src -no-cache
```

---

### Watch mode does not detect changes on WSL2

WSL2 has known issues with `inotify`-based watching when files are edited from Windows applications (e.g. VS Code on the Windows side). Edit files from within the WSL2 terminal to get reliable events. This is a WSL2 kernel limitation, not a ForgeZero bug.

---

### Windows: `gcc` not found

Make sure the MSYS2 `mingw64\bin` directory is added to your Windows `PATH`:

```
C:\msys64\mingw64\bin
```

Verify in PowerShell:

```powershell
gcc --version
```

---

## 16. Roadmap

| Feature | Status |
|---------|--------|
| `exclude` patterns in config file | Planned |
| `-asm-flag` and `-ld-flag` CLI flags for custom pass-through flags | Planned |
| Colored terminal output (green success / red error) | Planned |
| C++ support (`.cpp`, `.cxx`) with `g++` / `clang++` | Planned |
| GDB integration and improved debug workflow | Planned |
| Man page (`man fz`) | Planned |
| Windows native support without WSL2 | In progress |
| macOS full support and testing | In progress |

---

## 17. Contributing

Contributions are welcome: bug reports, feature requests, documentation improvements, and code patches.

1. **Open an issue** before starting significant work to align on the approach.
2. **Fork the repository** and create a descriptive feature branch (`feature/watch-debounce`, `fix/nasm-elf32`).
3. **Write tests** for new behavior and ensure existing tests pass:

   ```bash
   go test ./...
   ```

4. **Submit a Pull Request** with a clear description of the change and the problem it solves.

Commit messages should be concise and use the imperative mood: *"Add JSON output mode"* not *"Added JSON output mode"*.

Repository: [github.com/alexvoste/ForgeZero](https://github.com/alexvoste/ForgeZero)

---

## 18. License

ForgeZero is released under the **MIT License**.

```
MIT License

Copyright (c) 2026 AlexVoste

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
```

---

*If ForgeZero saves you time, consider giving the repository a ⭐️ on GitHub — it helps the project grow.*
