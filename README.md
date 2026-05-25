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
        <img src="https://img.shields.io/github/go-mod/go-version/alexvoste/ForgeZero" alt="Go Version"/>
        &nbsp;
        <img src="https://img.shields.io/github/license/alexvoste/ForgeZero" alt="License"/>
        &nbsp;
        <img src="https://img.shields.io/github/commits-since/alexvoste/ForgeZero/v1.5.0" alt="Commits"/>
      </td>
    </tr>
  </table>
</div>

> **Version:** 4.1.0 Citadel &nbsp;·&nbsp; **Language:** Go &nbsp;·&nbsp; **License:** MIT &nbsp;·&nbsp; **Platform:** Linux · Windows · macOS

ForgeZero is a high-performance, zero-overhead build tool for assembly and C developers. It wraps NASM, GAS, FASM, GCC, Clang, Zig, and LD into a single unified command-line interface — no Makefiles, no build scripts, no configuration required to get started.

> Inspired by the simplicity of **Suckless** and the efficiency of **TinyCC**

*Non nobis, Domine, non nobis, sed nomini tuo da Gloriam*

## ⚡ Performance: Full Scaling Benchmark

| Modules | ForgeZero (`fz`) | Traditional (`make -j4`) | Speedup |
|---------|------------------|-------------------------|---------|
| 20 | 19.3 ± 1.2 ms | 45.4 ± 2.3 ms | **2.35×** |
| 50 | 31.1 ± 1.3 ms | 85.0 ± 2.1 ms | **2.73×** |
| 100 | 57.0 ± 5.3 ms | 185.5 ± 7.7 ms | **3.25×** |
| 150 | 73.1 ± 4.3 ms | 229.3 ± 3.6 ms | **3.14×** |

### 🔹 Scaling Efficiency

| Metric | `fz` | `make -j4` |
|--------|-------|-----------|
| Time growth (20→150 modules) | **+279%** | **+405%** |
| Overhead per module | ~0.36 ms | ~1.23 ms |
| I/O operations | **0 intermediate files** | 2× modules (`.o` read/write) |
| Process forks | **1** | ~2× modules + 1 |

> ✅ **Conclusion:** ForgeZero maintains **~3× speedup** at scale. Traditional pipelines suffer from exponential overhead due to process spawning, I/O contention, and CPU cache thrashing.

### 🔹 Why the difference?

| Factor | Traditional (`make + nasm + ld`) | ForgeZero (`fz`) |
|--------|---------------------------------|-------------------|
| **Processes** | 40+ forks (`nasm`×20 + `ld`×20) | **1 process** (integrated pipeline) |
| **I/O** | Writes 20 intermediate `.o` files to disk | **Zero intermediate files** (in-memory) |
| **CPU Cache** | Cold start for every fork | **Hot cache** (code & data stay in L1/L2) |
| **Parallelism** | OS-level (`-j4`), high scheduling overhead | **Goroutines**, zero-cost concurrency |
| **Memory** | GC/Allocator overhead per process | **Zero-allocation hot path** (`0 allocs/op`) |

### 🔹 Scaling Projection

Based on linear scaling from the 20-module benchmark:

| Modules | `fz` (est.) | `make -j4` (est.) | Speedup |
|---------|--------------|-------------------|---------|
| 20 | 19 ms | 45 ms | **2.35×** |
| 50 | ~38 ms | ~115 ms | **~3.0×** |
| 100 | ~65 ms | ~240 ms | **~3.7×** |

*Note: Projections assume linear scaling; real-world results may vary based on I/O and CPU contention.*

### 🔹 How to reproduce

