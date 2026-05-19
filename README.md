# ☘️ ForgeZero (fz) — Complete Documentation

<div align="center">
  <table style="border:none; background:transparent;">
    <tr>
      <td style="vertical-align:middle; padding-right:32px; border:none;">
        <img src="pictures/fz.jpg" alt="ForgeZero Logo" width="180" />
      </td>
      <td style="vertical-align:middle; border:none;">
        <h3 style="margin:0 0 8px 0;">ForgeZero — zero-overhead build tool for assembly & C</h3>
        <p style="margin:0; color:#555;">One command. Any assembler. Any platform.</p>
        <br/>
        <img src="https://github.com/alexvoste/ForgeZero/actions/workflows/go.yml/badge.svg" alt="Build Status"/>
        &nbsp;
        <img src="https://img.shields.io/github/go-mod/go-version/alexvoste/ForgeZero" alt="Go Version"/>
        &nbsp;
        <img src="https://img.shields.io/github/license/alexvoste/ForgeZero" alt="License"/>
        &nbsp;
        <img src="https://img.shields.io/github/commits-since/alexvoste/ForgeZero/v1.5.0" alt="Commits"/>
      </td>
    </tr>
  </table>
</div>

> **Version:** 2.0.0 NEXUS &nbsp;·&nbsp; **Language:** Go &nbsp;·&nbsp; **License:** MIT &nbsp;·&nbsp; **Platform:** Linux · Windows · macOS

ForgeZero is a high-performance, zero-overhead build tool for assembly and C developers. It wraps NASM, GAS, FASM, GCC, Clang, and LD into a single unified command-line interface — no Makefiles, no build scripts, no configuration required to get started.