```bash
# Clone and build ForgeZero
git clone https://github.com/forgezero-cli/ForgeZero
cd ForgeZero
go build -o fz ./cmd/fz

# Run the benchmark script
./bench.sh  # Generates 20 test modules and runs hyperfine

---
```
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
    - 15.8 [Virtual Filesystem Layer (VFS)](#158-virtual-filesystem-layer-vfs)
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
18. [Zig Toolchain Backend](#18-zig-toolchain-backend)
19. [Supply Chain Security](#19-supply-chain-security)
    - 19.1 [SBOM Generation (fz sbom)](#191-sbom-generation-fz-sbom)
    - 19.2 [SAST Audit Scanner (fz audit)](#192-sast-audit-scanner-fz-audit)
20. [Reproducible Builds](#20-reproducible-builds)
21. [Source Tree Integrity (fz verify)](#21-source-tree-integrity-fz-verify)
22. [Build Profiler (fz bench)](#22-build-profiler-fz-bench)
23. [WebAssembly (WASM)](#23-webassembly-wasm)
24. [Project Initialization](#24-project-initialization)
25. [LSP & IDE Integration](#25-lsp--ide-integration)
26. [Self-Update](#26-self-update)
27. [Examples](#27-examples)
28. [Exit Codes](#28-exit-codes)
29. [Troubleshooting](#29-troubleshooting)
30. [Roadmap](#30-roadmap)
31. [Virtual Filesystem Layer (Aegis)](#31-virtual-filesystem-layer-aegis)
32. [Aegis Security Core](#32-aegis-security-core)
33. [System Self-Audit (`fz doctor`)](#33-system-self-audit-fz-doctor)
34. [Cross-Platform Readiness](#34-cross-platform-readiness)
35. [Testing Standards (Aegis)](#35-testing-standards-aegis)
36. [Contributing](#36-contributing)
37. [License](#37-license)

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
- Supports cross-compilation to ARM, RISC-V, WASM, and other targets via `-target`.
- Builds static libraries (`.a`) and shared libraries (`.so` / `.dylib`) in addition to executables.
- Compiles C++ (`.cpp`, `.cc`, `.cxx`) with the same strict standards as C.
- Manages external C/ASM dependencies via the built-in package manager (`fz pm`).
- Generates CycloneDX SBOMs with BLAKE3 hashes for supply chain transparency.
- Runs a built-in SAST scanner to detect secrets, license violations, and dangerous patterns.
- Guarantees byte-identical reproducible builds across machines.
- Verifies source tree integrity via BLAKE3 manifests.
- Profiles every build phase with nanosecond precision (`fz bench`).
- Compiles to WebAssembly via `wasm32-emscripten` and `wasm32-wasi`.
- Routes security-sensitive filesystem operations through a virtual layer with TOCTOU-safe `OpenVerified` reads (v3.1.0 Aegis).
- Runs `fz doctor` to verify toolchain presence, directory permissions, and platform integrity before builds.

**What's new in v3.1.0 Aegis:**

- **Virtual filesystem abstraction (`internal/fs`)** — all durable I/O routes through a `FileSystem` interface with Unix and native Windows implementations. `OpenVerified` closes the TOCTOU window between metadata check and read open via `Lstat` + `SameFile` identity verification.
- **Hardened subprocess execution** — external tools (`git`, `ar`, `zig`, `fasm`, `gcc`, `ld`, `nasm`, …) invoke exclusively through `utils.RunCommand`, which resolves binaries with `exec.LookPath`, validates every argument, and runs under a fixed, reproducibility-oriented environment.
- **Atomic secure writes** — manifests, configuration files, SBOM output, and doctor probe files use `SecureWriteFile`: mode `0600` temporary file, flush via close, platform-specific atomic rename (retry loop on Windows file locks).
- **Constant-time toolchain checksum comparison** — optional per-tool BLAKE3 expectations in config are verified with `crypto/subtle.ConstantTimeCompare` to reduce timing leakage during integrity audits.
- **`fz doctor`** — four-stage self-audit: toolchain reachability, recursive permission probe (read/write via `OpenVerified`), platform integrity (`GOOS`/`GOARCH`, VFS backend name, execution root, CPU count), and consolidated human or `--json` reporting.
- **Native Windows I/O path** — `//go:build windows` selects `fs.Windows`, `CleanPath` for drive/UNC normalization, and `renameAtomic` with bounded retries when AV software holds transient locks.
- **Fault-injection test suite** — `fs.Mock` injects `ErrDiskFull`, `ErrPermission`, `ErrTimeout`, and related errors; critical internal packages target **90%+** statement coverage. See [Section 35](#35-testing-standards-aegis).

**What's new in v3.0.0 GLORIA:**

- **Zig toolchain backend** — `zig cc` / `zig c++` as primary or alternative backend for C/C++. Zero external dependencies for cross-compilation: Zig ships all headers, libc, and sysroots internally.
- **SBOM (CycloneDX + BLAKE3)** — `fz sbom` generates a Software Bill of Materials in CycloneDX format with BLAKE3 hashes for every component in the build.
- **SAST audit (`fz audit`)** — built-in security scanner: hardcoded secrets, license compliance (MPL/GPL detection), and dangerous C patterns (unbounded `gets`, format string bugs, unchecked `malloc`, etc.).
- **Reproducible builds (`--reproducible`)** — automatic suppression of build IDs, timestamps, and non-deterministic path references; object files sorted before linking for byte-identical output across machines.
- **Source tree verification (`fz verify`)** — generates and checks BLAKE3 manifests of the entire source tree to detect unauthorized modifications.
- **Symlink boundary protection** — the recursive scanner now validates that every symlink resolves to a path inside the project root, blocking symlink race attacks.
- **`fz bench`** — nanosecond-precision build profiler showing time per file, linker, and audit phase. Supports multi-run averaging and JSON output.
- **Multithreading safety** — all race conditions in the parallel build and logging pipeline eliminated; verified with `go test -race`.
- **FASM improvements** — automatic `format ELF64` injection and correct `-dDEBUG=1` / debug symbol pass-through for FASM files.
- **WebAssembly** — `wasm32-emscripten` and `wasm32-wasi` targets; Zig backend is the recommended zero-dependency path for WASI.

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

When using `-zig`, no prefixed toolchain is required — Zig resolves the target internally. See [Section 18](#18-zig-toolchain-backend).

### Optional tools (used internally)

| Tool       | Purpose |
|------------|---------|
| `nm`       | Pre-link duplicate symbol check (primary) |
| `objdump`  | Fallback for symbol check |
| `readelf`  | Second fallback for symbol check |
| `git`      | Required for `fz pm add` (package manager) |
| `zig`      | Required for `-zig` backend (v3.0.0+) |
| `emcc`     | Required for `wasm32-emscripten` target (v3.0.0+) |

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

**Install Zig (optional, for `-zig` backend):**

```bash
wget https://ziglang.org/download/0.13.0/zig-linux-x86_64-0.13.0.tar.xz
tar -xf zig-linux-x86_64-0.13.0.tar.xz
sudo mv zig-linux-x86_64-0.13.0 /opt/zig
echo 'export PATH="$PATH:/opt/zig"' >> ~/.bashrc
source ~/.bashrc
zig version
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

**Install Zig (optional):**

```bash
# Download from ziglang.org — no dnf package in older releases
wget https://ziglang.org/download/0.13.0/zig-linux-x86_64-0.13.0.tar.xz
tar -xf zig-linux-x86_64-0.13.0.tar.xz
sudo mv zig-linux-x86_64-0.13.0 /opt/zig
echo 'export PATH="$PATH:/opt/zig"' >> ~/.bashrc
```

**Install ForgeZero:**

```bash
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
```

---

### 3.3 Linux — Arch Linux / Manjaro

**Install system dependencies:**

```bash
sudo pacman -S --noconfirm nasm gcc binutils clang git zig

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
brew install nasm gcc go git zig
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

As of **v3.1.0 Aegis**, ForgeZero ships a first-class native Windows filesystem backend (`internal/fs.Windows`) and Windows-specific atomic rename retry logic. WSL2 remains the path of least resistance for full toolchain parity, but native Windows builds no longer depend on translating paths through a Linux compatibility layer for ForgeZero's own I/O.

The recommended approach for day-to-day development on Windows is still **WSL2** (Windows Subsystem for Linux), which provides a full Linux environment and the best compatibility with all ForgeZero features.

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

Native Windows I/O behavior (path cleaning, `OpenVerified`, locked-file renames) is documented in [Section 34](#34-cross-platform-readiness).

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
go test ./internal/... -cover
```

The `internal/` tree is the primary coverage target for v3.1.0 Aegis. Critical packages (`pkgman`, `fs`, `doctor`, `config`, `zig`, and others) are held to **90%+** statement coverage; fault injection via `fs.Mock` is mandatory for I/O failure paths. See [Section 35](#35-testing-standards-aegis).

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

**Build with Zig backend (no extra toolchain needed):**

```bash
fz -cc main.c -zig -target aarch64-linux-musl
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

**Generate a Software Bill of Materials:**

```bash
fz sbom
```

**Run the security audit:**

```bash
fz audit
```

**Reproducible build:**

```bash
fz -dir ./src --reproducible
```

**Verify source tree integrity:**

```bash
fz verify
```

**Profile the build:**

```bash
fz bench
```

**Build for WebAssembly (WASI, via Zig):**

```bash
fz -cc main.c -zig -target wasm32-wasi -out main.wasm
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
| `.fasm`            | Assembly   | FASM             | Requires separate install; auto `format ELF64` injection in v3.0.0 |
| `.c`               | C          | GCC, Clang, or `zig cc` | Strict flags + sanitizers by default |
| `.cpp` / `.cc` / `.cxx` | C++   | G++, Clang++, or `zig c++` | Same strict flags as C (v1.7.0+) |

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

At least one of `-asm`, `-cc`, `-dir`, `-init`, `-shell`, `pm`, `sbom`, `audit`, `verify`, `bench`, `doctor`, or a valid config file must be present.

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
| `-target` | `<triple>` | — | Cross-compilation target triple (e.g. `arm-linux-gnueabihf`, `wasm32-wasi`). |
| `-zig` | — | off | Use Zig (`zig cc` / `zig c++`) as the compiler backend. See [Section 18](#18-zig-toolchain-backend). |
| `--reproducible` | — | off | Enable deterministic builds: suppress build IDs, timestamps, path references; sort objects. |
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
| `-manifest` | `<file>` | `.fz.manifest` | Path to the BLAKE3 source manifest used by `fz verify`. |
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

### Security & Integrity Sub-commands

| Command | Description |
|---------|-------------|
| `fz sbom` | Generate a CycloneDX SBOM with BLAKE3 hashes for all build components. |
| `fz sbom -o <path>` | Write the SBOM to a specific output file (default: `sbom.cdx.json`). |
| `fz sbom -dir <dir>` | Generate SBOM scoped to a specific source directory. |
| `fz sbom -json` | Emit the SBOM as JSON (always true; flag reserved for future format options). |
| `fz audit` | Run the built-in SAST scanner: secrets, license compliance, and dangerous patterns. |
| `fz audit -dir <dir>` | Audit a specific directory. |
| `fz audit -json` | Emit the audit report as a JSON object to stdout. |
| `fz verify` | Verify the current source tree against the stored BLAKE3 manifest. |
| `fz verify --generate` | Generate a new BLAKE3 manifest of the current source tree. |
| `fz verify --strict` | Also report UNTRACKED files (files on disk not in the manifest). |
| `fz verify -manifest <f>` | Use a specific manifest file instead of the default `.fz.manifest`. |
| `fz doctor` | Run the Aegis self-audit: toolchain, permissions, platform integrity. |
| `fz doctor -root <dir>` | Audit a specific project root (default: current working directory). |
| `fz doctor -json` | Emit the audit report as JSON; exit code `1` if `healthy` is false. |

### Performance Sub-commands

| Command | Description |
|---------|-------------|
| `fz bench` | Profile the build: measure and report nanosecond-precision timing for every phase. |
| `fz bench -n <N>` | Run the build N times and report average and standard deviation per phase. |
| `fz bench -json` | Emit the benchmark report as a JSON object. |

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

> **Note:** Sanitizers are automatically disabled for WebAssembly targets (`wasm32-*`). See [Section 23](#23-webassembly-wasm).

---

## 10. C++ Compilation

Added in **v1.7.0**. ForgeZero compiles `.cpp`, `.cc`, and `.cxx` files using `g++` or `clang++` (or `zig c++` when `-zig` is active). The same strict warning flags applied to C are applied identically to C++:

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

`fz` dispatches `.c` files to `gcc`/`clang`/`zig cc` and `.cpp`/`.cc`/`.cxx` files to `g++`/`clang++`/`zig c++` automatically. All objects are linked together in a single step.

**Disable sanitizers for release:**

```bash
fz -cc main.cpp -sanitize=false
```

---

## 11. Cross-Compilation

Added in **v1.9.0**. The `-target` flag enables cross-compilation to any architecture supported by the GNU toolchain installed on your system. In **v3.0.0**, combining `-target` with `-zig` removes the need to install prefixed toolchain packages entirely.

### Basic Usage

```bash
fz -cc main.c -target arm-linux-gnueabihf
fz -cc main.c -target aarch64-linux-gnu
fz -cc main.c -target riscv64-linux-gnu
fz -dir ./src -target arm-linux-gnueabihf -out firmware

# With Zig backend — no cross-compiler package required
fz -cc main.c -zig -target aarch64-linux-musl
fz -cc main.c -zig -target riscv64-linux-musl
```

### How It Works

When `-target <triple>` is set **without** `-zig`, `fz` constructs the expected prefixed compiler and linker names by prepending the triple to the tool name:

- Compiler: `<triple>-gcc` (e.g. `arm-linux-gnueabihf-gcc`)
- C++ compiler: `<triple>-g++`
- Linker: `<triple>-gcc` or `<triple>-ld` depending on the linking mode
- Archiver: `<triple>-ar` (when `-type static`)

When `-target <triple>` is set **with** `-zig`, `fz` passes the triple directly to `zig cc` via its own `-target` flag. No prefixed binary is looked up on `PATH`.

`fz` verifies that the required compiler is available on `PATH` before starting the build. If the cross-compiler is not found, the build exits with code `2` and a clear error message naming the missing tool.

### Installing Cross-Compilers (without Zig)

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
- For WebAssembly cross-compilation targets (`wasm32-*`), see [Section 23](#23-webassembly-wasm).

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
- With the Zig backend: `fz -dir ./src -type static -lib mylib -zig -target aarch64-linux-musl`.

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

As of **v3.0.0 GLORIA**, the parallel build and logging pipeline is fully race-condition-free, verified with `go test -race` across the complete test suite.

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

### 15.8 Virtual Filesystem Layer (VFS)

> **New in v3.1.0 Aegis**

ForgeZero no longer calls `os.Open`, `os.WriteFile`, or `os.Rename` directly from security-sensitive code paths. The `internal/fs` package defines a `FileSystem` interface; `internal/utils` binds a process-wide implementation through `SetFileSystem` / `fileSystem()` so production code, tests, and fault injection share one API.

| Component | Role |
|-----------|------|
| `FileSystem` interface | Contract for mkdir, read, write, verified open, temp files, rename, stat, symlinks |
| `fs.Unix` | POSIX implementation (`//go:build !windows`) |
| `fs.Windows` | Native Windows implementation (`//go:build windows`) |
| `fs.Default` | Platform `Default` variable (`Unix{}` or `Windows{}`) selected at compile time |
| `fs.Mock` | Test double that delegates to `Base` and injects per-operation errors |
| `utils.SetFileSystem` | Runtime swap used only in tests |

Operations that touch project secrets, manifests, or configuration — including `SecureWriteFile`, `ReadFileSecure`, `OpenVerifiedRead`, and `fz doctor`'s tree walk — use this layer.

Full architectural detail: [Section 31](#31-virtual-filesystem-layer-aegis).

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
| `backend` | string | `auto` | Compiler backend: `auto`, `gcc`, `clang`, or `zig` |
| `reproducible` | bool | `false` | Enable reproducible (deterministic) build mode |
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
3. Check symlink boundary — skip and warn if the resolved path is outside the project root.
4. Check `include` patterns — skip if none match (when `include` is set).
5. Check supported extensions — skip all others.

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
# backend: zig          # use Zig toolchain
# reproducible: true    # deterministic byte-identical output

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

As of **v3.0.0 GLORIA**, ForgeZero automatically injects `format ELF64 executable` when no `format` directive is found at the top of the file, and correctly passes `-dDEBUG=1` and debug symbol flags when `-debug` is active. See [Section 17.3 details](#173-fasm-fasm) and [Section 18.6](#186-fasm-improvements-v300) for the full breakdown.

```asm
; hello.fasm — format directive can be omitted for ELF64 targets (v3.0.0+)
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

## 18. Zig Toolchain Backend

> **New in v3.0.0 GLORIA**

ForgeZero now supports Zig as a first-class compiler backend for C and C++ compilation. When the `-zig` flag is set, `zig cc` replaces `gcc`/`clang` and `zig c++` replaces `g++`/`clang++` throughout the entire build pipeline — compilation, linking, and archiving.

### 18.1 Why Zig?

The Zig compiler ships as a single self-contained binary that includes:

- A full copy of musl libc, glibc stubs, and WASI sysroot for every supported target.
- All necessary C and C++ runtime headers.
- A Clang-based C/C++ frontend (`zig cc` is `clang` under the hood).
- Built-in cross-compilation support for every target triple that Zig supports — no sysroot packages to install.

This makes ForgeZero genuinely zero-dependency for cross-compilation when the Zig backend is active.

### 18.2 Activating the Zig Backend

```bash
# Single file
fz -cc main.c -zig

# Full directory
fz -dir ./src -zig

# With cross-compilation (no extra package needed)
fz -cc main.c -zig -target aarch64-linux-musl
fz -cc main.c -zig -target riscv64-linux-musl
fz -cc main.c -zig -target x86_64-windows-gnu

# Static library via Zig
fz -dir ./src -type static -lib mylib -zig -target aarch64-linux-musl
```

In `.fz.yaml`:

```yaml
source_dirs:
  - src
output: myapp
backend: zig
target: aarch64-linux-musl
sanitize: false
```

### 18.3 Toolchain Selection Logic

| Flags active | C compiler | C++ compiler |
|---|---|---|
| (default) | `gcc` or `clang` | `g++` or `clang++` |
| `-zig` | `zig cc` | `zig c++` |
| `-zig -target <T>` | `zig cc -target T` | `zig c++ -target T` |
| `-zig -strict` | `zig cc` (Clang-based, full sanitizers) | `zig c++` |

When `-zig` and `-target` are both set, the target triple is passed directly to Zig's own `-target` flag. ForgeZero does **not** search `PATH` for a prefixed compiler binary.

### 18.4 Cross-Compilation Without Extra Packages

```bash
# ARM64 musl — no gcc-aarch64-linux-gnu needed
fz -cc main.c -zig -target aarch64-linux-musl

# RISC-V bare-metal
fz -cc kernel.c -zig -target riscv64-linux-musl -mode raw

# Windows executable from Linux
fz -cc win_app.c -zig -target x86_64-windows-gnu
```

### 18.5 Installing Zig

```bash
# Debian / Ubuntu
wget https://ziglang.org/download/0.13.0/zig-linux-x86_64-0.13.0.tar.xz
tar -xf zig-linux-x86_64-0.13.0.tar.xz
sudo mv zig-linux-x86_64-0.13.0 /opt/zig
echo 'export PATH="$PATH:/opt/zig"' >> ~/.bashrc
source ~/.bashrc

# Arch Linux
sudo pacman -S zig

# macOS
brew install zig

# Verify
zig version
```

### 18.6 FASM Improvements (v3.0.0)

The FASM backend received two improvements in v3.0.0 GLORIA, independent of the Zig backend:

**Automatic `format ELF64` injection.** When ForgeZero compiles a `.fasm` file and the file does not begin with a `format` directive, it automatically prepends `format ELF64 executable` in a temporary preprocessed copy before passing the file to FASM. The original source file is never modified. If a `format` directive is already present, it is left unchanged.

**Debug flag pass-through.** When `-debug` is passed to `fz`, ForgeZero now correctly injects `-dDEBUG=1` into the FASM command line, making the `DEBUG` symbol available for conditional assembly. DWARF debug information is also emitted in the ELF64 output when `-debug` is active, enabling source-level debugging with `gdb`.

```asm
; Conditional debug code in FASM using the injected DEBUG symbol
if defined DEBUG
    ; print register state, extra checks, etc.
end if
```

```bash
# Debug build of a FASM file with full symbol output
fz -asm boot.fasm -debug
gdb ./boot
(gdb) break _start
(gdb) run
```

---

## 19. Supply Chain Security

> **New in v3.0.0 GLORIA** · hardened in **v3.1.0 Aegis** ([Section 32](#32-aegis-security-core))

### 19.1 SBOM Generation (fz sbom)

`fz sbom` generates a Software Bill of Materials (SBOM) for the current project in the **CycloneDX** standard format. Every component — source files, vendored packages, and detected system libraries — is listed with its BLAKE3 hash, version (where available), and SPDX license identifier.

#### Running fz sbom

```bash
# Generate SBOM for the current project (output: sbom.cdx.json)
fz sbom

# Specify a custom output path
fz sbom -o /tmp/myproject-sbom.cdx.json

# Scoped to a specific source directory
fz sbom -dir ./src -o release-sbom.cdx.json
```

#### What CycloneDX Is

CycloneDX is an open SBOM standard maintained by OWASP and widely adopted for supply-chain compliance (NIST SSDF, US Executive Order 14028). The file produced by `fz sbom` is valid CycloneDX JSON and can be imported directly into Dependency-Track, Grype, Syft, or any other compatible platform.

#### What the SBOM Contains

Each component entry includes:

- **name** — file path relative to the project root, or package name for vendored dependencies.
- **version** — Git commit SHA for vendored packages; absent for local source files.
- **hashes** — BLAKE3 hash of the file contents at generation time.
- **type** — one of `source-file`, `vendored-package`, or `system-library`.
- **licenses** — SPDX identifier detected from file headers, `LICENSE` files, or `.fz.yaml` metadata.

#### Vendored Package Integrity

Any package installed via `fz pm add` or `fz pm install` is included in the SBOM with its Git commit SHA as the version and its BLAKE3 hash as the integrity value. If the on-disk content of a vendored package has changed since installation, the SBOM flags the component as `modified`.

#### BLAKE3 Hashing

All hashes in the SBOM are computed using BLAKE3, consistent with the rest of ForgeZero's internal hashing infrastructure. Generating an SBOM for a large project takes milliseconds.

---

### 19.2 SAST Audit Scanner (fz audit)

`fz audit` is a built-in static analysis scanner that inspects the current project for security issues without requiring any external tools. It performs three classes of checks.

#### Running fz audit

```bash
# Full audit on the current project
fz audit

# Audit a specific directory
fz audit -dir ./src

# JSON output for CI/CD
fz audit -json
```

#### Check 1 — Hardcoded Secrets

`fz audit` scans all source files for patterns consistent with credentials committed directly into code. The scanner uses a curated set of entropy-weighted regular expressions.

Patterns detected include:

- Provider-specific API keys: `AWS_SECRET_ACCESS_KEY`, `GITHUB_TOKEN`, `SLACK_TOKEN`, and over forty other formats.
- Generic high-entropy strings assigned to variables named `key`, `secret`, `password`, `token`, or `credential`.
- Private key PEM blocks (`-----BEGIN ... PRIVATE KEY-----`).
- Connection strings containing plaintext passwords (e.g. `postgres://user:password@host`).

Each finding includes the file path, line number, matched pattern type, and severity (`HIGH` or `CRITICAL`). The scanner never prints the actual secret value in its output to prevent leaking it into CI logs.

#### Check 2 — License Compliance

`fz audit` inspects every file in `vendor/` and any source file containing SPDX identifiers or common license headers. It flags licenses that may introduce restrictions incompatible with your project.

| License | Concern | Severity |
|---|---|---|
| MPL-2.0 | Modified files must be released under MPL | WARNING |
| GPL-2.0 / GPL-3.0 | Strong copyleft; may require entire project to be GPL | HIGH |
| AGPL-3.0 | Network copyleft; applies to server-side deployments | HIGH |
| LGPL variants | Weak copyleft; generally safe for dynamic linking | INFO |
| Proprietary / non-OSI in `vendor/` | May prohibit redistribution | CRITICAL |

The check reads SPDX identifiers from file headers (`// SPDX-License-Identifier: ...`), `LICENSE` files, and `.fz.yaml` package metadata. No external scanner or network request is involved.

#### Check 3 — Dangerous Patterns

`fz audit` detects dangerous constructs in C source code and build configuration files.

In C source files:

- `gets()`, `sprintf()`, `strcpy()` without bounds — use `fgets()`, `snprintf()`, `strncpy()` instead.
- `alloca()` inside loops — potential stack overflow.
- Format string vulnerabilities: `printf(user_input)` without a format specifier.
- Unchecked return values from `malloc()`, `realloc()`, and file I/O functions.

In configuration files and shell scripts:

- Inline credential assignments (`export PASSWORD=...`).
- Insecure file permission settings (`chmod 777`).
- `curl | sh` or `wget | bash` patterns.

#### Audit Output Format

By default `fz audit` prints a colour-coded summary grouped by severity. Pass `-json` for a machine-readable report:

```json
{
  "status": "findings",
  "total": 3,
  "findings": [
    {
      "type": "hardcoded_secret",
      "severity": "CRITICAL",
      "file": "src/config.c",
      "line": 42,
      "pattern": "AWS_SECRET_ACCESS_KEY assignment",
      "description": "High-entropy string assigned to variable named 'secret'"
    }
  ]
}
```

Exit code is `0` when no findings are present, `1` when any finding of severity WARNING or above is detected.

---

## 20. Reproducible Builds

> **New in v3.0.0 GLORIA**

ForgeZero v3.0.0 GLORIA guarantees that two separate builds of the same source tree on two different machines produce **byte-for-byte identical** output binaries when `--reproducible` is active.

### 20.1 Why Reproducible Builds Matter

Non-deterministic builds make it impossible to independently verify that a distributed binary was compiled from a specific source revision. They also make it harder to detect supply-chain attacks, because a compromised build system injects differences into the binary that only show up when two independently built copies are compared.

### 20.2 Sources of Non-Determinism Eliminated

**Build IDs.** GCC and Clang embed a randomly generated build ID in an ELF `.note.gnu.build-id` section. ForgeZero suppresses it with `-Wl,--build-id=none`.

**Timestamps.** The `__DATE__` and `__TIME__` preprocessor macros embed the current wall-clock time. ForgeZero sets `SOURCE_DATE_EPOCH` to the timestamp of the most recent Git commit in the project (or zero if not a Git repository) and overrides both macros with deterministic values.

**Path mapping.** DWARF debug information embeds absolute source file paths. ForgeZero passes `-fdebug-prefix-map=/absolute/project/path=.` to the compiler and assembler, replacing all absolute paths with project-relative ones.

**Object sort order.** The order in which the OS returns directory entries is non-deterministic. ForgeZero sorts all object file paths lexicographically before invoking the linker.

### 20.3 Enabling Reproducible Builds

```bash
# Command-line
fz -dir ./src --reproducible

# .fz.yaml
reproducible: true
```

ForgeZero prints a summary of measures applied at the end of the build:

```
[fz] Reproducible build mode enabled.
[fz]   Build ID:        suppressed (-Wl,--build-id=none)
[fz]   Timestamps:      SOURCE_DATE_EPOCH=1716220800 (2024-05-20T12:00:00Z)
[fz]   Path mapping:    -fdebug-prefix-map applied
[fz]   Object ordering: lexicographic sort applied
[fz] Build complete: myapp (sha256: e3b0c44298fc1c149afb...)
```

### 20.4 Verifying Reproducibility

```bash
# Build on machine A
fz -dir ./src --reproducible -out release_a
sha256sum release_a

# Build on machine B (same source, same commit)
fz -dir ./src --reproducible -out release_b
sha256sum release_b

# Both hashes must match
```

---

## 21. Source Tree Integrity (fz verify)

> **New in v3.0.0 GLORIA**

### 21.1 fz verify — Overview

`fz verify` generates and checks a BLAKE3 manifest of every source file in the project. It provides a tamper-evident record of the source tree at a known-good state and detects unauthorized changes to any file.

### 21.2 Generating a Manifest

```bash
# Generate a manifest of the current source tree (output: .fz.manifest)
fz verify --generate

# Specify a custom manifest path
fz verify --generate -manifest ./release.manifest
```

The manifest is a plain text file, one line per source file:

```
<BLAKE3-hex-hash>  <relative-file-path>

# Example:
3a4e9f1b2c...  src/main.c
7b2d0f4e8a...  src/utils.c
c1f9e23d01...  include/api.h
```

### 21.3 Verifying Against a Manifest

```bash
# Check the current tree against the default manifest (.fz.manifest)
fz verify

# Check against a specific manifest
fz verify -manifest ./release.manifest

# Also report files on disk that are not in the manifest
fz verify --strict
```

`fz verify` reports three categories of finding:

- **MODIFIED** — file exists on disk but its BLAKE3 hash does not match the recorded value.
- **MISSING** — file listed in the manifest is not present on disk.
- **UNTRACKED** — file exists on disk but is not listed in the manifest (only with `--strict`).

Exit code is `0` if no MODIFIED or MISSING files are found, `1` if any integrity violation is detected.

### 21.4 CI/CD Integration

```bash
# After checkout, verify the source tree before building
fz verify -manifest ./release.manifest
# Exits 1 and fails the pipeline if any file has been tampered with.
```

### 21.5 Symlink Boundary Protection

The recursive directory scanner introduced in v3.0.0 validates every symlink it encounters. When a symlink is found, its target is resolved to an absolute canonical path and checked against the project root. If the resolved path falls outside the project root, the symlink is skipped and a warning is emitted:

```
[fz] WARNING: Skipping symlink 'src/external_link' — resolved path
              '/etc/passwd' is outside the project boundary.
              This may indicate a symlink race attack. Investigate
              before proceeding.
```

Symlinks that resolve inside the project root are followed normally. This protection is always active and cannot be disabled — it is a security invariant, not a configurable option.

---

## 22. Build Profiler (fz bench)

> **New in v3.0.0 GLORIA**

`fz bench` runs a full build of the project and records the elapsed wall-clock time for every stage with nanosecond precision. It provides a structured, repeatable profiling workflow integrated directly into ForgeZero.

### 22.1 Basic Usage

```bash
# Profile the current project
fz bench

# Profile a specific directory
fz bench -dir ./src

# Run 5 iterations; report average and standard deviation per phase
fz bench -n 5

# JSON output for CI/CD analysis
fz bench -json
```

### 22.2 Output Format

```
fz bench — ForgeZero Build Profiler
Project: ./src   Files: 12   Mode: auto   Cache: cold

Phase                       Start (ns)      Duration        % Total
─────────────────────────────────────────────────────────────────────
Cache check                        0 ns       421,330 ns      0.12%
Compile: src/main.c          421,330 ns    18,204,772 ns      5.21%
Compile: src/utils.c      18,626,102 ns    12,409,003 ns      3.55%
Compile: src/parser.c     31,035,105 ns    87,304,221 ns     24.99%
... (12 files total)
Pre-link symbol check    298,104,552 ns     1,203,449 ns      0.34%
Link                     299,307,001 ns    46,882,004 ns     13.42%
Audit (fz audit)         346,189,005 ns    12,034,221 ns      3.44%
─────────────────────────────────────────────────────────────────────
Total                                      349,323,226 ns    100.00%
                                           (~349 ms wall clock)
```

### 22.3 Cache-Warm vs Cache-Cold Profiling

```bash
# Cold build (no cache hits) — simulates fresh checkout
fz bench -dir ./src -no-cache

# Warm build (all files cached) — run twice; first run warms the cache
fz bench -dir ./src
fz bench -dir ./src
```

### 22.4 JSON Output

```bash
fz bench -json
```

```json
{
  "total_ns": 349323226,
  "total_ms": 349,
  "cache": "cold",
  "phases": [
    { "name": "Compile: src/main.c", "start_ns": 421330,    "duration_ns": 18204772 },
    { "name": "Link",                "start_ns": 299307001, "duration_ns": 46882004 }
  ]
}
```

---

## 23. WebAssembly (WASM)

> **New in v3.0.0 GLORIA**

ForgeZero v3.0.0 adds WebAssembly as a supported compilation target. C source files can be compiled to `.wasm` modules targeting either Emscripten (browser) or WASI (server-side / cloud-native runtimes).

### 23.1 Supported Targets

| Target triple | Runtime | Use case |
|---|---|---|
| `wasm32-emscripten` | Emscripten / Browser | Browser WebAssembly with full libc emulation |
| `wasm32-wasi` | Wasmtime, WasmEdge, WAMR, etc. | Server-side / cloud-native WASM modules |

### 23.2 Building for wasm32-emscripten

The `wasm32-emscripten` target requires the Emscripten SDK (`emcc`) to be installed and activated.

```bash
# Install Emscripten (one-time setup)
git clone https://github.com/emscripten-core/emsdk.git
cd emsdk && ./emsdk install latest && ./emsdk activate latest
source ./emsdk_env.sh

# Compile C to WebAssembly for browser use
fz -cc main.c -target wasm32-emscripten -out main.js
# Produces main.wasm + main.js (JavaScript glue loader)
```

Emscripten provides a full POSIX libc emulation layer. Code that uses `stdio.h`, `stdlib.h`, and similar standard headers compiles without modification.

### 23.3 Building for wasm32-wasi

The `wasm32-wasi` target produces standalone `.wasm` modules conforming to the WASI specification, executable by any WASI-compatible runtime without a browser or JavaScript engine.

**Recommended approach — Zig backend (no extra SDK needed):**

```bash
# Zig ships the WASI sysroot; no separate SDK required
fz -cc main.c -zig -target wasm32-wasi -out main.wasm

# Run with wasmtime
wasmtime main.wasm
```

**Alternative — Clang + WASI SDK:**

```bash
# Requires: https://github.com/WebAssembly/wasi-sdk
fz -cc main.c -target wasm32-wasi -cc-flag "--sysroot=/opt/wasi-sdk/share/wasi-sysroot"
```

### 23.4 WASM in .fz.yaml

```yaml
# .fz.yaml for a WASI library module
source_dirs:
  - src
output: mymodule.wasm
backend: zig
target: wasm32-wasi
sanitize: false
flags:
  cc:
    - -O2
    - -fvisibility=hidden
```

### 23.5 Sanitizers and WASM

AddressSanitizer and UndefinedBehaviorSanitizer are not supported for WebAssembly targets. ForgeZero automatically disables sanitizers when a `wasm32-*` target is detected, regardless of the `-sanitize` flag, and prints a notice:

```
[fz] NOTE: Sanitizers disabled for target wasm32-wasi
           (ASan/UBSan are not supported for WebAssembly targets).
           Use -sanitize=false to suppress this notice.
```

### 23.6 Installing wasmtime (optional, for running WASI modules)

```bash
# Linux / macOS
curl https://wasmtime.dev/install.sh -sSf | bash

# macOS
brew install wasmtime

# Arch Linux
sudo pacman -S wasmtime
```

---

## 24. Project Initialization

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

## 25. LSP & IDE Integration

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

## 26. Self-Update

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

## 27. Examples

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

### Cross-compile for ARM (with system toolchain)

```bash
fz -cc main.c -target arm-linux-gnueabihf -sanitize=false
```

---

### Cross-compile for ARM64 musl (Zig backend — no packages needed)

```bash
fz -cc main.c -zig -target aarch64-linux-musl -sanitize=false
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
fz pm add github.com/me/my-lib
fz pm add github.com/me/my-lib@v1.2.3
fz pm install esp-idf
fz pm search crypto
fz pm list
fz pm update
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

### Generate a Software Bill of Materials

```bash
fz sbom
cat sbom.cdx.json
```

---

### Run the security audit

```bash
fz audit
fz audit -json | tee audit_report.json
```

---

### Reproducible build

```bash
fz -dir ./src --reproducible
sha256sum ./src
```

---

### Generate and verify source tree manifest

```bash
# Generate manifest at a known-good state
fz verify --generate

# Later — verify nothing has changed
fz verify
```

---

### Profile the build

```bash
fz bench -dir ./src
fz bench -dir ./src -n 5 -json | tee bench_report.json
```

---

### Build for WebAssembly (WASI, via Zig)

```bash
fz -cc main.c -zig -target wasm32-wasi -out main.wasm
wasmtime main.wasm
```

---

### Build for WebAssembly (browser, via Emscripten)

```bash
source /path/to/emsdk/emsdk_env.sh
fz -cc main.c -target wasm32-emscripten -out main.js
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

## 28. Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success — binary was produced without errors; or `fz verify` / `fz audit` found no violations. |
| `1` | Build error — assembler, compiler, or linker failed; duplicate global symbol detected; `fz verify` found MODIFIED or MISSING files; `fz audit` found findings of WARNING severity or above. |
| `2` | Argument error — invalid or missing flags, source file not found, cross-compiler not found on PATH, or unreadable configuration file. |

---

## 29. Troubleshooting

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

### `zig: command not found`

```bash
# Download from ziglang.org
wget https://ziglang.org/download/0.13.0/zig-linux-x86_64-0.13.0.tar.xz
tar -xf zig-linux-x86_64-0.13.0.tar.xz
sudo mv zig-linux-x86_64-0.13.0 /opt/zig
echo 'export PATH="$PATH:/opt/zig"' >> ~/.bashrc
source ~/.bashrc

# Or via package manager (Arch / macOS)
sudo pacman -S zig
brew install zig
```

---

### Cross-compiler not found (without Zig)

Install the appropriate cross-compilation toolchain:

```bash
sudo apt install gcc-arm-linux-gnueabihf     # Debian / Ubuntu
sudo dnf install gcc-arm-linux-gnu           # Fedora
sudo pacman -S arm-linux-gnueabihf-gcc       # Arch
```

Or switch to the Zig backend to avoid installing cross-compiler packages:

```bash
fz -cc main.c -zig -target arm-linux-gnueabihf
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

### Sanitizers silently disabled for WASM target

This is expected behaviour. ASan and UBSan are not supported for WebAssembly targets. ForgeZero disables them automatically and prints a notice. Pass `-sanitize=false` to suppress the notice.

---

### `fz verify` reports MODIFIED files unexpectedly

Re-generate the manifest from the current known-good state:

```bash
fz verify --generate
```

If the changes are unexpected, investigate which files were modified and by what process before accepting the new manifest.

---

### `fz audit` false positive on a secret pattern

The SAST scanner uses heuristic entropy-based detection. If a false positive occurs, annotate the line with a suppression comment:

```c
const char *example = "not-a-real-key-just-documentation"; // fz-audit: ignore
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

## 30. Roadmap

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
| Zig toolchain backend (`-zig`) | ✅ Done (v3.0.0) |
| SBOM generation (`fz sbom`, CycloneDX + BLAKE3) | ✅ Done (v3.0.0) |
| SAST audit scanner (`fz audit`) | ✅ Done (v3.0.0) |
| Reproducible builds (`--reproducible`) | ✅ Done (v3.0.0) |
| Source tree verification (`fz verify`) | ✅ Done (v3.0.0) |
| Symlink boundary protection | ✅ Done (v3.0.0) |
| Build profiler (`fz bench`, nanosecond precision) | ✅ Done (v3.0.0) |
| Race-condition-free parallel pipeline | ✅ Done (v3.0.0) |
| FASM native ELF64 auto-injection | ✅ Done (v3.0.0) |
| FASM debug flag pass-through (`-dDEBUG=1`) | ✅ Done (v3.0.0) |
| WebAssembly (`wasm32-emscripten` / `wasm32-wasi`) | ✅ Done (v3.0.0) |
| Colored terminal output (green success / red error) | Planned |
| GDB integration and improved debug workflow | Planned |
| Man page (`man fz`) | Planned |
| Windows native support without WSL2 | In progress |
| macOS full support and testing | In progress |
| VFS abstraction + `OpenVerified` TOCTOU hardening | ✅ Done (v3.1.0) |
| Aegis hardened `RunCommand` wrapper | ✅ Done (v3.1.0) |
| `SecureWriteFile` atomic write pipeline | ✅ Done (v3.1.0) |
| Constant-time toolchain checksum verify | ✅ Done (v3.1.0) |
| `fz doctor` self-audit command | ✅ Done (v3.1.0) |
| Native Windows `fs.Windows` + rename retry | ✅ Done (v3.1.0) |
| 90%+ coverage + `fs.Mock` fault injection | ✅ Done (v3.1.0) |

---

## 31. Virtual Filesystem Layer (Aegis)

> **Package:** `internal/fs` · **Consumers:** `internal/utils`, `internal/doctor`, `internal/verify`, `internal/sbom`, `internal/pkgman` (via utils), all manifest and config writers

ForgeZero v3.1.0 introduces a deliberate separation between *what* filesystem operations the build tool requires and *how* the host operating system performs them. The goal is twofold: enable deterministic fault-injection tests without patching `os` globally, and centralize symlink and path-substitution defenses in one audited code path.

### 31.1 Design: The `FileSystem` Interface

All durable and security-sensitive operations flow through a single interface:

```go
type FileSystem interface {
    MkdirAll(path string, perm os.FileMode) error
    WriteFile(path string, data []byte, perm os.FileMode) error
    ReadFile(path string) ([]byte, error)
    Open(path string) (io.ReadCloser, error)
    OpenVerified(path string) (io.ReadCloser, error)
    CreateTemp(dir, pattern string) (*os.File, error)
    Remove(name string) error
    RemoveAll(path string) error
    Rename(oldpath, newpath string) error
    Stat(name string) (os.FileInfo, error)
    Lstat(name string) (os.FileInfo, error)
    ReadDir(name string) ([]os.DirEntry, error)
    Chmod(name string, mode os.FileMode) error
    Readlink(name string) (string, error)
    EvalSymlinks(path string) (string, error)
    SameFile(a, b os.FileInfo) bool
}
```

The interface is intentionally narrow. It mirrors the subset of `os` and `path/filepath` calls ForgeZero actually needs, not a full virtual filesystem framework. Higher-level packages never import `os` for manifest reads, config writes, or doctor tree walks when a verified or atomic path exists in `internal/utils`.

**Binding model:**

- At compile time, `fs.Default` is set to `Unix{}` or `Windows{}` via build tags in `default_unix.go` / `default_windows.go`.
- At run time, `utils.fileSystem()` returns the active implementation (production: `fs.Default`; tests: `fs.Mock` installed with `utils.SetFileSystem`).
- `fs.ImplName()` reports `"unix"` or `"windows"` for inclusion in `fz doctor` platform output.

### 31.2 Unix Implementation

The `Unix` type (`unix.go`, build tag `!windows`) is a thin, explicit wrapper over the Go standard library. It exists so that:

1. Every platform-specific behavior is visible in one file per OS.
2. Tests can substitute `Mock` without build-tag fragmentation in consumer packages.

`OpenVerified` on Unix executes the following sequence:

```go
pre, err := os.Lstat(path)          // metadata without following final symlink hop
if pre.Mode()&os.ModeSymlink != 0 {
    return nil, ErrSymlink            // symlinks are rejected for verified reads
}
f, err := os.Open(path)               // open the same path
post, err := f.Stat()                 // metadata of the opened descriptor
if !os.SameFile(pre, post) {
    f.Close()
    return nil, ErrPathChanged        // inode/device identity changed between check and use
}
return f, nil
```

**Why `Lstat` and not `Stat`?** `Stat` follows symlinks. An attacker could place a symlink at `path` pointing outside the project root; `Stat` would describe the target file, while the subsequent open might race to different content. `Lstat` inspects the link itself. If it is a symlink, ForgeZero returns `ErrSymlink` immediately.

**Why `SameFile` after open?** This addresses classic **TOCTOU (time-of-check to time-of-use)** symlink substitution:

| Phase | Attacker action | Without `SameFile` | With `SameFile` |
|-------|-----------------|--------------------|-----------------|
| T0 | `path` is a regular file the auditor expects | Check passes | `Lstat` records inode A |
| T1 | Attacker replaces `path` with symlink to `/etc/passwd` | — | — |
| T2 | Reader calls `Open` | May read sensitive file | `Open` follows symlink; `Stat` on fd may differ |
| T3 | Compare identities | — | `os.SameFile(pre, post)` fails → `ErrPathChanged` |

`os.SameFile` compares device and inode (or equivalent on the platform). If the object opened is not the same file that was inspected at T0, the operation aborts. The error surface is `fs.ErrPathChanged` (distinct from `fs.ErrSymlink`).

`Rename` on Unix delegates to `renameAtomic`, which is currently `os.Rename` — on POSIX, rename within the same filesystem is atomic with respect to readers observing the destination path.

### 31.3 Windows Implementation and Build Tags

Native Windows support is not a runtime fork inside Unix code. It is a **separate compilation unit**:

| File | Build constraint | Purpose |
|------|------------------|---------|
| `unix.go`, `default_unix.go`, `rename_unix.go`, `impl_unix.go` | `!windows` | POSIX backend |
| `windows.go`, `default_windows.go`, `rename_windows.go`, `impl_windows.go` | `windows` | Win32 API backend |
| `pathnorm.go` | all platforms | `CleanPath`, `HasDrivePrefix`, `IsUNC`, `NormalizeAbs` |

The `Windows` struct implements the same `FileSystem` contract. Differences that matter for correctness:

- **Path normalization:** Every entry point passes paths through `CleanPath` before syscalls. Drive letters, backslashes, and UNC prefixes are normalized consistently before `Lstat` / `Open`.
- **`OpenVerified`:** Identical logical steps as Unix (`Lstat` → reject symlink → `Open` → `Stat` → `SameFile`), with `isSymlinkMode` abstracting mode-bit checks.
- **`Chmod`:** Best-effort on Windows (permission model differs); failures do not block writes that already succeeded.
- **`renameAtomic`:** Documented in [Section 34](#34-cross-platform-readiness).

`ImplName()` returns `"windows"` so CI logs and `fz doctor -json` output explicitly state which backend executed.

### 31.4 Path Normalization and UNC Handling

`pathnorm.go` provides shared helpers:

- `CleanPath` — trims, applies `filepath.Clean`, preserves `\\` UNC prefixes on Windows.
- `IsUNC` — detects `\\server\share` roots.
- `HasDrivePrefix` — detects `C:\` style paths.
- `NormalizeAbs` — resolves to absolute form; UNC paths skip redundant `Abs` behavior.

`utils.ValidateCLIPath` and `ResolveSecurePath` cooperate with these helpers so CLI-supplied paths cannot inject shell metacharacters or traverse above the execution root.

### 31.5 The `Mock` Implementation and Fault Injection

`fs.Mock` embeds a `Base FileSystem` (defaults to `fs.Default`) and a mutex-protected `failures` map keyed by `operation:path` or `operation:` (global op failure).

```go
m := fs.NewMock(fs.Default)
m.SetFailOp("Rename", fs.ErrDiskFull)
m.SetFail("OpenVerified", resolvedPath, fs.ErrPermission)
utils.SetFileSystem(m)
defer utils.SetFileSystem(nil)
```

Canonical injected errors (defined in `fs.go`):

| Variable | Simulated condition |
|----------|---------------------|
| `ErrDiskFull` | `ENOSPC` / quota exhaustion |
| `ErrPermission` | `EACCES` / `EPERM` |
| `ErrTimeout` | I/O timeout |
| `ErrInterrupted` | `EINTR` |
| `ErrSymlink` | Policy violation (not only mock — production path too) |
| `ErrPathChanged` | TOCTOU detection |

Tests in `internal/utils`, `internal/doctor`, `internal/fs`, and other packages call `SetFileSystem` to force `SecureWriteFile`, `ReadFileSecure`, and `scanTree` through failure branches without root privileges or filling disks.

### 31.6 Consumer Integration in `internal/utils`

Production code paths use:

| Function | VFS operations used |
|----------|---------------------|
| `SecureWriteFile` | `MkdirAll`, `CreateTemp`, `Chmod`, `Write`, `Close`, `Rename`, `Chmod` |
| `ReadFileSecure` | `ResolveSecurePath` → `OpenVerified` → read |
| `OpenVerifiedRead` | Same as above |
| `CopyFile` | `OpenVerified` source, temp file, `Rename` |
| `RemovePath` | `Remove` |
| `StatResolved` / `ReadDirResolved` | `Stat`, `ReadDir` on resolved paths |

`ResolveSecurePath` evaluates symlinks when present, but treats not-exist as a non-fatal condition for write destinations (parent directory creation). Reads of existing files always go through `OpenVerified` when integrity matters.

---

## 32. Aegis Security Core

> **Packages:** `internal/utils`, `internal/fs`, `internal/pkgman`, `internal/assembler`, `internal/linker`, `internal/zig`

v3.1.0 Aegis names the cross-cutting security properties that apply regardless of which assembler backend or linker mode is selected. They are not optional hardening flags; they are invariant behaviors of the binary.

### 32.1 Command Hardening: The `RunCommand` Pipeline

Every external process ForgeZero spawns — including but not limited to `git`, `ar`, `zig`, `fasm`, `gcc`, `g++`, `clang`, `ld`, `nasm`, `objdump`, `nm`, and `readelf` — must pass through `utils.RunCommand` (or a thin wrapper such as `pkgman.runGit` or `zig.RunCommand` that ultimately calls the same primitives).

**Stage 1 — Command name validation**

```go
if err := ValidateCLIArg(name); err != nil { return err }
```

Forbidden characters in arguments and paths are defined in `forbiddenArgChars()` / `forbiddenPathChars()` and include shell metacharacters, backticks, pipes, redirection symbols, and embedded NUL/newline bytes. This blocks injection via malicious config flags or crafted filenames passed through to tools.

**Stage 2 — Absolute executable resolution**

```go
resolved, err := lookExecutable(name)  // exec.LookPath on Unix; .exe/.bat search on Windows
cmd := exec.CommandContext(ctx, resolved, args...)
```

ForgeZero does not invoke bare names found in `PATH` without resolution. The actual executable path is fixed before `exec`. Combined with `ValidateCLIArg` on each argument, the command vector is fully expanded and inspectable when `-verbose` is enabled.

On Windows, `lookExecutable` probes `name`, `name.exe`, and `name.bat` in order.

**Stage 3 — Argument sanitization**

Each argument is validated with `ValidateCLIArg`, except the shell escape hatch: when the resolved basename is `sh` or `bash` and the invocation is `sh -c script`, the script body at index `1` is skipped (legacy compatibility for wrapped tools). All other arguments, including every `git` subcommand token and every linker flag, are validated.

**Stage 4 — Fixed environment**

```go
cmd.Env = deterministicEnv()
```

`deterministicEnv()` copies `os.Environ()` then forces:

| Variable | Value | Rationale |
|----------|-------|-----------|
| `LC_ALL` | `C` | Stable locale sorting and diagnostics |
| `LANG` | `C` | Same |
| `TZ` | `UTC` | Reproducible timestamps in tool output |
| `SOURCE_DATE_EPOCH` | `1600000000` | Aligns with reproducible build expectations |

Child processes do not inherit ambient locale randomness that could change compiler warning text or path behavior across machines.

**Stage 5 — Execution root**

When `utils.SetExecutionRoot` has been called (build, doctor, SBOM generation), `cmd.Dir` is set to that directory so relative paths in tool invocations resolve inside the project tree.

**Public entry points:**

| API | Use case |
|-----|----------|
| `RunCommand(ctx, verbose, stdout, stderr, name, args...)` | Full control of streams; captures output in internal buffer when streams nil |
| `RunCommandSilent(ctx, verbose, name, args...)` | Assembler, linker, zig backends |
| `RunCommandOutput(ctx, name, args...)` | Symbol extraction (`nm`, `objdump`) |

Injectability: `utils.CheckToolFunc` replaces toolchain presence checks in tests. `pkgman.runGit` is swappable for git failure simulation. `linker.runner` and `assembler.runCommand` provide additional seams for mock runners without bypassing validation.

### 32.2 Atomic Writes: `SecureWriteFile`

Non-atomic `os.WriteFile` is not used for security-sensitive outputs. `SecureWriteFile` implements write-via-temporary-file:

```
Validate path → SecureMkdirAll(parent)
    → resolveDest(path) [symlink-aware absolute path]
    → atomicWrite(resolved, data)
```

`atomicWrite` algorithm:

1. `CreateTemp(dir, ".fz_write_*.tmp")` in the destination directory (same volume for atomic rename).
2. `Chmod(tmpName, 0600)` — `utils.FilePerm` is `os.FileMode(0600)`.
3. Write full payload to the temporary descriptor.
4. `Close` the descriptor (buffers flushed to the kernel).
5. `renameResolved(tmpName, resolved)` → platform `Rename` (see Section 34 for Windows retries).
6. `Chmod(resolved, 0600)` on the final path.

On failure after temp creation, a deferred cleanup removes the partial temp file.

**Why this matters:**

| Risk | Non-atomic write | `SecureWriteFile` |
|------|------------------|-------------------|
| Crash mid-write | Truncated `.fz.yaml`, corrupt manifest | Readers see old file or nothing; never half-written final path |
| Concurrent reader | Partial JSON / YAML | Readers observe previous complete version until rename |
| Permission leak | umask-dependent final mode | Temp and final file explicitly `0600` |
| Symlink redirect | Write escapes project | `resolveDest` + verified read paths reject symlinks on read; write path resolved |

Files written through this path include: `.fz.yaml` updates from `fz pm`, `.fz.manifest`, `compile_commands.json`, SBOM outputs, `fz -init` templates, and the `fz doctor` probe file `.fz_doctor_probe`.

### 32.3 Constant-Time Toolchain Checksum Verification

`.fz.yaml` may specify expected BLAKE3 digests per tool binary:

```yaml
tool_checksums:
  gcc: "abc123..."
  nasm: "def456..."
```

During `utils.CheckTool(name)`:

1. Resolve tool with `lookExecutable`.
2. If no expectation configured, return success after presence check.
3. Otherwise `HashFile(path)` computes BLAKE3 of the on-disk executable.
4. Compare with `constantTimeEqual(actual, expected)` → `crypto/subtle.ConstantTimeCompare`.

**Threat model:** A local attacker who can influence timing of manifest verification should not learn expected hash bytes byte-by-byte through a standard string comparison that short-circuits on first differing octet. Constant-time comparison removes that side channel for equal-length hex strings.

Mismatch produces `tool checksum mismatch for <name>` and fails the build before any compilation.

### 32.4 Execution Root and Path Confinement

`SetExecutionRoot` / `GetExecutionRoot` store the canonical project directory. Functions such as `EnsureInsideRoot`, `HashDirWithRoot`, and doctor's `scanTree` use `ResolveSecurePath` to ensure scanned paths do not escape the root via symlinks or `..` segments after evaluation.

Symlink files encountered during directory walks for integrity or doctor are skipped (not followed) when policy requires containment.

### 32.5 Synergy with Supply-Chain Commands

| Command | Security mechanism |
|---------|-------------------|
| `fz pm add` / `update` | `runGit` → `RunCommand`; timeout via context |
| `fz verify` | `ReadFileSecure` on manifest; BLAKE3 over tree |
| `fz sbom` | `HashDirWithRoot`; `SecureWriteFile` if persisting |
| `fz audit` | `OpenVerifiedRead` per source file in doctor-style walks |
| `fz doctor` | `SecureWriteFile` probe; `OpenVerifiedRead` on every regular file |

---

## 33. System Self-Audit (`fz doctor`)

> **Package:** `internal/doctor` · **Entry point:** `fz doctor [options]`

`fz doctor` is a pre-flight diagnostic command. It does not compile code. It answers whether the current machine and project directory satisfy ForgeZero's minimum operational requirements before a long build or CI job is started.

### 33.1 Invocation

```bash
fz doctor
fz doctor -root /path/to/project
fz doctor -json
fz doctor -root ./myapp -json
```

| Flag | Default | Effect |
|------|---------|--------|
| `-root` | current working directory | Directory to audit for permissions and file readability |
| `-json` | off | Emit machine-readable `Report` JSON |

**Exit codes:**

| Code | Meaning |
|------|---------|
| `0` | `report.Healthy == true` |
| `1` | Degraded or failed audit (`Healthy == false`, or run error) |

### 33.2 Audit Pipeline (Four Stages)

`doctor.Run` executes stages in a fixed order:

```
SetExecutionRoot(root)
  → Stage A: auditToolchain()
  → Stage B: auditPermissions(root)
  → Stage C: Platform metadata (GOOS, GOARCH, ImplName, CPUs)
  → Stage D: healthyFromChecks() + status aggregation
```

#### Stage A — Toolchain Reachability

`auditToolchain` probes a minimal required set:

| Tool | Required when | Check method |
|------|---------------|--------------|
| `zig` | always (v3.1.0 policy) | `utils.LookExecutable` |
| `fasm` | non-Windows only | `LookExecutable` |
| `wasm-ld` | optional | `LookExecutable` |

Each result is a `ToolCheck` record: `name`, `required`, `found`, `path` (if found), `error` (if missing).

Missing **required** tools set `Healthy = false` and append `toolchain: <name> unavailable` to `Errors`.

#### Stage B — Recursive Permission Audit

`auditPermissions(root)` performs:

1. **Root resolution** — `ResolveSecurePath(root)`; failure records `resolve root` error.
2. **Root stat** — must be a directory.
3. **Writability probe** — writes `.fz_doctor_probe` via `SecureWriteFile`, deletes via `RemovePath`. Failure marks `Writable = false`.
4. **Tree scan** — iterative directory stack (not recursive syscalls) via `scanTree`:
   - Skips `.git`, `.fz_objs`, `.fz_cache`, `vendor` directories.
   - Counts directories (`DirsScanned`) and regular files (`FilesSeen`).
   - For each regular file: `Lstat` → skip symlinks → `OpenVerifiedRead` → close.

Any `OpenVerifiedRead` failure marks the tree **not readable** and records the path in `Permissions.Error`.

This stage validates that the same `OpenVerified` path used in production can read every source file under the project root.

#### Stage C — Platform Integrity

Populated at report construction:

| Field | Source |
|-------|--------|
| `platform.goos` | `runtime.GOOS` |
| `platform.goarch` | `runtime.GOARCH` |
| `platform.path_separator` | `os.PathSeparator` |
| `platform.filesystem_impl` | `fs.ImplName()` (`unix` / `windows`) |
| `platform.execution_root` | `utils.GetExecutionRoot()` after set |
| `platform.num_cpu` | `runtime.NumCPU()` |

#### Stage D — Health Aggregation

`healthyFromChecks()` sets `Healthy = false` if:

- Any required toolchain entry has `found == false`.
- `Permissions.Readable == false` OR `Permissions.Writable == false`.

`Status` is `"ok"` when healthy, `"degraded"` otherwise. Panics in permission audit are recovered and reported as `doctor panic: ...` without crashing the process silently.

### 33.3 Human-Readable Output Example

```
fz doctor: ok
platform: linux/amd64 fs=unix sep="/" root=/home/dev/myproject cpus=16
toolchain:
  zig (required): /usr/local/bin/zig
  fasm (required): /usr/bin/fasm
  wasm-ld: missing
permissions: root=/home/dev/myproject readable=true writable=true dirs=42 files=318
```

Degraded example:

```
fz doctor: degraded
platform: linux/amd64 fs=unix sep="/" root=/home/dev/myproject cpus=16
toolchain:
  zig (required): missing
  fasm (required): /usr/bin/fasm
  wasm-ld: missing
permissions: root=/home/dev/myproject readable=true writable=false dirs=10 files=0
  error: write probe: permission denied
issue: toolchain: zig unavailable
```

### 33.4 JSON Output Example

```bash
fz doctor -json
```

```json
{
  "status": "degraded",
  "healthy": false,
  "toolchain": [
    {
      "name": "zig",
      "required": true,
      "found": false,
      "error": "required tool not found in PATH: zig"
    },
    {
      "name": "fasm",
      "required": true,
      "found": true,
      "path": "/usr/bin/fasm"
    },
    {
      "name": "wasm-ld",
      "required": false,
      "found": false,
      "error": "required tool not found in PATH: wasm-ld"
    }
  ],
  "permissions": {
    "root": "/home/dev/myproject",
    "writable": true,
    "readable": true,
    "dirs_scanned": 42,
    "files_seen": 318
  },
  "platform": {
    "goos": "linux",
    "goarch": "amd64",
    "path_separator": "/",
    "filesystem_impl": "unix",
    "execution_root": "/home/dev/myproject",
    "num_cpu": 16
  },
  "errors": [
    "toolchain: zig unavailable"
  ]
}
```

JSON is suitable for CI gates:

```bash
fz doctor -json | jq -e '.healthy'
```

### 33.5 Relationship to `fz audit`

| Command | Purpose |
|---------|---------|
| `fz doctor` | Machine and workspace *operational* readiness (tools, permissions, platform) |
| `fz audit` | Source code *security* patterns (secrets, licenses, dangerous APIs) |

Run `fz doctor` before builds on a fresh runner; run `fz audit` before release tagging.

---

## 34. Cross-Platform Readiness

> **Build tags:** `windows` / `!windows` · **Packages:** `internal/fs`, `internal/utils`

v3.1.0 Aegis treats Windows as a compile-time target with its own filesystem adapter, not as an afterthought on the Unix code path.

### 34.1 Compile-Time Backend Selection

```
GOOS=linux   → package fs compiles: unix.go, default_unix.go, rename_unix.go, impl_unix.go
GOOS=windows → package fs compiles: windows.go, default_windows.go, rename_windows.go, impl_windows.go
GOOS=linux   → utils.IsWindows() == false
GOOS=windows → utils.IsWindows() == true (output naming, path validation)
```

There is no `if runtime.GOOS == "windows"` inside `OpenVerified` itself; the correct struct is selected by the Go toolchain. This keeps dead branches out of Linux binaries and allows Windows-only syscalls and retry policies without `#ifdef` clutter in shared files.

### 34.2 Windows Path Handling

`CleanPath` is applied at the boundary of every `Windows` method:

- Converts slash-mixed paths to OS-native separators where appropriate.
- Preserves UNC paths (`\\host\share\...`) without breaking `filepath.Clean` assumptions.
- Cooperates with `ValidateCLIPath` rejection of unsafe UNC forms (`isUnsafeUNC` in `internal/utils/security.go`).

Output binary naming in `builder` uses `utils.IsWindows()` to select `.exe` suffix when deriving default output names.

### 34.3 Atomic Rename on Windows

POSIX `rename(2)` replacing an existing destination is atomic. Windows may return sharing violations when antivirus, indexing, or another process holds a handle on the destination file.

`rename_windows.go` implements bounded retry:

```go
for attempt := 0; attempt < 8; attempt++ {
    if err := os.Rename(oldpath, newpath); err == nil {
        return nil
    }
    last = err
    time.Sleep(time.Millisecond * time.Duration(10*(attempt+1)))
}
return last
```

Backoff schedule: 10ms, 20ms, … up to 80ms between attempts. This integrates with `SecureWriteFile` so manifest updates survive transient locks on `compile_commands.json` or `.fz.yaml` on corporate Windows images.

Unix continues to use single-shot `os.Rename` (`rename_unix.go`).

### 34.4 `OpenVerified` Parity

Windows receives the same TOCTOU logic as Unix (see Section 31.2). Symlink rejection uses mode-bit inspection compatible with Windows reparse points where applicable.

### 34.5 Toolchain Notes on Windows

| Component | Native Windows status |
|-----------|----------------------|
| ForgeZero I/O (`internal/fs`) | Supported (v3.1.0) |
| NASM + GCC via MSYS2 MinGW | Supported (manual PATH setup) |
| `fz doctor` | Reports `filesystem_impl: "windows"` |
| `-sanitize` / `-strict` | Requires LLVM ASan build for Windows |
| WSL2 | Still recommended for simplest toolchain install |

`fz doctor` should be the first command run after installing `fz.exe` on a native Windows host to confirm PATH and directory permissions.

### 34.6 macOS and BSD

macOS builds use the `Unix` backend (`ImplName: "unix"`). Behavior matches Linux for `OpenVerified` and atomic rename. No separate Darwin struct is required because Go's `os` package maps to POSIX semantics on macOS.

---

## 35. Testing Standards (Aegis)

> **Policy version:** v3.1.0 · **Command:** `go test ./internal/... -cover`

### 35.1 Coverage Targets

The v3.1.0 release cycle raised statement coverage across `internal/` packages. The engineering standard for security-critical packages is **≥ 90%** statement coverage. Representative results from the Aegis test blitz (your tree may vary slightly by platform):

| Package | Coverage class | Notes |
|---------|----------------|-------|
| `internal/pkgman` | ≥ 90% | HTTP catalog mock, `runGit` injection, install/hash mismatch |
| `internal/fs` | ≥ 90% | `Mock` all ops, `OpenVerified`, pathnorm |
| `internal/doctor` | ≥ 90% | Permission failures, JSON, `scanTree` open errors |
| `internal/config` | ≥ 90% | `LoadMerged`, `Merge`, validation |
| `internal/zig` | ≥ 90% | `RunCommand` mock, link/compile failures |
| `internal/man` | 100% | Man page generator |
| `internal/assembler` | high 80s–90s | Mocked `runCommand`, all target triple branches |
| `internal/linker` | high 70s–80s | Windows impl tests, response file, symbol parsers |
| `internal/shell` | high 80s | Executor branches, `cmdBuild` paths |

Packages below 90% are dominated by platform-specific linker backends or optional external tools (`objdump` integration tests skipped when tool absent). Coverage gaps are tracked; new code in security paths must include tests.

### 35.2 Fault Injection via `fs.Mock`

Every error return defined in `internal/fs` is injectable:

```go
m := fs.NewMock(fs.Default)
m.SetFailOp("CreateTemp", fs.ErrDiskFull)
utils.SetFileSystem(m)
t.Cleanup(func() { utils.SetFileSystem(nil) })
err := utils.SecureWriteFile("out/config.yaml", data)
// expect error, no partial final file
```

Tests cover:

- **Disk full** on temp creation, write, or rename.
- **Permission denied** on `OpenVerified`, `MkdirAll`, `Remove`.
- **I/O timeout** on read paths.
- **Symlink** and **path changed** policy errors from `OpenVerified`.

This validates that higher-level commands (`fz doctor`, `fz verify`, `fz pm` config writes) degrade gracefully rather than panic or leave partial state.

### 35.3 Subprocess Mocking

| Seam | Package | Injected behavior |
|------|---------|-------------------|
| `runGit` | `pkgman` | Clone/checkout/pull failures without network |
| `RunCommand` | `zig` | Compile/link success and failure |
| `runner` | `linker` | Linker exit codes and stdout |
| `runCommand` | `assembler` | Assembler/compiler failures |
| `CheckToolFunc` | `utils` | Missing toolchain simulation |

### 35.4 HTTP and Catalog Tests

`pkgman` tests replace `httpClient` with `httptest` transports returning malformed JSON, HTTP 404, and truncated bodies to verify catalog fetch error aggregation without contacting external networks.

### 35.5 Race and Integration Commands

```bash
go test ./... -race
go test ./internal/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Parallel build races were addressed in v3.0.0; v3.1.0 extends race-safety expectations to VFS-backed I/O in doctor and verify paths.

### 35.6 Contributor Requirements

Pull requests that touch `internal/fs`, `internal/utils` (I/O or `RunCommand`), or `internal/doctor` must:

1. Include table-driven tests for new branches.
2. Use `fs.Mock` or existing seams for failure paths (no skipped `err` handlers).
3. Not regress package coverage below the 90% bar for the modified package.
4. Pass `go test ./...` locally before review.

---

## 36. Contributing

Contributions are welcome: bug reports, feature requests, documentation improvements, and code patches.

1. **Open an issue** before starting significant work to align on the approach.
2. **Fork the repository** and create a descriptive feature branch (`feature/watch-debounce`, `fix/nasm-elf32`).
3. **Write tests** for new behavior and ensure existing tests pass:

   ```bash
   go test ./...
   go test ./internal/... -cover
   ```

   Security-sensitive changes must include fault-injection or mock tests per [Section 35](#35-testing-standards-aegis).

4. **Submit a Pull Request** with a clear description of the change and the problem it solves.

Commit messages should be concise and use the imperative mood: *"Add JSON output mode"* not *"Added JSON output mode"*.

Repository: [github.com/forgezero-cli/ForgeZero](https://github.com/forgezero-cli/ForgeZero)

---

## 37. License

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