> Inspired by the simplicity of **Suckless** and the efficiency of **TinyCC**

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
10. [C++ Compilation](#10-c-compilation-1)
11. [Cross-Compilation](#11-cross-compilation)
12. [Static Library Mode](#12-static-library-mode)
13. [Shared Library Mode](#13-shared-library-mode)
14. [Package Manager (fz pm)](#14-package-manager-fz-pm)
15. [Internal Mechanisms](#15-internal-mechanisms)
    - 15.1 [Build Cache (BLAKE3)](#151-build-cache-blake3)
    - 15.2 [Pre-link Symbol Check](#152-pre-link-symbol-check)
    - 15.3 [Watch Mode](#153-watch-mode)
    - 15.4 [JSON Output](#154-json-output)
    - 15.5 [Clean](#155-clean)
    - 15.6 [Parallel Builds](#156-parallel-builds)
    - 15.7 [Interactive Shell](#157-interactive-shell)
16. [Configuration File Reference](#16-configuration-file-reference)
    - 16.1 [Basic Fields](#161-basic-fields)
    - 16.2 [Multiple Source Directories](#162-multiple-source-directories)
    - 16.3 [Explicit Source File Lists](#163-explicit-source-file-lists)
    - 16.4 [Include & Exclude Patterns](#164-include--exclude-patterns)
    - 16.5 [Library Linking](#165-library-linking)
    - 16.6 [Custom Compiler & Linker Flags](#166-custom-compiler--linker-flags)
    - 16.7 [.fzignore File](#167-fzignore-file)
    - 16.8 [Full Annotated Example](#168-full-annotated-example)
17. [Assembler Backends](#17-assembler-backends)
    - 17.1 [NASM (.asm)](#171-nasm-asm)
    - 17.2 [GAS (.s / .S)](#172-gas-s--s)
    - 17.3 [FASM (.fasm)](#173-fasm-fasm)
18. [Project Initialization](#18-project-initialization)
19. [LSP & IDE Integration](#19-lsp--ide-integration)
20. [Self-Update](#20-self-update)
21. [Examples](#21-examples)
22. [Exit Codes](#22-exit-codes)
23. [Troubleshooting](#23-troubleshooting)
24. [Roadmap](#24-roadmap)
25. [Contributing](#25-contributing)
26. [License](#26-license)

---

## 1. Overview

ForgeZero removes the friction between writing assembly (or C) code and running it. Instead of managing assembler flags, linker invocations, and object file paths by hand, you point `fz` at a source file or directory and it handles everything:

- Detects the file type and selects the correct assembler backend automatically.
- Compiles each source file into an object file with appropriate flags.
- Checks for duplicate global symbols across all objects before linking.
- Links everything into a single binary using the most appropriate linker.
- Caches compiled objects using **BLAKE3** so unchanged files are never recompiled.
- Optionally watches the filesystem and rebuilds on every save.
- Emits structured JSON build reports for CI/CD integration.
- Generates `compile_commands.json` for full LSP and IDE integration.
- Supports cross-compilation to ARM, RISC-V, and other targets via `-target`.
- Builds static libraries (`.a`) and shared libraries (`.so` / `.dylib`) in addition to executables.
- Compiles C++ (`.cpp`, `.cc`, `.cxx`) with the same strict standards as C.
- Manages external C/ASM dependencies via the built-in package manager (`fz pm`).

**What's new in v2.0.0 NEXUS:**

- **BLAKE3 hashing** — cache is up to 7× faster. File hash time for 10 MB dropped from ~58 ms to ~8.7 ms. `internal/utils/hash.go` uses `github.com/zeebo/blake3` with a fast parallel implementation.
- **Package manager (`fz pm`)** — manage external C/ASM dependencies from Git or the official catalog. Supports `add`, `remove`, `list`, `update`, `catalog`, `search`, and `install` with BLAKE3 hash verification.
- **Shared library support** — `-shared`, `-cc-flag`, and `-ld-flag` flags for building `.so`, `.dylib`, and `.dll` targets.
- **High test coverage** — utils 84%, linker 60%, assembler 60%, builder 56%.
- **All `golangci-lint` warnings fixed** — `errcheck`, `govet`, `ineffassign`, and others.
- **Context and timeouts for all network/git operations** — no more hanging on slow connections.

**What's new in v1.9.0:**

- **Cross-compilation** — `-target <triple>` supports ARM, RISC-V, x86_64, and any standard GNU cross-compilation triple. `fz` auto-detects the correct prefixed compilers and linkers.
- **LSP support** — `-compile-commands` generates `compile_commands.json` for clangd, ccls, and any LSP-aware editor (Neovim, VSCode, CLion, etc.).
- **Smart self-update** — `fz -update` now creates a backup of the old binary at `/usr/local/bin/fz.old` before installing the new one.
- **Improved test coverage** — linker coverage raised to 60%+ (from 17%); all packages above 40%.
- **Pluggable `CheckTool`** — internal tool-presence checks are now injectable for testing toolchain-absent scenarios.
- **Shell builds single files** — the interactive shell (`fz -shell`) can now compile and run individual source files.
- **Object file collision fix** — multi-directory projects no longer produce colliding `.o` names; each object is uniquely named from its full source path.

**What's new in v1.8.0:**

- **Static libraries** — `-type static` and `-lib` build `.a` archives via `ar` instead of producing a linked executable.
- **Unique object file names** — directory builds with files of the same base name in different subdirectories now compile without conflicts.
- **Builder stability** — fixed test reliability issues and removed `..` path components from all generated object file names.

**What's new in v1.7.0:**

- **Parallel builds** — `-j N` compiles all source files concurrently (0 = auto = number of CPU cores).
- **Linker scripts and text address** — `-T <script>` and `-Ttext <addr>` pass linker scripts and entry addresses directly to `ld`.
- **Interactive shell** — `fz -shell` opens a REPL for running `fz` commands without re-invoking the binary.
- **Output formats** — `-format elf32`, `-format elf64`, `-format bin` explicitly control the output format (default `elf64`).
- **C++ support** — `.cpp`, `.cc`, and `.cxx` files are compiled with `g++` or `clang++`, subject to the same strict warning flags.

**What's new in v1.6.0:**

- **Project initialization** — `fz -init` scaffolds a new project: creates `.fz.yaml`, `.fzignore`, and `README.md` in the current directory.
- **Flat binary output** — `-format bin` produces raw flat binaries for bootloaders, firmware, and embedded targets.
- **Library linking** — the `libs` field in config adds `-l<lib>` flags to the linker without manual `flags.ld` entries.
- **Custom flags** — `flags.cc`, `flags.asm`, and `flags.ld` in `.fz.yaml` pass arbitrary extra arguments to each tool.
- **`utils.CopyFile`** — internal utility for safe file duplication used by the update and static library subsystems.

**What's new in v1.5.0:**

- **Multiple source directories** — `source_dirs` accepts a list of directories scanned in parallel.
- **Explicit source file lists** — `source_files` lets you enumerate exactly which files to build, bypassing directory scanning entirely.
- **`include` patterns** — counterpart to `exclude`; only files matching at least one pattern are considered.
- **Library linking** — `libs` config field adds `-l<lib>` flags to the linker without manual `flags.ld` entries.
- **Per-tool custom flags** — `flags.asm`, `flags.cc`, and `flags.ld` in config pass arbitrary arguments to each tool.
- **`.fzignore` file** — a `.gitignore`-style file for fine-grained exclusion rules during recursive scanning.
- **Multi-level config merging** — system-level, user-level, and project-level YAML configs are merged in order.

ForgeZero is intentionally lightweight — a single statically compiled Go binary with no runtime dependencies beyond the standard assembler/compiler toolchain.

---

## 2. Requirements

### Assembler and compiler tools

| Source type        | Required tool          | Notes |
|--------------------|------------------------|-------|
| `.asm`             | `nasm`                 | x86/x86-64 Intel syntax |
| `.s` / `.S`        | `gcc` (drives `as`)    | AT&T syntax; `.S` files are C-preprocessed first |
| `.fasm`            | `fasm`                 | Must be downloaded separately from flatassembler.net |
| `.c`               | `gcc` or `clang`       | Strict flags + sanitizers by default |
| `.cpp` / `.cc` / `.cxx` | `g++` or `clang++` | Same strict flags as C; `clang++` preferred in strict mode |

### Linker tools

| Linker  | Required for |
|---------|--------------|
| `gcc`   | Default linking, C runtime support |
| `ld`    | Raw linking (`-mode raw`), linker scripts |
| `clang` | Strict sanitizer mode (`-strict`) |
| `ar`    | Static library mode (`-type static`) |

### Cross-compilation tools (optional)

When using `-target <triple>`, `fz` looks for prefixed toolchain binaries on your `PATH`. For example:

| Target triple            | Expected compiler prefix     |
|--------------------------|------------------------------|
| `arm-linux-gnueabihf`    | `arm-linux-gnueabihf-gcc`    |
| `aarch64-linux-gnu`      | `aarch64-linux-gnu-gcc`      |
| `riscv64-linux-gnu`      | `riscv64-linux-gnu-gcc`      |
| `x86_64-linux-gnu`       | `x86_64-linux-gnu-gcc`       |

Install cross-compilers via your package manager (e.g. `sudo apt install gcc-arm-linux-gnueabihf`).

### Optional tools (used internally)

| Tool       | Purpose |
|------------|---------|
| `nm`       | Pre-link duplicate symbol check (primary) |
| `objdump`  | Fallback for symbol check |
| `readelf`  | Second fallback for symbol check |
| `git`      | Required for `fz pm add` (package manager) |

### Go version (build from source only)

Go **1.21** or later is required to build `fz` from source.

---

## 3. Installation

### 3.1 Linux — Debian / Ubuntu

**Install system dependencies:**

```bash
sudo apt update
sudo apt install -y nasm gcc binutils git
```

**Install Clang (optional, for `-strict` mode):**

```bash
sudo apt install -y clang
```

**Install cross-compilation toolchain (optional):**

```bash
sudo apt install -y gcc-arm-linux-gnueabihf
sudo apt install -y gcc-aarch64-linux-gnu
sudo apt install -y gcc-riscv64-linux-gnu
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
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
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
sudo dnf install -y nasm gcc binutils clang git

# RHEL / CentOS — enable EPEL first for nasm
sudo dnf install -y epel-release
sudo dnf install -y nasm gcc binutils clang git
```

**Install ForgeZero:**

```bash
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
```

---

### 3.3 Linux — Arch Linux / Manjaro

**Install system dependencies:**

```bash
sudo pacman -S --noconfirm nasm gcc binutils clang git

# FASM is available in the AUR
yay -S fasm
```

**Install ForgeZero:**

```bash
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
```

---

### 3.4 Linux — openSUSE

**Install system dependencies:**

```bash
sudo zypper install -y nasm gcc binutils clang git
```

**Install ForgeZero:**

```bash
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
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
brew install nasm gcc go git
```

> **Note:** macOS ships `clang` as the system compiler under the `gcc` alias via Xcode Command Line Tools. For full GCC, `brew install gcc` installs it as `gcc-14` (or similar). ForgeZero uses whatever `gcc` resolves to on your `PATH`.

**Install ForgeZero:**

```bash
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
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
pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-binutils mingw-w64-x86_64-clang git
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
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
```

Or build from source:

```powershell
git clone https://github.com/forgezero-cli/ForgeZero.git
cd ForgeZero
go build -o fz.exe ./cmd/fz/main.go
```

Move `fz.exe` to a directory on your `PATH`.

> **Known limitation:** `-sanitize` and `-strict` require Clang with AddressSanitizer support compiled for Windows. This is available via the LLVM official Windows release but requires additional setup beyond MSYS2. Basic NASM assembly and GCC linking work without any extra configuration.

---

### 3.7 Build from Source (All Platforms)

```bash
git clone https://github.com/forgezero-cli/ForgeZero.git
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
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
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

**Compile a C++ file:**

```bash
fz -cc main.cpp
./main
```

**Build an entire directory:**

```bash
fz -dir ./src
./src
```

**Initialize a new project:**

```bash
fz -init
```

**Build with cross-compilation:**

```bash
fz -cc main.c -target arm-linux-gnueabihf
```

**Generate LSP compilation database:**

```bash
fz -compile-commands
```

**Build a static library:**

```bash
fz -dir ./src -type static -lib mylib
```

**Build a shared library:**

```bash
fz -cc mylib.c -shared -o libmylib.so
```

**Add a package dependency:**

```bash
fz pm add github.com/me/my-lib
```

**Build multiple directories (v1.5.0):**

```yaml
# .fz.yaml
source_dirs:
  - kernel
  - libc
  - drivers
output: myos
```

```bash
fz
```

---

## 5. Supported Languages & Extensions

| Extension          | Language   | Backend          | Notes |
|--------------------|------------|------------------|-------|
| `.asm`             | Assembly   | NASM             | x86/x86-64, Intel syntax, ELF64 |
| `.s`               | Assembly   | GAS via `gcc -c` | AT&T syntax |
| `.S`               | Assembly   | GAS via `gcc -c` | AT&T syntax + C preprocessor |
| `.fasm`            | Assembly   | FASM             | Requires separate install |
| `.c`               | C          | GCC or Clang     | Strict flags + sanitizers by default |
| `.cpp` / `.cc` / `.cxx` | C++   | G++ or Clang++   | Same strict flags as C (v1.7.0+) |

All other file extensions are silently ignored during directory and recursive scanning.

---

## 6. Build Modes

### 6.1 Single File Mode

Compiles and links a single source file into a binary.

```bash
fz -asm program.asm
fz -cc main.c
fz -cc main.cpp
```

- Output binary name is derived from the source filename (`program.asm` → `program`).
- A single object file is created and removed after linking unless `-keep-obj` is set.
- Override the binary name with `-out` and the object file name with `-out-obj`.

### 6.2 Directory Mode

Recursively scans a directory for all supported source files, compiles each to a uniquely named object file, then links everything into a single binary.

```bash
fz -dir ./src
```

**Object file naming** — names are generated to prevent collisions across subdirectories. Each object file name is derived from the full relative path of its source file, ensuring uniqueness even when files share the same base name:

| Source file          | Object file          |
|----------------------|----------------------|
| `src/hello.asm`      | `src_hello_asm.o`    |
| `src/hello.s`        | `src_hello_s.o`      |
| `src/sub/hello.asm`  | `src_sub_hello_asm.o` |
| `lib/hello.asm`      | `lib_hello_asm.o`    |

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

**Config merging (v1.5.0):** ForgeZero now supports multi-level config files merged in priority order:

1. System-level config: `/etc/fz/fz.yaml`
2. User-level config: `~/.config/fz/fz.yaml`
3. Project-level config: `.fz.yaml` in the working directory
4. CLI flags (highest priority, always override everything)

Each level overrides values from the previous one. This lets you set organization-wide defaults at the system level, personal preferences at the user level, and project-specific overrides in the project directory.

---

## 7. CLI Reference

### Synopsis

```
fz [options]
```

At least one of `-asm`, `-cc`, `-dir`, `-init`, `-shell`, `pm`, or a valid config file must be present.

### Full Flag Reference

| Flag | Argument | Default | Description |
|------|----------|---------|-------------|
| `-asm` | `<file>` | — | Assemble the given assembly source file. |
| `-cc` | `<file>` | — | Compile the given C or C++ source file. |
| `-dir` | `<dir>` | — | Recursively build all supported files in the directory. |
| `-out` | `<name>` | Derived from source | Name of the output binary. |
| `-out-obj` | `<name>` | `<basename>.o` | Object file name (single-file mode only). |
| `-mode` | `auto\|c\|raw` | `auto` | Linking mode. See [Linking Modes](#8-linking-modes). |
| `-format` | `elf32\|elf64\|bin` | `elf64` | Output format for assembled binaries. |
| `-target` | `<triple>` | — | Cross-compilation target triple (e.g. `arm-linux-gnueabihf`). |
| `-type` | `executable\|static` | `executable` | Output type: linked binary or static library (`.a`). |
| `-lib` | `<name>` | — | Output library name when `-type static` is used (without `lib` prefix or `.a` suffix). |
| `-shared` | — | off | Build a shared library (`.so` / `.dylib` / `.dll`). |
| `-cc-flag` | `<flags>` | — | Extra compiler flags, space-separated, injected after standard flags. |
| `-ld-flag` | `<flags>` | — | Extra linker flags, space-separated, appended to the linker command. |
| `-j` | `<N>` | `1` | Parallel compilation jobs. `0` = auto (number of CPU cores). |
| `-T` | `<script>` | — | Linker script to pass to `ld`. |
| `-Ttext` | `<addr>` | — | Entry point address to pass to the linker (hex or decimal). |
| `-debug` | — | off | Pass `-g` to the assembler/compiler to emit debug symbols. |
| `-verbose` | — | off | Print each external command to stdout before running it. |
| `-keep-obj` | — | off | Preserve object files after linking (directory mode). |
| `-no-cache` | — | off | Disable the build cache; always recompile every file. |
| `-no-symbol-check` | — | off | Skip the pre-link duplicate symbol check. |
| `-sanitize` | — | **on** | Enable `-fsanitize=address,undefined` for C/C++. Disable with `-sanitize=false`. |
| `-strict` | — | off | Stricter sanitizers + use-after-return/scope checks. Prefers `clang`/`clang++`. |
| `-json` | — | off | Suppress normal output; emit a JSON build report to stdout. |
| `-watch` | — | off | Watch source files for changes and rebuild automatically. |
| `-clean` | — | off | Remove all build artifacts and exit. |
| `-compile-commands` | — | off | Generate `compile_commands.json` for LSP/IDE integration. |
| `-init` | — | off | Scaffold a new project: creates `.fz.yaml`, `.fzignore`, and `README.md`. |
| `-shell` | — | off | Open interactive REPL shell. |
| `-update` | — | off | Download and install the latest `fz` binary; backs up current binary to `fz.old`. |
| `-config` | `<file>` | auto-detect | Path to a YAML configuration file. |
| `-timeout` | `<sec>` | `60` | Timeout in seconds for each sub-command. |
| `-h`, `--help` | — | — | Print help and exit. |
| `-v`, `--version` | — | — | Print version and exit. |

### Package Manager Sub-commands

| Command | Description |
|---------|-------------|
| `fz pm add <repo>[@version]` | Clone a package from a Git repository and register it in `.fz.yaml`. |
| `fz pm remove <package>` | Remove a package and clean up `.fz.yaml` and empty parent directories. |
| `fz pm list` | List all installed packages. |
| `fz pm update` | Update all installed packages to the latest commit or tag. |
| `fz pm catalog` | Browse the official ForgeZero package catalog. |
| `fz pm search <query>` | Search the catalog by name or keyword. |
| `fz pm install <name>` | Install a package from the catalog with BLAKE3 hash verification. |

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

## 10. C++ Compilation

Added in **v1.7.0**. ForgeZero compiles `.cpp`, `.cc`, and `.cxx` files using `g++` or `clang++`. The same strict warning flags applied to C are applied identically to C++:

```
-Wall -Wextra -Werror -Wpedantic -Wshadow -Wconversion
```

Sanitizers are also enabled by default for C++ in the same way as C.

`clang++` is preferred when `-strict` is active. If `clang++` is not available, `g++` is used with the supported subset of sanitizer flags.

**Single C++ file:**

```bash
fz -cc main.cpp
fz -cc main.cc
fz -cc main.cxx
```

**Mixed C and C++ project directory:**

```bash
fz -dir ./src
```

`fz` dispatches `.c` files to `gcc`/`clang` and `.cpp`/`.cc`/`.cxx` files to `g++`/`clang++` automatically. All objects are linked together in a single step.

**Disable sanitizers for release:**

```bash
fz -cc main.cpp -sanitize=false
```

---

## 11. Cross-Compilation

Added in **v1.9.0**. The `-target` flag enables cross-compilation to any architecture supported by the GNU toolchain installed on your system.

### Basic Usage

```bash
fz -cc main.c -target arm-linux-gnueabihf
fz -cc main.c -target aarch64-linux-gnu
fz -cc main.c -target riscv64-linux-gnu
fz -dir ./src -target arm-linux-gnueabihf -out firmware
```

### How It Works

When `-target <triple>` is set, `fz` constructs the expected prefixed compiler and linker names by prepending the triple to the tool name:

- Compiler: `<triple>-gcc` (e.g. `arm-linux-gnueabihf-gcc`)
- C++ compiler: `<triple>-g++`
- Linker: `<triple>-gcc` or `<triple>-ld` depending on the linking mode
- Archiver: `<triple>-ar` (when `-type static`)

`fz` verifies that the prefixed compiler is available on `PATH` before starting the build. If the cross-compiler is not found, the build exits with code `2` and a clear error message naming the missing tool.

### Installing Cross-Compilers

**Debian / Ubuntu:**

```bash
sudo apt install gcc-arm-linux-gnueabihf         # ARMv7 hard-float
sudo apt install gcc-aarch64-linux-gnu           # ARM64
sudo apt install gcc-riscv64-linux-gnu           # RISC-V 64-bit
```

**Fedora:**

```bash
sudo dnf install gcc-arm-linux-gnu
sudo dnf install gcc-aarch64-linux-gnu
```

**Arch Linux:**

```bash
sudo pacman -S arm-linux-gnueabihf-gcc
sudo pacman -S aarch64-linux-gnu-gcc
```

### Cross-Compilation with a Config File

```yaml
# .fz.yaml
source_dirs:
  - src
output: firmware.elf
target: arm-linux-gnueabihf
mode: raw
flags:
  cc:
    - -mcpu=cortex-m4
    - -mfpu=fpv4-sp-d16
    - -mfloat-abi=hard
    - -ffreestanding
  ld:
    - -T
    - linker.ld
```

```bash
fz
```

### Notes

- All standard `fz` features work with cross-compilation: build cache, parallel builds, sanitizer flags (if the cross-compiler supports them), static libraries, and JSON output.
- Sanitizers may not be available for all cross-compilation targets. If the cross-compiler reports an unsupported sanitizer flag, use `-sanitize=false`.
- `-strict` mode selects `<triple>-clang` if available; otherwise falls back to `<triple>-gcc`.

---

## 12. Static Library Mode

Added in **v1.8.0**. ForgeZero can produce static libraries (`.a` archives) instead of linked executables.

### Basic Usage

```bash
fz -dir ./src -type static -lib mylib
```

This compiles all source files in `./src/` into object files, then archives them into `libmylib.a` using `ar`.

### Flags

| Flag | Description |
|------|-------------|
| `-type static` | Build a static library instead of an executable |
| `-lib <name>` | Name of the library (without `lib` prefix and `.a` suffix) |

The output file is always named `lib<name>.a`. For example, `-lib mylib` produces `libmylib.a`.

### Using a Config File

```yaml
# .fz.yaml
source_dirs:
  - src
type: static
lib: mylib
```

```bash
fz
```

### Linking Against the Produced Library

```yaml
# dependent project .fz.yaml
source_files:
  - main.c
libs:
  - mylib
flags:
  ld:
    - -L./path/to/libmylib
```

### Notes

- `-type static` is incompatible with `-mode raw`. Use `-mode c` or `-mode auto` when producing a static library.
- Cross-compilation works: `fz -dir ./src -type static -lib mylib -target arm-linux-gnueabihf` uses `arm-linux-gnueabihf-ar`.

---

## 13. Shared Library Mode

Added in **v2.0.0 NEXUS**. ForgeZero can produce shared libraries (`.so`, `.dylib`, `.dll`) using the `-shared` flag.

### Basic Usage

```bash
# Build a shared library from a C file
fz -cc mylib.c -shared -o libmylib.so

# With extra compiler and linker flags
fz -cc mylib.c -shared -cc-flag "-O2 -fPIC" -ld-flag "-pthread" -o libmylib.so
```

### Flags

| Flag | Description |
|------|-------------|
| `-shared` | Build a shared library instead of an executable |
| `-cc-flag "<flags>"` | Extra compiler flags, space-separated, injected after standard flags |
| `-ld-flag "<flags>"` | Extra linker flags, space-separated, appended to the linker command |

### How It Works

When `-shared` is set, `fz` adds `-shared` to the linker command. The `-cc-flag` and `-ld-flag` values are split by spaces and injected into the compiler and linker argument lists respectively. The same flags work for C++ files (`.cpp`, `.cc`, `.cxx`).

### Notes

- When building shared libraries for Linux, include `-fPIC` in `-cc-flag` to produce position-independent code.
- `-shared` can be combined with `-target` for cross-compiled shared libraries.
- Sanitizer flags are applied to object compilation as normal.

---

## 14. Package Manager (fz pm)

Added in **v2.0.0 NEXUS**. `fz pm` is a built-in package manager for C/ASM projects. It manages external dependencies from Git repositories or from the official ForgeZero package catalog.

### Sub-commands

#### fz pm add

```bash
# Install a library from GitHub
fz pm add github.com/me/my-lib

# Install a specific version (tag or commit)
fz pm add github.com/me/my-lib@v1.2.3
```

`add` clones the repository into `vendor/`, runs `git checkout` if a version tag is given, and automatically updates `.fz.yaml` (`source_dirs`). All subsequent `fz` builds include the vendored package.

#### fz pm remove

```bash
fz pm remove my-lib
```

Deletes the package directory from `vendor/`, cleans up `.fz.yaml`, and removes any empty parent directories left behind.

#### fz pm list

```bash
fz pm list
```

Lists all currently installed packages and their versions as recorded in `.fz.yaml`.

#### fz pm update

```bash
fz pm update
```

Pulls the latest changes for all installed packages. If a package was installed at a specific tag, it is updated to the latest commit on the default branch unless a version constraint is specified in `.fz.yaml`.

#### fz pm catalog

```bash
fz pm catalog
```

Fetches and displays the full list of packages available in the official ForgeZero catalog (`https://raw.githubusercontent.com/forgezero-cli/catalog/main/catalog.json`).

#### fz pm search

```bash
fz pm search iot
fz pm search crypto
```

Searches the official catalog by name or keyword and prints matching packages with their descriptions.

#### fz pm install

```bash
fz pm install esp-idf
```

Installs a named package from the official catalog. Unlike `fz pm add`, `install` additionally verifies the package content against a BLAKE3 hash stored in the catalog manifest to ensure integrity.

### How It Works

- Packages are cloned into `vendor/<package-name>/`.
- `.fz.yaml` is updated automatically to include `vendor/<package-name>` in `source_dirs`.
- All network and git operations run with context and configurable timeouts (default: 60 s, override with `-timeout`).
- The catalog is a community-driven JSON file hosted at `https://github.com/forgezero-cli/catalog`.

### Contributing Packages to the Catalog

To add your library to the official catalog, open a pull request at [github.com/forgezero-cli/catalog](https://github.com/forgezero-cli/catalog) and add an entry to `catalog.json` with your repository URL, description, and BLAKE3 hash.

---

## 15. Internal Mechanisms

### 15.1 Build Cache (BLAKE3)

ForgeZero caches compiled object files in `.fz_cache/` to skip recompilation of unchanged sources.

**Cache key** is computed from:

- **BLAKE3** hash of the source file contents (replaces SHA256 as of v2.0.0 — up to 7× faster)
- `-debug` flag state (`true`/`false`)
- `-mode` value (`auto`, `c`, or `raw`)
- `-target` value (empty string for native builds)

If all four match an existing cache entry, the stored object file is reused. The assembler/compiler is never invoked for that file.

**BLAKE3 performance:**

| File size | SHA256 (pre-2.0.0) | BLAKE3 (2.0.0+) |
|-----------|-------------------|-----------------|
| 10 MB     | ~58 ms            | ~8.7 ms         |

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

### 15.2 Pre-link Symbol Check

Before invoking the linker, `fz` scans all compiled object files for duplicate global symbol definitions. This catches conflicts — such as `_start` or a global function defined in two files — before the linker emits a cryptic error.

**Tools used (in order of preference):** `nm` → `objdump` → `readelf`

If a conflict is found, `fz` reports which files define the duplicate symbol and exits with code `1`, without attempting to link.

**Disable the check:**

```bash
fz -dir ./src -no-symbol-check
```

Use this when intentionally relying on weak symbols or linker scripts that resolve conflicts at link time.

---

### 15.3 Watch Mode

Watch mode monitors source files (and the config file, if present) for filesystem changes and triggers a rebuild automatically.

```bash
fz -dir ./src -watch
fz -asm main.asm -watch
```

Uses [fsnotify](https://github.com/fsnotify/fsnotify) for cross-platform filesystem event detection. Rebuilds are debounced with a **500 ms** delay — multiple rapid saves within that window produce only one rebuild.

Press `Ctrl+C` to exit.

---

### 15.4 JSON Output

When `-json` is passed, all standard output is suppressed. On completion (or on error), a single JSON object is written to stdout:

```json
{
  "status": "success",
  "exit_code": 0,
  "duration_ms": 342,
  "binary": "./src",
  "source_files": ["src/main.asm", "src/utils.asm"],
  "object_files": ["src_main_asm.o", "src_utils_asm.o"],
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

### 15.5 Clean

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

### 15.6 Parallel Builds

Added in **v1.7.0**. The `-j` flag controls how many source files are compiled concurrently.

```bash
fz -dir ./src -j 4      # compile up to 4 files simultaneously
fz -dir ./src -j 0      # auto: use all available CPU cores
```

By default, `fz` compiles files sequentially (`-j 1`). On large projects, parallel builds significantly reduce total build time.

When `-j 0` is specified, `fz` queries the system for the number of logical CPU cores and uses that value. On a 16-core machine, this is equivalent to `-j 16`.

Parallel compilation does not affect the linking step — all objects must be compiled before linking begins. Build cache hits are served from disk without spawning a compiler process, so cached files do not consume a worker slot.

---

### 15.7 Interactive Shell

Added in **v1.7.0**, extended in **v1.9.0**.

```bash
fz -shell
```

Opens a REPL where you can issue `fz` commands interactively without re-invoking the binary. Useful for rapid iteration during development.

**Supported shell commands:**

| Command | Description |
|---------|-------------|
| `build <file>` | Compile and link a single source file |
| `build -dir <dir>` | Build all files in a directory |
| `set <flag> <value>` | Set a build flag for subsequent commands (e.g. `set mode raw`) |
| `clean` | Remove build artifacts |
| `help` | List available commands |
| `exit` | Exit the shell |

**Example session:**

```
fz> build main.c
[fz] Compiling main.c...
[fz] Linking...
[fz] Done: ./main (1 file, 214ms)

fz> set mode raw
[fz] mode = raw

fz> build boot.asm
[fz] Assembling boot.asm...
[fz] Linking (raw)...
[fz] Done: ./boot (1 file, 89ms)

fz> exit
```

---

## 16. Configuration File Reference

ForgeZero accepts YAML configuration files. The file is searched automatically in this order: `.fz.yaml`, `fz.yaml`, `.fz.yml`, `fz.yml`. Use `-config <path>` to specify explicitly.

CLI flags always override config file values.

### 16.1 Basic Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `source_dir` | string | — | Single source directory (kept for backward compatibility) |
| `source_dirs` | `[]string` | — | Multiple source directories, each scanned recursively |
| `source_files` | `[]string` | — | Exact list of files to build; if set, `source_dirs` is ignored |
| `output` | string | auto | Output binary name |
| `mode` | string | `auto` | Linking mode: `auto`, `c`, or `raw` |
| `format` | string | `elf64` | Output format: `elf32`, `elf64`, or `bin` |
| `target` | string | — | Cross-compilation target triple |
| `type` | string | `executable` | Output type: `executable` or `static` |
| `lib` | string | — | Library name for `-type static` (without `lib` prefix / `.a` suffix) |
| `jobs` | int | `1` | Parallel compilation jobs (`0` = auto) |
| `debug` | bool | `false` | Emit debug symbols (`-g`) |
| `verbose` | bool | `false` | Print all invoked commands |
| `keep_obj` | bool | `false` | Preserve object files after linking |
| `no_cache` | bool | `false` | Disable build cache |
| `sanitize` | bool | `true` | Enable ASan + UBSan for C/C++ |
| `strict` | bool | `false` | Strict sanitizer mode, prefers `clang`/`clang++` |
| `ignore_file` | string | `.fzignore` | Path to a `.gitignore`-style exclusion file |

---

### 16.2 Multiple Source Directories

The `source_dirs` field (new in v1.5.0) lets you build from multiple directories in a single `fz` invocation. All directories are scanned recursively and their files are compiled together into one binary.

```yaml
source_dirs:
  - kernel
  - libc
  - drivers
output: forgeos.elf
mode: raw
```

Object file names are prefixed with their parent directory to avoid collisions:

| Source file         | Object file          |
|---------------------|----------------------|
| `kernel/boot.asm`   | `kernel_boot_asm.o`  |
| `libc/string.c`     | `libc_string_c.o`    |
| `drivers/uart.c`    | `drivers_uart_c.o`   |

---

### 16.3 Explicit Source File Lists

When `source_files` is set, `fz` builds exactly and only those files. Directory scanning is skipped entirely.

```yaml
source_files:
  - boot/start.asm
  - kernel/main.c
  - kernel/irq.c
output: kernel.elf
mode: raw
```

Each path is verified to exist at startup. If a file is missing, `fz` exits with code `2` before compiling anything.

`source_files` takes precedence over `source_dirs` and `source_dir` — if all three are set, only `source_files` is used.

---

### 16.4 Include & Exclude Patterns

**`exclude`** — glob patterns; any file or directory matching at least one pattern is skipped:

```yaml
exclude:
  - "test_*"
  - "*/legacy/"
  - "*.tmp"
```

**`include`** (new in v1.5.0) — glob patterns; only files matching at least one pattern are considered:

```yaml
include:
  - "*.asm"
  - "*.c"
```

**Evaluation order during recursive scanning:**

1. Check `exclude` patterns — skip if matched.
2. Check `.fzignore` file — skip if matched.
3. Check `include` patterns — skip if none match (when `include` is set).
4. Check supported extensions — skip all others.

---

### 16.5 Library Linking

The `libs` field specifies system libraries to link against. Each entry is passed to the linker as `-l<lib>`:

```yaml
libs:
  - m         # -lm  (math)
  - pthread   # -lpthread
  - c         # -lc
```

To add non-standard search paths, use `flags.ld: ["-L/path/to/libs"]`.

---

### 16.6 Custom Compiler & Linker Flags

```yaml
flags:
  asm:
    - -DDEBUG_BUILD
    - -I./include

  cc:
    - -O3
    - -march=native
    - -DNDEBUG
    - -ffreestanding

  ld:
    - -T
    - linker.ld
    - -Map
    - output.map
    - -z
    - max-page-size=0x1000
```

**Flag insertion points:**

| Tool | Standard flags | Your `flags.*` | Final flag |
|------|---------------|----------------|------------|
| NASM | `-felf64 <src> -o <obj>` | inserted before `-o` | — |
| GCC (asm) | `-c <src> -o <obj>` | inserted before `-c` | — |
| GCC (C/C++) | `-Wall -Wextra ... -c <src> -o <obj>` | inserted after warning flags | — |
| GCC/LD (link) | `<objects>` | inserted after objects | `-o <binary>` |

---

### 16.7 .fzignore File

The `.fzignore` file works exactly like `.gitignore`. It is loaded from the project root (or from the path set by `ignore_file` in config) and applied during all recursive directory scans.

**Example `.fzignore`:**

```
# Compiled objects
*.o
*.swp

# Directories to skip entirely
temp/
test_*/
vendor/

# Specific files
legacy/old_abi.asm
```

`.fzignore` is evaluated after `exclude` patterns. If a file is excluded by either, it is skipped.

---

### 16.8 Full Annotated Example

```yaml
# fz.yaml

source_dirs:
  - kernel
  - libc
  - drivers

output: forgeos.elf
format: elf64

# target: arm-linux-gnueabihf

# type: static
# lib: forgeos

mode: raw
debug: true
verbose: false
keep_obj: true
no_cache: false
jobs: 0

sanitize: true
strict: false

exclude:
  - "test_*"
  - "*/legacy/"
  - "*.tmp"

include:
  - "*.asm"
  - "*.c"
  - "*.cpp"
  - "*.s"

libs:
  - gcc
  - m

flags:
  asm:
    - -DDEBUG_BUILD
    - -I./include
  cc:
    - -O2
    - -march=native
    - -ffreestanding
  ld:
    - -T
    - linker.ld

ignore_file: .myfzignore
```

---

## 17. Assembler Backends

### 17.1 NASM (.asm)

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
fz -asm hello.asm
./hello
```

---

### 17.2 GAS (.s / .S)

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
    movq  $1,   %rax
    movq  $1,   %rdi
    movq  $msg, %rsi
    movq  $len, %rdx
    syscall

    movq  $60,  %rax
    xorq  %rdi, %rdi
    syscall
```

```bash
fz -asm hello.s
./hello
```

---

### 17.3 FASM (.fasm)

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

## 18. Project Initialization

Added in **v1.6.0**. The `-init` flag scaffolds a new ForgeZero project in the current directory.

```bash
mkdir myproject && cd myproject
fz -init
```

**Files created:**

| File | Contents |
|------|----------|
| `.fz.yaml` | Minimal project configuration with commented fields |
| `.fzignore` | Sensible default ignore rules |
| `README.md` | Project README template with `fz` build instructions |

If any of these files already exist, `fz -init` skips them and reports which files were created versus skipped. No existing file is overwritten.

**Generated `.fz.yaml`:**

```yaml
# .fz.yaml — ForgeZero project configuration
source_dir: src
output: myproject
mode: auto
sanitize: true
```

**Generated `.fzignore`:**

```
*.o
*.a
*.out
*.bin
*.elf
.fz_objs/
.fz_cache/
vendor/
*.swp
*.swo
*~
```

---

## 19. LSP & IDE Integration

Added in **v1.9.0**. The `-compile-commands` flag generates `compile_commands.json` in the project root.

```bash
fz -compile-commands
fz -dir ./src -compile-commands
```

**Editor setup:**

| Editor | Language server | Notes |
|--------|----------------|-------|
| Neovim | clangd | Install via Mason (`MasonInstall clangd`); clangd auto-detects `compile_commands.json` |
| VSCode | clangd extension | Install the `clangd` extension; point it to the project root |
| CLion | Built-in | Open the project root; CLion reads `compile_commands.json` automatically |
| Helix | clangd | Set `language-server = "clangd"` in `languages.toml` |
| Emacs | eglot / lsp-mode | Both read `compile_commands.json` from the project root |

**Combine with a regular build:**

```bash
fz -dir ./src -compile-commands
```

**Cross-compilation and LSP:**

```bash
fz -dir ./src -target arm-linux-gnueabihf -compile-commands
```

---

## 20. Self-Update

Added in **v1.9.0**.

```bash
fz -update
```

**What happens:**

1. `fz` fetches the latest release binary from the ForgeZero GitHub releases page.
2. The current binary is copied to `/usr/local/bin/fz.old`.
3. The new binary replaces the current one.
4. `fz` reports the version it upgraded from and to.

**Rolling back:**

```bash
sudo cp /usr/local/bin/fz.old /usr/local/bin/fz
```

Run with `sudo fz -update` if the binary is installed in a system directory.

---

## 21. Examples

### Minimal builds

```bash
fz -asm hello.asm
fz -asm hello.s
fz -asm hello.fasm
fz -cc main.c
fz -cc main.cpp
```

---

### Initialize a new project

```bash
mkdir myproject && cd myproject
fz -init
mkdir src
echo 'int main(void) { return 0; }' > src/main.c
fz
```

---

### Debug build with verbose output

```bash
fz -asm hello.asm -debug -verbose
gdb ./hello
```

---

### Bare-metal / bootloader binary

```bash
fz -asm boot.asm -mode raw -format bin -out boot.bin
```

---

### C with strict sanitizers

```bash
fz -cc main.c -strict
```

---

### Build a full project directory

```bash
fz -dir ./src
```

---

### Parallel build

```bash
fz -dir ./src -j 0
```

---

### Cross-compile for ARM

```bash
fz -cc main.c -target arm-linux-gnueabihf -sanitize=false
```

---

### Build a static library

```bash
fz -dir ./src -type static -lib mylib
ls libmylib.a
```

---

### Build a shared library

```bash
fz -cc mylib.c -shared -cc-flag "-O2 -fPIC" -o libmylib.so
```

---

### Package manager

```bash
# Add a dependency from GitHub
fz pm add github.com/me/my-lib

# Add a specific version
fz pm add github.com/me/my-lib@v1.2.3

# Install from the official catalog (with hash verification)
fz pm install esp-idf

# Search the catalog
fz pm search crypto

# List installed packages
fz pm list

# Update all packages
fz pm update

# Remove a package
fz pm remove my-lib
```

---

### Generate LSP compilation database

```bash
fz -dir ./src -compile-commands
cat compile_commands.json
```

---

### Build from multiple directories

```bash
# .fz.yaml: source_dirs: [src, lib], output: release
fz
```

---

### Link against system libraries

```yaml
# .fz.yaml
source_files: [calc.c]
libs: [m]
output: calc
```

```bash
fz
```

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

### Interactive shell session

```bash
fz -shell
# fz> build main.c
# fz> set mode raw
# fz> build boot.asm
# fz> exit
```

---

### Clean all build artifacts

```bash
fz -dir . -clean
```

---

### Update fz with rollback

```bash
sudo fz -update
# If something breaks:
sudo cp /usr/local/bin/fz.old /usr/local/bin/fz
```

---

## 22. Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success — binary was produced without errors. |
| `1` | Build error — assembler, compiler, or linker failed; or a duplicate global symbol was detected. Check stderr for details. |
| `2` | Argument error — invalid or missing flags, source file not found, cross-compiler not found on PATH, or unreadable configuration file. |

---

## 23. Troubleshooting

### `fz: command not found`

Ensure Go's binary directory is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

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

Download from [flatassembler.net](https://flatassembler.net):

```bash
wget https://flatassembler.net/fasm-1.73.32.tgz
tar -xzf fasm-1.73.32.tgz
sudo cp fasm/fasm /usr/local/bin/
chmod +x /usr/local/bin/fasm
```

---

### `g++: command not found`

```bash
sudo apt install g++           # Debian / Ubuntu
sudo dnf install gcc-c++       # Fedora
sudo pacman -S gcc             # Arch (g++ included)
brew install gcc               # macOS
```

---

### Cross-compiler not found

Install the appropriate cross-compilation toolchain:

```bash
sudo apt install gcc-arm-linux-gnueabihf     # Debian / Ubuntu
sudo dnf install gcc-arm-linux-gnu           # Fedora
sudo pacman -S arm-linux-gnueabihf-gcc       # Arch
```

---

### `undefined reference to _start`

Define `_start` explicitly:

```asm
; NASM
global _start
_start:
    ; ...
```

Or use the C runtime:

```bash
fz -asm program.asm -mode c
```

---

### Binary crashes immediately (segfault on startup)

Likely cause: `-mode raw` was used but the code references libc symbols. Switch to:

```bash
fz -asm program.asm -mode c
```

---

### Pre-link duplicate symbol error

Check your sources for conflicting `global` declarations. To skip the check when using weak symbols intentionally:

```bash
fz -dir ./src -no-symbol-check
```

---

### Sanitizer error at runtime

Fix the reported memory/UB issue, then rerun. To temporarily disable:

```bash
fz -cc main.c -sanitize=false
```

---

### Build hangs / times out

```bash
fz -asm big_program.asm -timeout 300
```

---

### Cache returns stale results

```bash
fz -dir . -clean
fz -dir ./src
```

Or for a one-off build:

```bash
fz -dir ./src -no-cache
```

---

### `fz pm add` fails / git not found

`fz pm` requires `git` on your `PATH`:

```bash
sudo apt install git     # Debian / Ubuntu
sudo dnf install git     # Fedora
sudo pacman -S git       # Arch
brew install git         # macOS
```

---

### `fz pm install` hash mismatch

The downloaded package content does not match the BLAKE3 hash in the catalog manifest. This may indicate a corrupted download or a tampered package. Do not use the package. Report the issue at [github.com/forgezero-cli/catalog](https://github.com/forgezero-cli/catalog).

---

### `source_files` path not found

```
fz: argument error: source file not found: kernel/main.c
```

Check that the path is relative to the directory where you run `fz`, not relative to the config file location.

---

### `libs` not found at link time

Add the directory containing the library via `flags.ld`:

```yaml
libs:
  - mylib
flags:
  ld:
    - -L/path/to/custom/libs
```

---

### `compile_commands.json` not picked up by editor

Ensure the file is in the **project root**. Regenerate after adding new source files:

```bash
fz -compile-commands
```

---

### `fz -update` fails with permission denied

```bash
sudo fz -update
```

---

### Watch mode does not detect changes on WSL2

Edit files from within the WSL2 terminal to get reliable events. This is a WSL2 kernel limitation with `inotify` when files are edited from Windows applications.

---

### Windows: `gcc` not found

Ensure MSYS2 `mingw64\bin` is in your Windows `PATH`:

```
C:\msys64\mingw64\bin
```

---

## 24. Roadmap

| Feature | Status |
|---------|--------|
| `exclude` patterns in config file | ✅ Done (v1.5.0) |
| `include` patterns in config file | ✅ Done (v1.5.0) |
| Multiple `source_dirs` | ✅ Done (v1.5.0) |
| Explicit `source_files` list | ✅ Done (v1.5.0) |
| `libs` field for library linking | ✅ Done (v1.5.0) |
| `flags.cc` for C compiler flags | ✅ Done (v1.5.0) |
| `.fzignore` file support | ✅ Done (v1.5.0) |
| Multi-level config merging | ✅ Done (v1.5.0) |
| `fz -init` project scaffolding | ✅ Done (v1.6.0) |
| `-format bin` flat binary output | ✅ Done (v1.6.0) |
| `utils.CopyFile` internal utility | ✅ Done (v1.6.0) |
| Parallel builds (`-j N`) | ✅ Done (v1.7.0) |
| Linker scripts (`-T`, `-Ttext`) | ✅ Done (v1.7.0) |
| Interactive shell (`fz -shell`) | ✅ Done (v1.7.0) |
| Output format selection (`elf32`, `elf64`, `bin`) | ✅ Done (v1.7.0) |
| C++ support (`.cpp`, `.cc`, `.cxx`) | ✅ Done (v1.7.0) |
| Static library mode (`-type static`) | ✅ Done (v1.8.0) |
| Unique object file names (path-based) | ✅ Done (v1.8.0) |
| Builder stability and `..` path fix | ✅ Done (v1.8.0) |
| Cross-compilation (`-target <triple>`) | ✅ Done (v1.9.0) |
| LSP integration (`-compile-commands`) | ✅ Done (v1.9.0) |
| Smart self-update with rollback (`fz -update`) | ✅ Done (v1.9.0) |
| Linker test coverage 60%+ | ✅ Done (v1.9.0) |
| Shell builds single files + shell tests | ✅ Done (v1.9.0) |
| BLAKE3 hashing (7× faster cache) | ✅ Done (v2.0.0) |
| Package manager (`fz pm`) | ✅ Done (v2.0.0) |
| Official package catalog | ✅ Done (v2.0.0) |
| Shared library support (`-shared`) | ✅ Done (v2.0.0) |
| `-cc-flag` / `-ld-flag` CLI pass-through flags | ✅ Done (v2.0.0) |
| High test coverage (utils 84%, linker 60%+) | ✅ Done (v2.0.0) |
| Colored terminal output (green success / red error) | Planned |
| GDB integration and improved debug workflow | Planned |
| Man page (`man fz`) | Planned |
| Windows native support without WSL2 | In progress |
| macOS full support and testing | In progress |

---

## 25. Contributing

Contributions are welcome: bug reports, feature requests, documentation improvements, and code patches.

1. **Open an issue** before starting significant work to align on the approach.
2. **Fork the repository** and create a descriptive feature branch (`feature/watch-debounce`, `fix/nasm-elf32`).
3. **Write tests** for new behavior and ensure existing tests pass:

   ```bash
   go test ./...
   ```

4. **Submit a Pull Request** with a clear description of the change and the problem it solves.

Commit messages should be concise and use the imperative mood: *"Add JSON output mode"* not *"Added JSON output mode"*.

Repository: [github.com/forgezero-cli/ForgeZero](https://github.com/forgezero-cli/ForgeZero)

---

## 26. License

ForgeZero is released under the **MIT License**.

```
MIT License

Copyright (c) AlexVoste

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
