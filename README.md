<p align="center">
  <a href="https://www.youtube.com/watch?v=0mbejIDcxsw">
    <img src="https://img.youtube.com/vi/0mbejIDcxsw/maxresdefault.jpg" alt="ForgeZero vs Ninja Benchmark" width="100%" style="border-radius: 8px;">
  </a>
  <br>
  <b>▶️ Click here to watch the full benchmark (4.5x faster) on YouTube</b>
</p>

# ForgeZero (fz) — Complete Documentation

<div align="center">
  <table style="border:none; background:transparent;">
    <tr>
      <td style="vertical-align:middle; padding-right:32px; border:none;">
        <img src="pictures/fz.jpg" alt="ForgeZero Logo" width="180" />
      </td>
      <td style="vertical-align:middle; border:none;">
        <h3 style="margin:0 0 8px 0;">ForgeZero 2.0 — zero-overhead build tool for assembly, C, and Gloria</h3>
        <p style="margin:0; color:#555;">One command. Any assembler. Any platform. Zero allocations.</p>
        <br/>
        <img src="https://img.shields.io/github/go-mod/go-version/forgezero-cli/ForgeZero" alt="Go Version"/>
        &nbsp;
        <img src="https://img.shields.io/github/license/forgezero-cli/ForgeZero" alt="License"/>
        &nbsp;
        <img src="https://img.shields.io/github/commits-since/forgezero-cli/ForgeZero/v5.0.0" alt="Commits"/>
      </td>
    </tr>
  </table>
</div>

> **Version:** 5.3.0 &nbsp;·&nbsp; **Language:** Go &nbsp;·&nbsp; **License:** GPLv3 &nbsp;·&nbsp; **Platform:** Linux · Windows · macOS

ForgeZero is a high-performance, zero-overhead build tool for assembly, C, C++, Objective-C, and Gloria developers. It wraps NASM, GAS, FASM, GCC, Clang, Zig, and LD into a single unified command-line interface — no Makefiles, no build scripts, no configuration required to get started.

> Inspired by the simplicity of **Suckless** and the efficiency of **TinyCC**

**Author:** [alexvoste](https://github.com/alexvoste)

*Non nobis, Domine, non nobis, sed nomini tuo da Gloriam*

---

## Performance: Full Scaling Benchmark

Benchmarks measured against standard `nasm -f elf64 && ld` and `make -j4` pipelines. Test environment: **Intel Core i5-10310U** (4C/8T, 1.7 GHz base), Arch Linux, Samsung 980 NVMe. Results are mean ± stddev over ≥10 runs via `hyperfine`.

| Modules | ForgeZero (`fz`) | Traditional (`make -j4`) | Speedup |
|---------|-------------------|--------------------------|---------|
| 20      | 19.3 ± 1.2 ms     | 45.4 ± 2.3 ms            | **2.35×** |
| 50      | 31.1 ± 1.3 ms     | 85.0 ± 2.1 ms            | **2.73×** |
| 100     | 57.0 ± 5.3 ms     | 185.5 ± 7.7 ms           | **3.25×** |
| 150     | 73.1 ± 4.3 ms     | 229.3 ± 3.6 ms           | **3.14×** |
| 200     | 82.2 ± 4.2 ms     | 291.1 ± 11.2 ms          | **3.54×** |
| 400     | 223.1 ± 9.8 ms    | 1105.0 ± 24.1 ms         | **4.95×** |

### Scaling Efficiency

| Metric | `fz` | `make -j4` |
|--------|-------|------------|
| Time growth (20→400 modules) | **+1056%** | **+2333%** |
| Overhead per module | ~0.36 ms | ~1.23 ms |
| I/O operations | **0 intermediate files** | 2× modules (`.o` read/write) |
| Process forks | **1** | ~2× modules + 1 |

> **Conclusion:** ForgeZero maintains **~3–5× speedup** at scale, growing to nearly **5× at 400 modules**. Traditional pipelines suffer from super-linear overhead due to process spawning, I/O contention, and CPU cache thrashing. ForgeZero's single-process design preserves cache locality across the entire build.

### Why the difference?

| Factor | Traditional (`make + nasm + ld`) | ForgeZero (`fz`) |
|--------|---------------------------------|-------------------|
| **Processes** | 400+ forks at scale | **1 process** (integrated pipeline) |
| **I/O** | Writes N intermediate `.o` files to disk | **Zero intermediate files** (in-memory) |
| **CPU Cache** | Cold start for every fork | **Hot cache** (code & data stay in L1/L2) |
| **Parallelism** | OS-level (`-j4`), high scheduling overhead | **Goroutines**, zero-cost concurrency |
| **Memory** | GC/Allocator overhead per process | **Zero-allocation hot path** (`0 allocs/op`, `0 B/op`) |
| **Allocations** | Unbounded heap churn | **Stack-buffered syscalls** in hot paths |

### Scaling Projection

| Modules | `fz` (est.) | `make -j4` (est.) | Speedup |
|---------|--------------|-------------------|---------|
| 20      | 19 ms        | 45 ms             | **2.35×** |
| 100     | 57 ms        | 185 ms            | **~3.2×** |
| 400     | 223 ms       | 1105 ms           | **~4.9×** |
| 1000    | ~530 ms      | ~3000+ ms         | **~5.5×** |

*Note: Projections beyond 400 modules assume continued sub-linear growth. Real-world results vary based on I/O and CPU contention.*

### How to reproduce

```bash
# Clone and build ForgeZero
git clone https://github.com/forgezero-cli/ForgeZero
cd ForgeZero
go build -o fz ./cmd/fz

# Run the benchmark script (generates N test modules and runs hyperfine)
./bench.sh  # Edit NUM_MODULES in script for different module counts

# Export benchmark results to Markdown
hyperfine --warmup 3 --prepare 'make clean && rm -rf .fz_objs fz_out' \
  './fz -dir . -out fz_out' 'make -j4' \
  --export-markdown results.md
```

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
8. [Build Profiles (`-profile` / `-p`)](#8-build-profiles--profile---p)
9. [Linking Modes](#9-linking-modes)
10. [C Compilation](#10-c-compilation)
    - 10.1 [Strict Warning Flags](#101-strict-warning-flags)
    - 10.2 [Sanitizers](#102-sanitizers)
11. [C++ Compilation](#11-c-compilation-1)
12. [Objective-C Compilation](#12-objective-c-compilation)
13. [Gloria Language](#13-gloria-language)
    - 13.1 [Overview](#131-overview)
    - 13.2 [Syntax Reference](#132-syntax-reference)
    - 13.3 [Control Flow](#133-control-flow)
    - 13.4 [Memory Access & I/O Ports](#134-memory-access--io-ports)
    - 13.5 [VGA Framebuffer Output](#135-vga-framebuffer-output)
    - 13.6 [Register Constants](#136-register-constants)
    - 13.7 [Compilation Pipeline](#137-compilation-pipeline)
14. [Cross-Compilation](#14-cross-compilation)
15. [Static Library Mode](#15-static-library-mode)
16. [Shared Library Mode](#16-shared-library-mode)
17. [Package Manager (fz pm)](#17-package-manager-fz-pm)
18. [Internal Mechanisms](#18-internal-mechanisms)
    - 18.1 [Build Cache (BLAKE3)](#181-build-cache-blake3)
    - 18.2 [Pre-link Symbol Check](#182-pre-link-symbol-check)
    - 18.3 [Watch Mode](#183-watch-mode)
    - 18.4 [JSON Output](#184-json-output)
    - 18.5 [Clean](#185-clean)
    - 18.6 [Parallel Builds](#186-parallel-builds)
    - 18.7 [Interactive Shell](#187-interactive-shell)
    - 18.8 [Virtual Filesystem Layer (VFS)](#188-virtual-filesystem-layer-vfs)
19. [Configuration File Reference](#19-configuration-file-reference)
    - 19.1 [Basic Fields](#191-basic-fields)
    - 19.2 [Multiple Source Directories](#192-multiple-source-directories)
    - 19.3 [Explicit Source File Lists](#193-explicit-source-file-lists)
    - 19.4 [Include & Exclude Patterns](#194-include--exclude-patterns)
    - 19.5 [Library Linking](#195-library-linking)
    - 19.6 [Custom Compiler & Linker Flags](#196-custom-compiler--linker-flags)
    - 19.7 [.fzignore File](#197-fzignore-file)
    - 19.8 [Full Annotated Example](#198-full-annotated-example)
20. [Assembler Backends](#20-assembler-backends)
    - 20.1 [NASM (.asm)](#201-nasm-asm)
    - 20.2 [GAS (.s / .S)](#202-gas-s--s)
    - 20.3 [FASM (.fasm)](#203-fasm-fasm)
21. [Zig Toolchain Backend](#21-zig-toolchain-backend)
22. [Supply Chain Security](#22-supply-chain-security)
    - 22.1 [SBOM Generation (fz sbom)](#221-sbom-generation-fz-sbom)
    - 22.2 [SAST Audit Scanner (fz audit)](#222-sast-audit-scanner-fz-audit)
23. [Reproducible Builds](#23-reproducible-builds)
24. [Source Tree Integrity (fz verify)](#24-source-tree-integrity-fz-verify)
25. [Build Profiler (fz bench)](#25-build-profiler-fz-bench)
26. [WebAssembly (WASM)](#26-webassembly-wasm)
27. [Project Initialization](#27-project-initialization)
28. [Contributor Guidance (fz contribute)](#28-contributor-guidance-fz-contribute)
29. [LSP & IDE Integration](#29-lsp--ide-integration)
30. [Self-Update](#30-self-update)
31. [Examples](#31-examples)
32. [Exit Codes](#32-exit-codes)
33. [Troubleshooting](#33-troubleshooting)
34. [Roadmap](#34-roadmap)
35. [Virtual Filesystem Layer (Aegis)](#35-virtual-filesystem-layer-aegis)
36. [Aegis Security Core](#36-aegis-security-core)
37. [System Self-Audit (`fz doctor`)](#37-system-self-audit-fz-doctor)
38. [Cross-Platform Readiness](#38-cross-platform-readiness)
39. [Testing Standards (Aegis)](#39-testing-standards-aegis)
40. [HADES Engine: Codegen & ELF Emission](#40-hades-engine-codegen--elf-emission)
41. [Contributing](#41-contributing)
42. [License](#42-license)

---

## 1. Overview

ForgeZero removes the friction between writing assembly (or C, C++, Objective-C, or Gloria) code and running it. Instead of managing assembler flags, linker invocations, and object file paths by hand, you point `fz` at a source file or directory and it handles everything:

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
- Compiles Objective-C (`.m`) via Clang/Zig with automatic framework linking.
- Compiles Gloria (`.glo`) directly to raw x86-64 ELF binaries with zero external dependencies.
- Manages external C/ASM dependencies via the built-in package manager (`fz pm`).
- Generates CycloneDX SBOMs with BLAKE3 hashes for supply chain transparency.
- Runs a built-in SAST scanner to detect secrets, license violations, and dangerous patterns.
- Guarantees byte-identical reproducible builds across machines.
- Verifies source tree integrity via BLAKE3 manifests.
- Profiles every build phase with nanosecond precision (`fz bench`).
- Compiles to WebAssembly via `wasm32-emscripten` and `wasm32-wasi`.
- Routes security-sensitive filesystem operations through a virtual layer with TOCTOU-safe `OpenVerified` reads (v3.1.0 Aegis).
- Runs `fz doctor` to verify toolchain presence, directory permissions, and platform integrity before builds.
- Achieves **zero heap allocations** on all linker hot-paths — `0 allocs/op`, `0 B/op` in micro-benchmarks (v4.1.0 Citadel).
- Uses **stack-buffered syscalls** (`openHot`, `unlinkHot`) for path conversion, bypassing `os.File` and `path/filepath` heap overhead.
- Emits **correct, deterministic ELF64 binaries** with local-before-global symbol table ordering and fixed absolute relocation offsets (HADES engine, v4.1.0).
- Applies **build profiles** (`-profile` / `-p`) to tune CPU core usage and optimization level per-session, persisted across runs (v4.7.0).
- Generates **`CONTRIBUTING_USER.md`** via `fz contribute` with environment diagnostics and contributor guidance (v4.7.0).

**What's new in v4.7.0:**

- **Build profiles (`-profile` / `-p`)** — three named profiles (`performance`, `balanced`, `power-saver`) that set GOMAXPROCS, `-j`, and the compiler optimization level in a single flag. The active profile persists between runs in `~/.config/fz/.profile.config`. Short form `-p` is equivalent to `-profile`.
- **Contributor guidance (`fz contribute`)** — generates `CONTRIBUTING_USER.md` in the current directory. The file includes environment diagnostics (via `fz doctor`), test and build recommendations, good-first-issue hints, and PR / commit guidelines.
- **Version output refinement** — `fz -v` prints the short version string (`4.7.0`); `fz --version` prints the full details block including platform, Go version, and build metadata.

**What's new in v4.6.0:**

- **Objective-C support** — `.m` files are now a first-class extension. Compilation is automatically delegated to the Clang or Zig backend. System libraries and Objective-C runtime frameworks are linked automatically with standardized diagnostic output. The full pipeline — from utility and parser recognition through intermediate assembly generation — is end-to-end. Verbose build output correctly displays each Objective-C compilation stage.

**What's new in v4.5.1:**

- **RISC-V + musl cross-compilation** — `riscv64-linux-musl` is now a validated static-target toolchain. Musl runtime components, linker argument generation, and supported architectures have extended validation and test coverage.
- **Gloria reliability** — improved error propagation in Gloria builtins; safer resource cleanup in runtime tests; explicit handling of walk and mapping errors.
- **Maintenance** — removed unused helpers and dead code; simplified internal implementations; modernized file operations to current Go APIs; improved lint compliance across the codebase.
- **Technical** — added musl linker argument validation tests; added runtime file verification tests; extended RISC-V target coverage; improved linker integration tests; refactored assembler, linker, builder, and plugin subsystems.

**What's new in v4.5.0-think:**

- **Embedded musl runtime** — musl runtime for Linux x86_64 is now embedded directly into the `fz` binary via `go:embed`. No external musl installation is required for static linking on the host architecture.
- **Update mechanism reliability** — improved self-update stability; linker and build system stability improvements.

**What's new in v4.4.0 (Gloria JIT):**

- **`while` loops** — conditional iteration based on a variable value. Supports `=`, `+=`, `-=` assignment and builtin calls inside the loop body.
- **`peek` / `poke`** — 16-bit memory read/write at arbitrary addresses. Both accept immediate integers or variable names.
- **I/O port primitives** — `in8(port)` reads a byte from an x86-64 I/O port (zero-extended); `out8(port, value)` writes a byte.
- **VGA framebuffer output** — `print()` in bare-metal mode writes directly to `0xB8000` with green text attribute `0x0A`. Register R15 is reserved as the VGA cursor offset and is preserved across function calls. Escape sequences `\n` and `\t` are resolved at compile time.
- **Register constants** — named constants for all 16 x86-64 general-purpose registers added for codegen clarity.
- **Extended register support** — `emitPushReg` and `emitPopReg` now support R8–R15.
- **Test infrastructure** — `patchVGA()` replaces the `0xB8000` constant with a heap-allocated fake VGA buffer; `dumpVGA()` renders the buffer to stdout using `syscall.Write` (zero allocations).

**What's new in v4.1.0 Citadel:**

- **Zero-allocation linker hot-path** — `copyFileHot`, `resolveSymbols`, and `emitRelocations` achieve `0 allocs/op` and `0 B/op` in all micro-benchmarks. GC pressure on the hot path is eliminated entirely. If you find an allocation in these paths, it is a bug.
- **Stack-buffered syscalls** — `openHot` and `unlinkHot` use fixed-size stack buffers for UTF-8 path conversion, removing all transient heap objects from the kernel boundary. Sustained copy throughput reaches ~1.18 GB/s on Intel i5-10310U hardware with zero GC interference.
- **HADES ELF emitter refactor** — local symbols now precede global symbols in `.symtab`, satisfying strict linker compliance. Absolute relocation offsets for `call` and `jmp` targets are calculated deterministically (verified via factorial execution test, exit code 120). Lexer/parser boundary checks prevent CPU instructions from being misidentified as user-defined labels.
- **Direct linker invocation (`-ld`)** — new flag bypasses compiler validation layers and invokes the linker directly, reducing orchestration overhead by ~3–5% on small projects.
- **Arena size override (`--arena-size=N`)** — overrides default arena capacity for plugin execution (advanced use).
- **Extended benchmark coverage** — 200-module and 400-module data points added; `4.95×` speedup confirmed at 400 modules.
- **Platform-specific syscall drivers** — conditional compilation via `//go:build linux`, `windows`, `darwin` build tags. Linux uses direct `SYS_OPENAT` / `SYS_UNLINKAT` via `golang.org/x/sys/unix`; Windows uses native `CreateFile` / `ReadFile` / `WriteFile` via `golang.org/x/sys/windows`; Darwin is stubbed with an isolated syscall layer.
- **Cross-architecture validation** — verified builds for `amd64` and `arm64` across all target OSes; no CGO dependencies, pure Go + static C linkage.
- **100% `golangci-lint` compliance** — strict configuration (`-E gofmt,govet,staticcheck,unused`), zero lint warnings in main branch.
- **Backward compatible** — v4.1.0 is fully compatible with v4.0.x configurations and CLI flags. No migration required.

**What's new in v3.1.0 Aegis:**

- **Virtual filesystem abstraction (`internal/fs`)** — all durable I/O routes through a `FileSystem` interface with Unix and native Windows implementations. `OpenVerified` closes the TOCTOU window between metadata check and read open via `Lstat` + `SameFile` identity verification.
- **Hardened subprocess execution** — external tools (`git`, `ar`, `zig`, `fasm`, `gcc`, `ld`, `nasm`, …) invoke exclusively through `utils.RunCommand`, which resolves binaries with `exec.LookPath`, validates every argument, and runs under a fixed, reproducibility-oriented environment.
- **Atomic secure writes** — manifests, configuration files, SBOM output, and doctor probe files use `SecureWriteFile`: mode `0600` temporary file, flush via close, platform-specific atomic rename (retry loop on Windows file locks).
- **Constant-time toolchain checksum comparison** — optional per-tool BLAKE3 expectations in config are verified with `crypto/subtle.ConstantTimeCompare` to reduce timing leakage during integrity audits.
- **`fz doctor`** — four-stage self-audit: toolchain reachability, recursive permission probe (read/write via `OpenVerified`), platform integrity (`GOOS`/`GOARCH`, VFS backend name, execution root, CPU count), and consolidated human or `--json` reporting.
- **Native Windows I/O path** — `//go:build windows` selects `fs.Windows`, `CleanPath` for drive/UNC normalization, and `renameAtomic` with bounded retries when AV software holds transient locks.
- **Fault-injection test suite** — `fs.Mock` injects `ErrDiskFull`, `ErrPermission`, `ErrTimeout`, and related errors; critical internal packages target **90%+** statement coverage.

**What's new in v3.0.0 GLORIA:**

- **Zig toolchain backend** — `zig cc` / `zig c++` as primary or alternative backend for C/C++. Zero external dependencies for cross-compilation: Zig ships all headers, libc, and sysroots internally.
- **SBOM (CycloneDX + BLAKE3)** — `fz sbom` generates a Software Bill of Materials in CycloneDX format with BLAKE3 hashes for every component.
- **SAST audit (`fz audit`)** — built-in security scanner: hardcoded secrets, license compliance (MPL/GPL detection), and dangerous C patterns.
- **Reproducible builds (`--reproducible`)** — suppression of build IDs, timestamps, and non-deterministic path references; object files sorted before linking for byte-identical output.
- **Source tree verification (`fz verify`)** — generates and checks BLAKE3 manifests of the entire source tree.
- **Symlink boundary protection** — every symlink is resolved and validated against the project root, blocking symlink race attacks.
- **`fz bench`** — nanosecond-precision build profiler. Multi-run averaging and JSON output supported.
- **Race-free parallel pipeline** — verified with `go test -race` across the complete test suite.
- **FASM improvements** — automatic `format ELF64` injection and correct `-dDEBUG=1` / debug symbol pass-through.
- **WebAssembly** — `wasm32-emscripten` and `wasm32-wasi` targets.

**What's new in v2.0.0 NEXUS:**

- **BLAKE3 hashing** — cache is up to 7× faster. 10 MB file hash time dropped from ~58 ms to ~8.7 ms.
- **Package manager (`fz pm`)** — manage external C/ASM dependencies from Git or the official catalog.
- **Shared library support** — `-shared`, `-cc-flag`, and `-ld-flag` flags for `.so`, `.dylib`, and `.dll` targets.
- **High test coverage** — utils 84%, linker 60%, assembler 60%, builder 56%.

**What's new in v1.9.0:**

- **Cross-compilation** — `-target <triple>` supports ARM, RISC-V, x86_64, and any standard GNU cross-compilation triple.
- **LSP support** — `-compile-commands` generates `compile_commands.json` for clangd, ccls, and LSP-aware editors.
- **Smart self-update** — `fz -update` backs up the old binary to `/usr/local/bin/fz.old` before replacing it.

**What's new in v1.8.0:**

- **Static libraries** — `-type static` and `-lib` build `.a` archives via `ar`.
- **Unique object file names** — multi-directory projects no longer produce colliding `.o` names.

**What's new in v1.7.0:**

- **Parallel builds** — `-j N` compiles all source files concurrently (0 = auto).
- **Interactive shell** — `fz -shell` opens a REPL for running `fz` commands without re-invoking the binary.
- **C++ support** — `.cpp`, `.cc`, `.cxx` compiled with `g++` or `clang++`.

**What's new in v1.6.0:**

- **Project initialization** — `fz -init` scaffolds `.fz.yaml`, `.fzignore`, and `README.md`.
- **Flat binary output** — `-format bin` for bootloaders, firmware, and embedded targets.

**What's new in v1.5.0:**

- **Multiple source directories** — `source_dirs` scanned in parallel.
- **Explicit source file lists** — `source_files` bypasses directory scanning entirely.
- **`.fzignore` file** — `.gitignore`-style exclusion rules for recursive scanning.
- **Multi-level config merging** — system, user, and project YAML configs merged in priority order.

ForgeZero is intentionally lightweight — a single statically compiled Go binary with no runtime dependencies beyond the standard assembler/compiler toolchain.

**What's new in v5.2.0:**

- **Config subsystem overhaul** — `internal/config` rewritten for deterministic, low-allocation merges: `LoadMerged` now respects `FZ_CONFIG_PATH`/`FZ_CONFIG` environment overrides, merges system→user→local config layers deterministically, and merges list/map fields while preserving user intent. `Config.Merge` was hardened to merge slices (deduped, ordered), maps (non-empty overrides), and ISO/custom-args properly.
- **Concurrent config load** — multiple config files discovered by `FindConfigs()` are loaded concurrently and merged deterministically to reduce I/O latency on large projects.
- **Inline Bash scripts** — configuration `scripts` entries may begin with `bash:` to include inline Bash code. `fz` prefers the system `bash`/`sh` when available; if not, a small internal runner executes common commands like `cd` and `export` and will fall back to executing binaries directly when available.
- **NASM fallback** — when a system `nasm` binary is not present, `fz` now falls back to the built-in NASM-like emitter/parser so `.asm` sources still assemble without an external dependency.
- **Improved validation & defaults** — `Validate()` and `fillDefaults()` now ensure safer defaults and clearer error messages for invalid modes, profiles, and isolation values.
- **Expanded tests** — new and updated unit tests for `internal/config` covering merge semantics, ISO handling, and invalid YAML handling.
- **Docs & examples** — README extended with configuration and testing quickstart guidance (you are reading it).

**Configuration Loading & Merge Semantics**

- Precedence: explicit path (CLI or `FZ_CONFIG_PATH`/`FZ_CONFIG`) → system config → user config → local project config. The effective configuration is the result of merging these layers in that order.
- Merge behavior: string scalars are overridden by later layers when non-empty; slices are concatenated with deduplication preserving earlier order; maps are merged with non-empty values overriding earlier entries; `source_file` clears directory lists and vice-versa to avoid ambiguous scan behavior.
- Useful environment variables:

```bash
export FZ_CONFIG_PATH=/etc/fz/project.yaml   # explicit config path takes precedence
export FZ_CONFIG=~/.fz.yaml                  # alternative env var supported
```

- API notes: `Load(path)` parses a single YAML file and runs `expand()` + `Validate()`; `LoadMerged()` handles discovery and merging. Use `DefaultConfigPath()` to find the local config file name used by `fz`.

**Testing & Contributing — Quickstart**

- Run package tests quickly:

```bash
# package under test (example)
go test ./internal/config

# run full test suite (may take longer)
go test ./...
```

- Run a specific test (example):

```bash
go test -run TestMerge ./internal/config
```

- Benchmarks and profiling:

```bash
go test -bench . -benchmem ./internal/...
```

- Contributing notes:
  - Follow `golangci-lint` style and run the test suite before opening PRs.
  - Use `fz contribute` to generate `CONTRIBUTING_USER.md` with environment diagnostics.


---

## 2. Requirements

### Assembler and compiler tools

| Source type              | Required tool          | Notes |
|--------------------------|------------------------|-------|
| `.asm`                   | `nasm`                 | x86/x86-64 Intel syntax |
| `.s` / `.S`              | `gcc` (drives `as`)    | AT&T syntax; `.S` files are C-preprocessed first |
| `.fasm`                  | `fasm`                 | Must be downloaded separately from flatassembler.net |
| `.c`                     | `gcc` or `clang`       | Strict flags + sanitizers by default |
| `.cpp` / `.cc` / `.cxx`  | `g++` or `clang++`     | Same strict flags as C; `clang++` preferred in strict mode |
| `.m`                     | `clang` or `zig cc`    | Objective-C; automatic framework linking (v4.6.0) |
| `.glo`                   | built-in (HADES)       | Gloria; compiles to raw x86-64 ELF, no external tools required |

### Linker tools

| Linker  | Required for |
|---------|--------------|
| `gcc`   | Default linking, C runtime support |
| `ld`    | Raw linking (`-mode raw`), linker scripts; direct invocation with `-ld` flag (v4.1.0) |
| `clang` | Strict sanitizer mode (`-strict`); Objective-C compilation |
| `ar`    | Static library mode (`-type static`) |

### Cross-compilation tools (optional)

When using `-target <triple>`, `fz` looks for prefixed toolchain binaries on your `PATH`:

| Target triple            | Expected compiler prefix     |
|--------------------------|------------------------------|
| `arm-linux-gnueabihf`    | `arm-linux-gnueabihf-gcc`    |
| `aarch64-linux-gnu`      | `aarch64-linux-gnu-gcc`      |
| `riscv64-linux-gnu`      | `riscv64-linux-gnu-gcc`      |
| `riscv64-linux-musl`     | `riscv64-linux-musl-gcc` or `-zig` (v4.5.1) |
| `x86_64-linux-gnu`       | `x86_64-linux-gnu-gcc`       |

Install cross-compilers via your package manager (e.g. `sudo apt install gcc-arm-linux-gnueabihf`).

When using `-zig`, no prefixed toolchain is required — Zig resolves the target internally. See [Section 21](#21-zig-toolchain-backend).

### Optional tools (used internally)

| Tool       | Purpose |
|------------|---------|
| `nm`       | Pre-link duplicate symbol check (primary) |
| `objdump`  | Fallback for symbol check |
| `readelf`  | Second fallback for symbol check |
| `git`      | Required for `fz pm add` |
| `zig`      | Required for `-zig` backend (v3.0.0+) |
| `emcc`     | Required for `wasm32-emscripten` target (v3.0.0+) |
| `hyperfine`| Benchmarking (`fz bench` and `bench.sh`) — optional but recommended |

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

**Install Clang (required for Objective-C; optional for `-strict` mode):**

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

> **Note:** macOS ships `clang` as the system compiler under the `gcc` alias via Xcode Command Line Tools. `brew install gcc` installs it as `gcc-14` (or similar). ForgeZero uses whatever `gcc` resolves to on your `PATH`. The Darwin syscall layer is currently stubbed — builds succeed but runtime behavior is untested. Objective-C compilation on macOS requires Xcode Command Line Tools (`xcode-select --install`) for the full system framework headers.

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

As of **v3.1.0 Aegis**, ForgeZero ships a first-class native Windows filesystem backend (`internal/fs.Windows`) and Windows-specific atomic rename retry logic. As of **v4.1.0 Citadel**, platform-specific syscall drivers are implemented via `golang.org/x/sys/windows` (`CreateFile`, `ReadFile`, `WriteFile`), making native Windows builds a fully supported compilation target.

WSL2 remains the path of least resistance for full toolchain parity on day-to-day development, but native Windows I/O no longer depends on a Linux compatibility shim.

#### Option A — WSL2 (Recommended)

1. Open **PowerShell as Administrator** and run:

```powershell
wsl --install
```

2. Restart your machine. Open **Ubuntu** from the Start menu.

3. Inside the WSL2 terminal, follow the [Debian/Ubuntu instructions](#31-linux--debian--ubuntu).

4. Access your Windows files from WSL2 at `/mnt/c/Users/<YourName>/`.

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

> **Known limitation:** `-sanitize` and `-strict` require Clang with AddressSanitizer support compiled for Windows, available via the official LLVM Windows release but requiring additional setup beyond MSYS2. Basic NASM assembly and GCC linking work without any extra configuration.

Run `fz doctor` as the first command after installation to confirm PATH and directory permissions are correctly configured.

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

The binary lands in `$GOPATH/bin`. Verify:

```bash
fz -v        # prints: 4.7.0
fz --version # prints: full details block
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

**Compile an Objective-C file:**

```bash
fz -cc main.m
./main
```

**Compile a Gloria file:**

```bash
fz -gloria main.glo
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

**Build with a profile:**

```bash
fz -p performance -cc main.c
fz -profile power-saver -dir ./src
```

**Build with cross-compilation:**

```bash
fz -cc main.c -target arm-linux-gnueabihf
```

**Build with Zig backend (no extra toolchain needed):**

```bash
fz -cc main.c -zig -target aarch64-linux-musl
```

**Direct linker invocation (v4.1.0 — bypasses compiler validation):**

```bash
fz -asm boot.asm -ld -out boot.elf
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

**Generate contributor guidance:**

```bash
fz contribute
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

| Extension               | Language      | Backend                    | Notes |
|-------------------------|---------------|----------------------------|-------|
| `.asm`                  | Assembly      | NASM                       | x86/x86-64, Intel syntax, ELF64 |
| `.s`                    | Assembly      | GAS via `gcc -c`           | AT&T syntax |
| `.S`                    | Assembly      | GAS via `gcc -c`           | AT&T syntax + C preprocessor |
| `.fasm`                 | Assembly      | FASM                       | Separate install; auto `format ELF64` injection (v3.0.0) |
| `.c`                    | C             | GCC, Clang, or `zig cc`    | Strict flags + sanitizers by default |
| `.cpp` / `.cc` / `.cxx` | C++           | G++, Clang++, or `zig c++` | Same strict flags as C (v1.7.0+) |
| `.m`                    | Objective-C   | Clang or `zig cc`          | Auto framework linking; Clang/Zig backend required (v4.6.0) |
| `.glo`                  | Gloria        | Built-in (HADES)           | Compiles to raw x86-64 ELF; 69–125 byte output |

All other extensions are silently ignored during directory and recursive scanning.

---

## 6. Build Modes

### 6.1 Single File Mode

Compiles and links a single source file into a binary.

```bash
fz -asm program.asm
fz -cc main.c
fz -cc main.cpp
fz -cc main.m
fz -gloria main.glo
```

- Output binary name is derived from the source filename (`program.asm` → `program`).
- A single object file is created and removed after linking unless `-keep-obj` is set.
- Override the binary name with `-out` and the object file name with `-out-obj`.

### 6.2 Directory Mode

Recursively scans a directory for all supported source files, compiles each to a uniquely named object file, then links everything into a single binary.

```bash
fz -dir ./src
```

**Object file naming** — names are derived from the full relative path to prevent collisions across subdirectories:

| Source file          | Object file           |
|----------------------|-----------------------|
| `src/hello.asm`      | `src_hello_asm.o`     |
| `src/hello.s`        | `src_hello_s.o`       |
| `src/sub/hello.asm`  | `src_sub_hello_asm.o` |
| `lib/hello.asm`      | `lib_hello_asm.o`     |

Object files live in `.fz_objs/` and are removed after linking unless `-keep-obj` is passed.

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

**Config merging (v1.5.0):** Three-level merge in priority order:

1. System-level: `/etc/fz/fz.yaml`
2. User-level: `~/.config/fz/fz.yaml`
3. Project-level: `.fz.yaml` in the working directory
4. CLI flags (highest priority)

---

## 7. CLI Reference

### Synopsis

```
fz [options]
```

At least one of `-asm`, `-cc`, `-gloria`, `-dir`, `-init`, `-shell`, `-contribute`, `pm`, `sbom`, `audit`, `verify`, `bench`, `doctor`, or a valid config file must be present.

### Full Flag Reference

| Flag | Argument | Default | Description |
|------|----------|---------|-------------|
| `-asm` | `<file>` | — | Assemble the given assembly source file. |
| `-cc` | `<file>` | — | Compile the given C, C++, or Objective-C source file. |
| `-gloria` | `<file>` | — | Compile the given Gloria (`.glo`) source file via the built-in HADES engine. |
| `-dir` | `<dir>` | — | Recursively build all supported files in the directory. |
| `-out` | `<name>` | Derived from source | Name of the output binary. |
| `-out-obj` | `<name>` | `<basename>.o` | Object file name (single-file mode only). |
| `-profile`, `-p` | `performance\|balanced\|power-saver` | `balanced` | Apply a named build profile. Persisted to `~/.config/fz/.profile.config`. (v4.7.0) |
| `-mode` | `auto\|c\|raw` | `auto` | Linking mode. See [Section 9](#9-linking-modes). |
| `-ld` | — | off | Invoke the linker directly, bypassing compiler validation layers. Reduces overhead ~3–5% on small projects. (v4.1.0) |
| `-format` | `elf32\|elf64\|bin` | `elf64` | Output format for assembled binaries. |
| `-target` | `<triple>` | — | Cross-compilation target triple (e.g. `arm-linux-gnueabihf`, `wasm32-wasi`). |
| `-zig` | — | off | Use Zig (`zig cc` / `zig c++`) as the compiler backend. |
| `--reproducible` | — | off | Enable deterministic builds: suppress build IDs, timestamps, path references; sort objects. |
| `--arena-size=N` | `<bytes>` | auto | Override default arena capacity for plugin execution. Advanced use only. (v4.1.0) |
| `-type` | `executable\|static` | `executable` | Output type: linked binary or static library (`.a`). |
| `-lib` | `<name>` | — | Output library name when `-type static` is used (without `lib` prefix or `.a` suffix). |
| `-shared` | — | off | Build a shared library (`.so` / `.dylib` / `.dll`). |
| `-cc-flag` | `<flags>` | — | Extra compiler flags, space-separated, injected after standard flags. |
| `-ld-flag` | `<flags>` | — | Extra linker flags, space-separated, appended to the linker command. |
| `-j` | `<N>` | Profile-dependent | Parallel compilation jobs. `0` = auto (number of CPU cores). Overridden by `-profile`. |
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
| `-v` | — | — | Print short version string and exit (`4.7.0`). |
| `--version` | — | — | Print full version details (platform, Go version, build metadata) and exit. |

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
| `fz audit` | Run the built-in SAST scanner: secrets, license compliance, and dangerous patterns. |
| `fz audit -dir <dir>` | Audit a specific directory. |
| `fz audit -json` | Emit the audit report as JSON to stdout. |
| `fz verify` | Verify the current source tree against the stored BLAKE3 manifest. |
| `fz verify --generate` | Generate a new BLAKE3 manifest of the current source tree. |
| `fz verify --strict` | Also report UNTRACKED files not present in the manifest. |
| `fz verify -manifest <f>` | Use a specific manifest file instead of the default `.fz.manifest`. |
| `fz doctor` | Run the Aegis self-audit: toolchain, permissions, platform integrity. |
| `fz doctor -root <dir>` | Audit a specific project root (default: current working directory). |
| `fz doctor -json` | Emit the audit report as JSON; exit code `1` if `healthy` is false. |

### Performance Sub-commands

| Command | Description |
|---------|-------------|
| `fz bench` | Profile the build: nanosecond-precision timing for every phase. |
| `fz bench -n <N>` | Run N times; report average and standard deviation per phase. |
| `fz bench -json` | Emit the benchmark report as JSON. |

### Contributor Sub-commands

| Command | Description |
|---------|-------------|
| `fz contribute` | Generate `CONTRIBUTING_USER.md` with environment diagnostics and contributor guidance. |

---

## 8. Build Profiles (`-profile` / `-p`)

> **New in v4.7.0**

Build profiles are named presets that configure CPU core usage, parallel job count, and compiler optimization level in a single flag. The active profile is persisted between runs in `~/.config/fz/.profile.config` and applied automatically on subsequent invocations unless overridden.

### Profile Table

| Profile | Cores (`-j`) | Optimization | GOMAXPROCS | Use case |
|---------|-------------|-------------|------------|----------|
| `performance` | All cores | `-O3` | All cores | Maximum build speed; CI, release builds |
| `balanced` | Half cores | `-O2` | Half cores | Default; daily development |
| `power-saver` | 1 | `-Os` | 1 | Battery-constrained environments; minimal heat |

### Usage

```bash
# Short form
fz -p performance -cc main.c
fz -p power-saver -dir ./src

# Long form
fz -profile performance -cc main.c
fz -profile balanced -dir ./src
fz -profile power-saver -dir ./src -verbose
```

### Persistence

The active profile is written to `~/.config/fz/.profile.config` on each invocation. Subsequent calls to `fz` without `-profile` or `-p` will use the last persisted profile. To reset to the default (`balanced`), either pass `-profile balanced` explicitly or delete the config file:

```bash
rm ~/.config/fz/.profile.config
```

### Interaction with other flags

Flags that directly override profile-managed settings take precedence over the profile:

```bash
# Profile sets -j to all cores, but -j 2 overrides it
fz -p performance -dir ./src -j 2
```

`GOMAXPROCS` is set internally by `fz` for the duration of the build process. It does not affect the calling shell environment.

### Profile in configuration file

```yaml
# .fz.yaml
profile: performance
```

CLI `-profile` / `-p` takes precedence over the config file value.

---

## 9. Linking Modes

The `-mode` flag controls how compiled object files are linked into a final binary.

### `auto` (default)

ForgeZero tries linkers in sequence until one succeeds:

1. `gcc` — standard linking with libc and C runtime.
2. `gcc -no-pie` — position-dependent executable; needed when code assumes fixed load addresses.
3. `ld` — raw system linker; last resort.

When `-strict` is active, `clang` with full sanitizer flags is tried first.

### `c` — Force GCC / Clang

```bash
fz -asm program.asm -mode c
fz -cc main.c -mode c
```

Always links using `gcc` (or `clang` in strict mode). Required when code calls libc functions (`printf`, `malloc`, `exit`, etc.) or depends on C runtime initialization.

### `raw` — Force LD

```bash
fz -asm kernel.asm -mode raw -out kernel.bin
```

Bypasses GCC entirely and invokes `ld` directly. Suitable for:

- OS kernels and bootloaders
- Bare-metal firmware
- Programs that define their own `_start` and use raw syscalls
- Embedded targets requiring full control over binary layout

> **Warning:** Raw-linked binaries cannot reference any libc symbol.

### Direct Linker Invocation (`-ld`, v4.1.0)

```bash
fz -asm boot.asm -ld -out boot.elf
```

The `-ld` flag invokes the linker directly, bypassing compiler validation layers entirely. This reduces orchestration overhead by ~3–5% on small projects. Unlike `-mode raw`, `-ld` does not imply a particular linking strategy — it simply removes the `gcc`/`clang` wrapper from the invocation chain. Use this when you need the tightest possible control over linker arguments and accept full responsibility for the link step.

---

## 10. C Compilation

### 10.1 Strict Warning Flags

Every `.c` file compiled by `fz` receives these flags unconditionally:

```
-Wall -Wextra -Werror -Wpedantic -Wshadow -Wconversion
```

| Flag | Effect |
|------|--------|
| `-Wall` | Enables most common warnings |
| `-Wextra` | Enables additional warnings beyond `-Wall` |
| `-Werror` | Promotes all warnings to errors |
| `-Wpedantic` | Enforces strict ISO C compliance |
| `-Wshadow` | Warns when a local variable shadows an outer one |
| `-Wconversion` | Warns on implicit type conversions that may lose precision |

Any warning is treated as an error and stops the build immediately. This is intentional — ForgeZero enforces clean, portable C code by default.

### 10.2 Sanitizers

**Standard mode (default — always enabled unless `-sanitize=false`):**

```
-fsanitize=address
-fsanitize=undefined
```

**Strict mode (`-strict`):**

```
-fsanitize=address
-fsanitize=undefined
-fsanitize-address-use-after-return=always    (Clang only)
-fsanitize-address-use-after-scope
```

**Disable sanitizers (release build / benchmarking):**

```bash
fz -cc main.c -sanitize=false
```

> **Note:** Sanitizers are automatically disabled for WebAssembly targets (`wasm32-*`). See [Section 26](#26-webassembly-wasm).

---

## 11. C++ Compilation

Added in **v1.7.0**. ForgeZero compiles `.cpp`, `.cc`, and `.cxx` files using `g++` or `clang++` (or `zig c++` when `-zig` is active). The same strict warning flags applied to C are applied identically to C++:

```
-Wall -Wextra -Werror -Wpedantic -Wshadow -Wconversion
```

Sanitizers are also enabled by default for C++ in the same way as C.

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

`fz` dispatches `.c` files to `gcc`/`clang`/`zig cc` and `.cpp`/`.cc`/`.cxx` files to `g++`/`clang++`/`zig c++` automatically. All objects are linked in a single step.

---

## 12. Objective-C Compilation

> **New in v4.6.0**

ForgeZero supports Objective-C (`.m`) source files as a first-class extension. No additional configuration is required — the build pipeline detects `.m` files and delegates compilation to Clang or the Zig backend automatically.

### Compilation pipeline

1. Source recognition — `.m` files are identified in utility and parser layers alongside `.c` and `.cpp`.
2. Backend selection — Clang is selected by default; Zig is used when `-zig` is active.
3. Compilation — the file is compiled with `-x objective-c` and the appropriate standard flags.
4. Auto-linking — the Objective-C runtime library (`-lobjc`) and any required system frameworks are linked automatically. Diagnostic output from the framework linker is normalized to ForgeZero's standard format.

### Usage

```bash
# Single Objective-C file
fz -cc main.m
./main

# With verbose output (displays Objective-C compilation stages)
fz -cc main.m -verbose

# Directory with mixed sources
fz -dir ./src

# With Zig backend
fz -cc main.m -zig
```

### Requirements

- **Clang** must be on `PATH`. On macOS, the Xcode Command Line Tools provide a full-featured Clang with all system framework headers. On Linux, `sudo apt install clang` is sufficient for most Objective-C use cases that do not require macOS-specific frameworks (Foundation, AppKit, etc.).
- When using `-zig`, Zig's bundled Clang frontend handles `.m` files; no separate Clang installation is needed.

> **Note:** Objective-C framework headers specific to macOS (Foundation, UIKit, AppKit, CoreData, etc.) are not available on Linux. Cross-platform Objective-C code using only the GNU Objective-C runtime compiles on Linux without issue.

---

## 13. Gloria Language

> **Initial release: v3.0.0 GLORIA** · **JIT features: v4.4.0** · **Reliability improvements: v4.5.1**

### 13.1 Overview

Gloria is ForgeZero's integrated systems programming language. It compiles directly to raw x86-64 ELF binaries via the HADES engine with no external assembler, compiler, or linker required. Gloria is designed for bare-metal targets, OS kernels, firmware, and other environments where binary size and toolchain footprint matter.

Key properties:

- Go/Rust-like syntax with a minimal feature set.
- Compiled output: **69–125 bytes** for typical programs.
- No runtime dependencies — the output is a self-contained ELF64 binary.
- Bare-metal VGA framebuffer output via `print()` when compiled in kernel mode.
- x86-64 I/O port access via `in8()` / `out8()`.
- Arbitrary memory read/write via `peek()` / `poke()`.

### 13.2 Syntax Reference

```go
fn main() {
    let a = 10
    let b = 20
    let c = a + b
    print("result computed")
}
```

**Supported constructs:**

| Construct | Example |
|-----------|---------|
| Variable declaration | `let x = 42` |
| Arithmetic | `let y = x + 1`, `x -= 5`, `x *= 2` |
| Function definition | `fn add(a, b) { ... }` |
| Function call | `let r = add(1, 2)` |
| Conditional | `if x { ... }` |
| Loop | `while x { ... }` |
| Print | `print("hello")` |
| Memory read | `let v = peek(0xB8000)` |
| Memory write | `poke(0xB8000, 0x0741)` |
| I/O port read | `let v = in8(0x60)` |
| I/O port write | `out8(0x3F8, 65)` |

### 13.3 Control Flow

**`if` statement:**

Evaluates a variable. If the variable is non-zero, the body executes.

```go
let x = 1
if x {
    print("x is set")
}
```

**`while` loop** (v4.4.0):

Iterates while the condition variable is non-zero. Supports `=`, `+=`, and `-=` assignment and builtin calls inside the loop body.

```go
let i = 5
while i {
    print("counting")
    i -= 1
}
```

### 13.4 Memory Access & I/O Ports

> **New in v4.4.0**

**`peek(address)`** — reads a 16-bit value from the specified memory address. The address may be an immediate integer literal or a variable.

```go
let val = peek(0xB8000)
let addr = 0x1000
let x = peek(addr)
```

**`poke(address, value)`** — writes a 16-bit value to the specified memory address. Both arguments may be immediates or variables.

```go
poke(0xB8000, 0x0741)   // write 'A' with white-on-black to VGA
let pos = 2
poke(pos, 0x0742)
```

**`in8(port)`** — reads one byte from an x86-64 I/O port. The return value is zero-extended to 64 bits.

```go
let scancode = in8(0x60)
```

**`out8(port, value)`** — writes one byte to an x86-64 I/O port.

```go
out8(0x3F8, 65)   // transmit 'A' on COM1
```

### 13.5 VGA Framebuffer Output

> **New in v4.4.0**

When Gloria is compiled in bare-metal (kernel) mode, `print()` writes directly to the VGA text-mode framebuffer at `0xB8000` using green text (`attribute byte 0x0A`).

**Behavior:**

- Register **R15** is reserved as the VGA cursor offset and is preserved across all function calls.
- Escape sequences `\n` and `\t` in string literals are parsed and converted to the corresponding control characters at compile time.
- `emitLowLevelPrint` dispatches to the VGA path when `kernelMode` is set, or to the `sys_write` syscall path in userspace mode.

```go
fn main() {
    print("Booting...\n")
    print("OK\t[done]")
}
```

In userspace mode (default), `print()` uses Linux `sys_write` (fd 1). The kernel/userspace dispatch is automatic based on the compilation target.

### 13.6 Register Constants

> **New in v4.4.0**

Named constants for all 16 x86-64 general-purpose registers are available to Gloria programs and are used internally by the HADES codegen layer for clarity and correctness:

| Constant | Register | Encoding |
|----------|----------|----------|
| `regRAX` | RAX | 0 |
| `regRCX` | RCX | 1 |
| `regRDX` | RDX | 2 |
| `regRBX` | RBX | 3 |
| `regRSP` | RSP | 4 |
| `regRBP` | RBP | 5 |
| `regRSI` | RSI | 6 |
| `regRDI` | RDI | 7 |
| `regR8`  | R8  | 8 |
| `regR9`  | R9  | 9 |
| `regR10` | R10 | 10 |
| `regR11` | R11 | 11 |
| `regR12` | R12 | 12 |
| `regR13` | R13 | 13 |
| `regR14` | R14 | 14 |
| `regR15` | R15 | 15 |

R15 is reserved by the Gloria runtime as the VGA cursor register in bare-metal mode and must not be used as a general-purpose variable in kernel Gloria programs.

### 13.7 Compilation Pipeline

```bash
fz -gloria main.glo
./main

fz -gloria main.glo -verbose
fz -gloria main.glo -out kernel.elf
```

**Pipeline stages:**

1. **Lexer** — tokenizes `.glo` source; CPU instruction mnemonics are disambiguated from user labels at the lexer/parser boundary (v4.1.0 HADES).
2. **Parser** — builds an AST; malformed input triggers `ErrMalformedAST` before codegen begins; no silent fallback degradation.
3. **Codegen (HADES)** — emits x86-64 machine code from the AST using zero-allocation stack buffers.
4. **ELF emission** — writes a valid ELF64 binary with correct local-before-global `.symtab` ordering and deterministic relocation offsets.

The entire pipeline runs in a single process with zero intermediate files on disk. Output binary size is typically 69–125 bytes.

---

## 14. Cross-Compilation

Added in **v1.9.0**, extended with Zig backend in **v3.0.0**, validated for `amd64` and `arm64` in **v4.1.0**, extended to `riscv64-linux-musl` in **v4.5.1**.

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

Without `-zig`, `fz` constructs prefixed compiler and linker names by prepending the triple:

- Compiler: `<triple>-gcc`
- C++ compiler: `<triple>-g++`
- Linker: `<triple>-gcc` or `<triple>-ld` depending on linking mode
- Archiver: `<triple>-ar` (when `-type static`)

With `-zig`, the target triple is passed directly to `zig cc -target`. No prefixed binary lookup happens. Zig ships all libc variants (musl, glibc, WASI sysroot) internally, making it the recommended backend for static musl targets including `riscv64-linux-musl`.

### Installing Cross-Compilers (without Zig)

```bash
# Debian / Ubuntu
sudo apt install gcc-arm-linux-gnueabihf
sudo apt install gcc-aarch64-linux-gnu
sudo apt install gcc-riscv64-linux-gnu

# Fedora
sudo dnf install gcc-arm-linux-gnu
sudo dnf install gcc-aarch64-linux-gnu

# Arch Linux
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

---

## 15. Static Library Mode

Added in **v1.8.0**. ForgeZero can produce static libraries (`.a` archives) instead of linked executables.

```bash
fz -dir ./src -type static -lib mylib
```

This compiles all source files in `./src/` into object files, then archives them into `libmylib.a` using `ar`.

| Flag | Description |
|------|-------------|
| `-type static` | Build a static library instead of an executable |
| `-lib <name>` | Name of the library (without `lib` prefix and `.a` suffix) |

The output file is always named `lib<name>.a`.

**Config file:**

```yaml
source_dirs:
  - src
type: static
lib: mylib
```

**Cross-compilation with static library:**

```bash
fz -dir ./src -type static -lib mylib -target arm-linux-gnueabihf
fz -dir ./src -type static -lib mylib -zig -target aarch64-linux-musl
```

---

## 16. Shared Library Mode

Added in **v2.0.0 NEXUS**.

```bash
fz -cc mylib.c -shared -o libmylib.so
fz -cc mylib.c -shared -cc-flag "-O2 -fPIC" -ld-flag "-pthread" -o libmylib.so
```

| Flag | Description |
|------|-------------|
| `-shared` | Build a shared library instead of an executable |
| `-cc-flag "<flags>"` | Extra compiler flags injected after standard flags |
| `-ld-flag "<flags>"` | Extra linker flags appended to the linker command |

When building shared libraries for Linux, always include `-fPIC` in `-cc-flag`.

---

## 17. Package Manager (fz pm)

Added in **v2.0.0 NEXUS**. `fz pm` manages external C/ASM dependencies from Git repositories or the official ForgeZero package catalog.

### Sub-commands

#### fz pm add

```bash
fz pm add github.com/me/my-lib
fz pm add github.com/me/my-lib@v1.2.3
```

Clones the repository into `vendor/`, checks out the version tag if given, and updates `.fz.yaml` (`source_dirs`). All subsequent `fz` builds include the vendored package.

#### fz pm remove

```bash
fz pm remove my-lib
```

Deletes the package directory from `vendor/`, cleans up `.fz.yaml`, and removes empty parent directories.

#### fz pm list

```bash
fz pm list
```

Lists all currently installed packages and their versions.

#### fz pm update

```bash
fz pm update
```

Pulls the latest changes for all installed packages.

#### fz pm catalog

```bash
fz pm catalog
```

Fetches and displays the full list of packages from the official ForgeZero catalog (`https://raw.githubusercontent.com/forgezero-cli/catalog/main/catalog.json`).

#### fz pm search

```bash
fz pm search iot
fz pm search crypto
```

Searches the official catalog by name or keyword.

#### fz pm install

```bash
fz pm install esp-idf
```

Installs a named package from the catalog with BLAKE3 hash verification.

**How it works:** packages are cloned into `vendor/<package-name>/`. `.fz.yaml` is updated automatically. All network and git operations run with configurable timeouts (default: 60 s, override with `-timeout`).

---

## 18. Internal Mechanisms

### 18.1 Build Cache (BLAKE3)

ForgeZero caches compiled object files in `.fz_cache/` to skip recompilation of unchanged sources.

**Cache key** computed from:

- BLAKE3 hash of the source file contents
- `-debug` flag state
- `-mode` value
- `-target` value
- active build profile

ForgeZero now supports `cache_mode` in project config. Valid values:

- `disk` — default on-disk caching in `.fz_cache/`
- `ram` — keep compiled object cache in memory only; avoids disk writes for ephemeral or conditional cache builds
- `off` — disable the cache entirely

The `-no-cache` flag also forces cache mode to `off`.

| File size | SHA256 (pre-2.0.0) | BLAKE3 (2.0.0+) |
|-----------|--------------------|-----------------|
| 10 MB     | ~58 ms             | ~8.7 ms         |

**Disable caching:**

```bash
fz -dir ./src -no-cache
```

**Clear cache:**

```bash
fz -dir . -clean
```

---

### 18.2 Pre-link Symbol Check

Before invoking the linker, `fz` scans all compiled object files for duplicate global symbol definitions.

**Tools used (in order of preference):** `nm` → `objdump` → `readelf`

If a conflict is found, `fz` reports which files define the duplicate symbol and exits with code `1`.

**Disable the check:**

```bash
fz -dir ./src -no-symbol-check
```

---

### 18.3 Watch Mode

```bash
fz -dir ./src -watch
fz -asm main.asm -watch
```

Uses [fsnotify](https://github.com/fsnotify/fsnotify) for cross-platform filesystem events. Rebuilds are debounced with a **500 ms** delay.

---

### 18.4 JSON Output

When `-json` is passed, a single JSON object is written to stdout on completion:

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

**CI/CD integration:**

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

### 18.5 Clean

```bash
fz -dir . -clean
```

Removes: `.fz_objs/`, `.fz_cache/`, all `.o` files, and executable files identified by the `+x` permission bit.

> **Caution:** Avoid running `-clean` in directories containing pre-built third-party binaries you did not intend to delete.

---

### 18.6 Parallel Builds

```bash
fz -dir ./src -j 4   # compile up to 4 files simultaneously
fz -dir ./src -j 0   # auto: use all available CPU cores
```

When a build profile is active, `-j` is set automatically by the profile. An explicit `-j` flag always overrides the profile.

As of **v3.0.0 GLORIA**, the parallel build and logging pipeline is race-condition-free, verified with `go test -race`.

---

### 18.7 Interactive Shell

```bash
fz -shell
```

Opens a REPL for issuing `fz` commands without re-invoking the binary.

| Command | Description |
|---------|-------------|
| `build <file>` | Compile and link a single source file |
| `build -dir <dir>` | Build all files in a directory |
| `set <flag> <value>` | Set a build flag for subsequent commands |
| `clean` | Remove build artifacts |
| `help` | List available commands |
| `exit` | Exit the shell |

---

### 18.8 Virtual Filesystem Layer (VFS)

> **New in v3.1.0 Aegis**

All durable and security-sensitive I/O routes through the `internal/fs` `FileSystem` interface. Production code never calls `os.Open`, `os.WriteFile`, or `os.Rename` directly on security-sensitive paths.

| Component | Role |
|-----------|------|
| `FileSystem` interface | Contract for mkdir, read, write, verified open, temp files, rename, stat, symlinks |
| `fs.Unix` | POSIX implementation (`//go:build !windows`) |
| `fs.Windows` | Native Windows implementation (`//go:build windows`) |
| `fs.Mock` | Test double that injects per-operation errors |
| `utils.SetFileSystem` | Runtime swap used only in tests |

Full detail: [Section 35](#35-virtual-filesystem-layer-aegis).

---

## 19. Configuration File Reference

### 19.1 Basic Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `source_dir` | string | — | Single source directory (backward compatible) |
| `source_dirs` | `[]string` | — | Multiple source directories, each scanned recursively |
| `source_files` | `[]string` | — | Exact list of files to build; overrides `source_dirs` |
| `output` | string | auto | Output binary name |
| `mode` | string | `auto` | Linking mode: `auto`, `c`, or `raw` |
| `format` | string | `elf64` | Output format: `elf32`, `elf64`, or `bin` |
| `target` | string | — | Cross-compilation target triple |
| `backend` | string | `auto` | Compiler backend: `auto`, `gcc`, `clang`, or `zig` |
| `profile` | string | `balanced` | Build profile: `performance`, `balanced`, or `power-saver` |
| `reproducible` | bool | `false` | Enable reproducible (deterministic) build mode |
| `type` | string | `executable` | Output type: `executable` or `static` |
| `lib` | string | — | Library name for `-type static` |
| `jobs` | int | Profile-dependent | Parallel compilation jobs (`0` = auto); overrides profile |
| `debug` | bool | `false` | Emit debug symbols (`-g`) |
| `verbose` | bool | `false` | Print all invoked commands |
| `keep_obj` | bool | `false` | Preserve object files after linking |
| `no_cache` | bool | `false` | Disable build cache |
| `cache_mode` | string | `disk` | Build cache mode: `disk`, `ram`, or `off`. `ram` stores object cache in memory only, `off` disables caching. |
| `sanitize` | bool | `true` | Enable ASan + UBSan for C/C++ |
| `strict` | bool | `false` | Strict sanitizer mode, prefers `clang`/`clang++` |
| `ignore_file` | string | `.fzignore` | Path to a `.gitignore`-style exclusion file |

---

### 19.2 Multiple Source Directories

```yaml
source_dirs:
  - kernel
  - libc
  - drivers
output: forgeos.elf
mode: raw
```

Object file names are prefixed with their parent directory:

| Source file       | Object file         |
|-------------------|---------------------|
| `kernel/boot.asm` | `kernel_boot_asm.o` |
| `libc/string.c`   | `libc_string_c.o`   |
| `drivers/uart.c`  | `drivers_uart_c.o`  |

---

### 19.3 Explicit Source File Lists

```yaml
source_files:
  - boot/start.asm
  - kernel/main.c
  - kernel/irq.c
output: kernel.elf
mode: raw
```

`source_files` takes precedence over `source_dirs` and `source_dir`. Each path is verified at startup.

---

### 19.4 Include & Exclude Patterns

```yaml
exclude:
  - "test_*"
  - "*/legacy/"
  - "*.tmp"

include:
  - "*.asm"
  - "*.c"
```

**Evaluation order:** `exclude` → `.fzignore` → symlink boundary check → `include` → supported extensions.

---

### 19.5 Library Linking

```yaml
libs:
  - m         # -lm  (math)
  - pthread   # -lpthread
  - c         # -lc
```

---

### 19.6 Custom Compiler & Linker Flags

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
```

---

### 19.7 .fzignore File

Works exactly like `.gitignore`. Evaluated after `exclude` patterns.

```
*.o
*.swp
temp/
test_*/
vendor/
legacy/old_abi.asm
```

---

### 19.8 Full Annotated Example

```yaml
# fz.yaml

source_dirs:
  - kernel
  - libc
  - drivers

output: forgeos.elf
format: elf64

profile: balanced
# target: arm-linux-gnueabihf
# backend: zig
# reproducible: true

mode: raw
debug: true
verbose: false
keep_obj: true
no_cache: false
cache_mode: disk
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
  - "*.m"
  - "*.glo"
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

## 20. Assembler Backends

### 20.1 NASM (.asm)

**Command:** `nasm -felf64 <file> -o <output.o>`

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

### 20.2 GAS (.s / .S)

**Command:** `gcc -c <file> -o <output.o>`

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

### 20.3 FASM (.fasm)

**Command:** `fasm <file> <output.o>`

As of **v3.0.0 GLORIA**, ForgeZero automatically injects `format ELF64 executable` when no `format` directive is found, and passes `-dDEBUG=1` when `-debug` is active.

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

## 21. Zig Toolchain Backend

> **New in v3.0.0 GLORIA** · cross-architecture validated in **v4.1.0 Citadel** · musl extended in **v4.5.1**

### 21.1 Why Zig?

The Zig compiler ships as a single self-contained binary that includes: a full copy of musl libc, glibc stubs, and WASI sysroot for every supported target; all necessary C and C++ runtime headers; a Clang-based C/C++ frontend. This makes ForgeZero genuinely zero-dependency for cross-compilation when the Zig backend is active.

### 21.2 Activating the Zig Backend

```bash
fz -cc main.c -zig
fz -dir ./src -zig
fz -cc main.c -zig -target aarch64-linux-musl
fz -cc main.c -zig -target riscv64-linux-musl
fz -cc main.c -zig -target x86_64-windows-gnu
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

### 21.3 Toolchain Selection Logic

| Flags active | C compiler | C++ compiler | Obj-C compiler |
|---|---|---|---|
| (default) | `gcc` or `clang` | `g++` or `clang++` | `clang` |
| `-zig` | `zig cc` | `zig c++` | `zig cc` |
| `-zig -target <T>` | `zig cc -target T` | `zig c++ -target T` | `zig cc -target T` |
| `-zig -strict` | `zig cc` (full sanitizers) | `zig c++` | `zig cc` |

### 21.4 Installing Zig

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

### 21.5 FASM Improvements (v3.0.0)

**Automatic `format ELF64` injection.** When ForgeZero compiles a `.fasm` file and the file does not begin with a `format` directive, it automatically prepends `format ELF64 executable` in a temporary preprocessed copy. The original source file is never modified.

**Debug flag pass-through.** When `-debug` is active, `-dDEBUG=1` is injected into the FASM command line and DWARF debug information is emitted.

```asm
if defined DEBUG
    ; print register state, extra checks
end if
```

```bash
fz -asm boot.fasm -debug
gdb ./boot
(gdb) break _start
(gdb) run
```

---

## 22. Supply Chain Security

> **New in v3.0.0 GLORIA** · atomic SBOM generation hardened in **v4.1.0 Citadel**

### 22.1 SBOM Generation (fz sbom)

```bash
fz sbom
fz sbom -o /tmp/myproject-sbom.cdx.json
fz sbom -dir ./src -o release-sbom.cdx.json
```

`fz sbom` generates a **CycloneDX** SBOM (valid JSON, importable into Dependency-Track, Grype, Syft). As of v4.1.0, SBOM artifacts are produced atomically via internal helpers, preventing partial output files on crash or interrupt.

Each component entry includes: name, version (Git commit SHA for vendored packages), BLAKE3 hash, type (`source-file`, `vendored-package`, `system-library`), and SPDX license identifier.

---

### 22.2 SAST Audit Scanner (fz audit)

```bash
fz audit
fz audit -dir ./src
fz audit -json
```

Three classes of checks:

**Check 1 — Hardcoded Secrets.** Entropy-weighted regex patterns detect AWS keys, GitHub tokens, Slack tokens, generic `key`/`secret`/`password`/`token` assignments, PEM private key blocks, and plaintext connection strings. Findings include file, line, pattern type, and severity — never the actual secret value.

**Check 2 — License Compliance.**

| License | Concern | Severity |
|---|---|---|
| MPL-2.0 | Modified files must be released under MPL | WARNING |
| GPL-2.0 / GPL-3.0 | Strong copyleft | HIGH |
| AGPL-3.0 | Network copyleft | HIGH |
| Proprietary in `vendor/` | May prohibit redistribution | CRITICAL |

**Check 3 — Dangerous Patterns.** Detects `gets()`, `sprintf()` without bounds, `alloca()` inside loops, format string vulnerabilities, unchecked `malloc()` / `realloc()`, `chmod 777`, and `curl | sh` patterns.

**JSON output:**

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

Exit code `0` = no findings; `1` = any WARNING or above.

---

## 23. Reproducible Builds

> **New in v3.0.0 GLORIA** · deterministic output validated across `amd64`/`arm64` in **v4.1.0 Citadel**

ForgeZero eliminates all known sources of non-determinism when `--reproducible` is active:

| Source of non-determinism | Elimination method |
|---------------------------|-------------------|
| Random build IDs | `-Wl,--build-id=none` |
| `__DATE__` / `__TIME__` macros | `SOURCE_DATE_EPOCH` forced to last Git commit timestamp |
| Absolute paths in DWARF info | `-fdebug-prefix-map=/absolute/path=.` |
| Object sort order | Lexicographic sort before linker invocation |

```bash
fz -dir ./src --reproducible
```

```yaml
reproducible: true
```

**Verifying reproducibility:**

```bash
# Machine A
fz -dir ./src --reproducible -out release_a
sha256sum release_a

# Machine B (same source, same commit)
fz -dir ./src --reproducible -out release_b
sha256sum release_b
# Both hashes must match
```

---

## 24. Source Tree Integrity (fz verify)

> **New in v3.0.0 GLORIA**

**Generate a manifest:**

```bash
fz verify --generate
fz verify --generate -manifest ./release.manifest
```

The manifest is a plain text file (one BLAKE3 hex hash + relative path per line).

**Verify against a manifest:**

```bash
fz verify
fz verify -manifest ./release.manifest
fz verify --strict   # also report UNTRACKED files
```

Three categories of finding: **MODIFIED**, **MISSING**, **UNTRACKED** (only with `--strict`).

**CI/CD integration:**

```bash
fz verify -manifest ./release.manifest
# Exits 1 if any file has been tampered with
```

**Symlink boundary protection** is always active and cannot be disabled. Every symlink target is resolved and checked against the project root. Symlinks escaping the project boundary are skipped with a warning.

---

## 25. Build Profiler (fz bench)

> **New in v3.0.0 GLORIA**

```bash
fz bench
fz bench -dir ./src
fz bench -n 5
fz bench -json
```

**Output format:**

```
fz bench — ForgeZero Build Profiler
Project: ./src   Files: 12   Mode: auto   Cache: cold

Phase                       Start (ns)      Duration        % Total
─────────────────────────────────────────────────────────────────────
Cache check                        0 ns       421,330 ns      0.12%
Compile: src/main.c          421,330 ns    18,204,772 ns      5.21%
Compile: src/parser.c     31,035,105 ns    87,304,221 ns     24.99%
Pre-link symbol check    298,104,552 ns     1,203,449 ns      0.34%
Link                     299,307,001 ns    46,882,004 ns     13.42%
─────────────────────────────────────────────────────────────────────
Total                                      349,323,226 ns    100.00%
```

**JSON output:**

```json
{
  "total_ns": 349323226,
  "total_ms": 349,
  "cache": "cold",
  "phases": [
    { "name": "Compile: src/main.c", "start_ns": 421330, "duration_ns": 18204772 },
    { "name": "Link", "start_ns": 299307001, "duration_ns": 46882004 }
  ]
}
```

---

## 26. WebAssembly (WASM)

> **New in v3.0.0 GLORIA**

| Target triple | Runtime | Use case |
|---|---|---|
| `wasm32-emscripten` | Emscripten / Browser | Full libc emulation, JS glue loader |
| `wasm32-wasi` | Wasmtime, WasmEdge, WAMR | Server-side / cloud-native modules |

**WASI via Zig (recommended — no extra SDK):**

```bash
fz -cc main.c -zig -target wasm32-wasi -out main.wasm
wasmtime main.wasm
```

**Emscripten:**

```bash
source /path/to/emsdk/emsdk_env.sh
fz -cc main.c -target wasm32-emscripten -out main.js
# Produces main.wasm + main.js
```

**WASM config:**

```yaml
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

> **Note:** ASan/UBSan are automatically disabled for `wasm32-*` targets. Pass `-sanitize=false` to suppress the notice.

---

## 27. Project Initialization

Added in **v1.6.0**.

```bash
mkdir myproject && cd myproject
fz -init
```

| File | Contents |
|------|----------|
| `.fz.yaml` | Minimal project configuration with commented fields |
| `.fzignore` | Sensible default ignore rules |
| `README.md` | Project README template with `fz` build instructions |

No existing file is overwritten.

---

## 28. Contributor Guidance (fz contribute)

> **New in v4.7.0**

```bash
fz contribute
```

`fz contribute` generates a `CONTRIBUTING_USER.md` file in the current working directory. This file is intended for developers who want to contribute to a ForgeZero-built project and need a clear, environment-specific starting point.

**Generated content:**

The file is populated dynamically based on the output of `fz doctor` and the current project's configuration:

- **Environment diagnostics** — a summary of toolchain reachability, directory permissions, and platform data collected by the `fz doctor` audit. This section reflects the actual state of the contributor's machine, not a generic checklist.
- **Build instructions** — the exact commands to build and test the project, derived from the active `.fz.yaml` or CLI defaults.
- **Test recommendations** — standard `go test` invocations, coverage targets, and lint commands aligned with the project's testing policy.
- **Good first issues guidance** — instructions for finding beginner-friendly issues, including a link to the repository's issue tracker filtered for the `good first issue` label.
- **PR and commit guidelines** — branch naming conventions, commit message format (imperative mood, present tense), and pull request description template.

**Example:**

```bash
fz contribute
# Writes CONTRIBUTING_USER.md to the current directory
cat CONTRIBUTING_USER.md
```

The file is written via `SecureWriteFile` (atomic write, mode `0644`). No existing `CONTRIBUTING_USER.md` is overwritten without confirmation.

---

## 29. LSP & IDE Integration

Added in **v1.9.0**.

```bash
fz -compile-commands
fz -dir ./src -compile-commands
```

| Editor | Language server | Notes |
|--------|----------------|-------|
| Neovim | clangd | Install via Mason; auto-detects `compile_commands.json` |
| VSCode | clangd extension | Point it to the project root |
| CLion | Built-in | Opens `compile_commands.json` automatically |
| Helix | clangd | Set `language-server = "clangd"` in `languages.toml` |
| Emacs | eglot / lsp-mode | Both read `compile_commands.json` from project root |

---

## 30. Self-Update

Added in **v1.9.0**.

```bash
fz -update
```

What happens: fetches the latest release binary, backs up the current binary to `fz.old`, installs the new one, reports version change.

**Rolling back:**

```bash
sudo cp /usr/local/bin/fz.old /usr/local/bin/fz
```

---

## 31. Examples

### Minimal builds

```bash
fz -asm hello.asm
fz -asm hello.s
fz -asm hello.fasm
fz -cc main.c
fz -cc main.cpp
fz -cc main.m
fz -gloria main.glo
```

### Initialize a new project

```bash
mkdir myproject && cd myproject
fz -init
mkdir src
echo 'int main(void) { return 0; }' > src/main.c
fz
```

### Build with a profile

```bash
# Maximum speed — all cores, -O3
fz -p performance -dir ./src

# Battery-friendly — single core, -Os
fz -p power-saver -cc main.c

# Default balanced profile (half cores, -O2)
fz -profile balanced -dir ./src
```

### Debug build with verbose output

```bash
fz -asm hello.asm -debug -verbose
gdb ./hello
```

### Bare-metal / bootloader binary

```bash
fz -asm boot.asm -mode raw -format bin -out boot.bin
```

### Direct linker invocation (v4.1.0)

```bash
fz -asm boot.asm -ld -out boot.elf
```

### C with strict sanitizers

```bash
fz -cc main.c -strict
```

### Objective-C application

```bash
fz -cc main.m -verbose
```

### Gloria bare-metal kernel output

```bash
fz -gloria kernel.glo -out kernel.elf
```

### Build a full project directory

```bash
fz -dir ./src
```

### Parallel build

```bash
fz -dir ./src -j 0
```

### Cross-compile for ARM (with system toolchain)

```bash
fz -cc main.c -target arm-linux-gnueabihf -sanitize=false
```

### Cross-compile for ARM64 musl (Zig — no packages needed)

```bash
fz -cc main.c -zig -target aarch64-linux-musl -sanitize=false
```

### Cross-compile for RISC-V musl (v4.5.1)

```bash
fz -cc main.c -zig -target riscv64-linux-musl -sanitize=false
```

### Build a static library

```bash
fz -dir ./src -type static -lib mylib
ls libmylib.a
```

### Build a shared library

```bash
fz -cc mylib.c -shared -cc-flag "-O2 -fPIC" -o libmylib.so
```

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

### Generate LSP compilation database

```bash
fz -dir ./src -compile-commands
cat compile_commands.json
```

### Build from multiple directories

```bash
# .fz.yaml: source_dirs: [src, lib], output: release
fz
```

### Generate a Software Bill of Materials

```bash
fz sbom
cat sbom.cdx.json
```

### Run the security audit

```bash
fz audit
fz audit -json | tee audit_report.json
```

### Reproducible build

```bash
fz -dir ./src --reproducible
sha256sum ./src
```

### Generate and verify source tree manifest

```bash
fz verify --generate
fz verify
```

### Profile the build

```bash
fz bench -dir ./src
fz bench -dir ./src -n 5 -json | tee bench_report.json
```

### Generate contributor guidance

```bash
fz contribute
cat CONTRIBUTING_USER.md
```

### Reproduce the official benchmark

```bash
git clone https://github.com/forgezero-cli/ForgeZero
cd ForgeZero
go build -o fz ./cmd/fz

# Run benchmark script (edit NUM_MODULES in script for different sizes)
./bench.sh

# Export results to Markdown
hyperfine --warmup 3 --prepare 'make clean && rm -rf .fz_objs fz_out' \
  './fz -dir . -out fz_out' 'make -j4' \
  --export-markdown results.md
```

### Build for WebAssembly (WASI, via Zig)

```bash
fz -cc main.c -zig -target wasm32-wasi -out main.wasm
wasmtime main.wasm
```

### Build for WebAssembly (browser, via Emscripten)

```bash
source /path/to/emsdk/emsdk_env.sh
fz -cc main.c -target wasm32-emscripten -out main.js
```

### Directory build with JSON output (CI/CD)

```bash
fz -dir ./src -json | tee build_report.json
```

### Watch mode during development

```bash
fz -dir ./kernel -watch
```

### Disable sanitizers for release

```bash
fz -cc main.c -sanitize=false -out main_release
```

### Keep object files for inspection

```bash
fz -dir ./src -keep-obj -verbose
ls .fz_objs/
```

### Interactive shell session

```bash
fz -shell
# fz> build main.c
# fz> set mode raw
# fz> build boot.asm
# fz> exit
```

### Clean all build artifacts

```bash
fz -dir . -clean
```

### Update fz with rollback

```bash
sudo fz -update
# If something breaks:
sudo cp /usr/local/bin/fz.old /usr/local/bin/fz
```

---

## 32. Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success — binary produced; or `fz verify` / `fz audit` found no violations; or `fz doctor` is healthy. |
| `1` | Build error — assembler, compiler, or linker failed; duplicate global symbol; `fz verify` found MODIFIED or MISSING files; `fz audit` found WARNING or above; `fz doctor` is degraded. |
| `2` | Argument error — invalid or missing flags, source file not found, cross-compiler not on PATH, unreadable config file. |

---

## 33. Troubleshooting

### `fz: command not found`

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### `nasm: command not found`

```bash
sudo apt install nasm          # Debian / Ubuntu
sudo dnf install nasm          # Fedora
sudo pacman -S nasm            # Arch
brew install nasm              # macOS
pacman -S mingw-w64-x86_64-nasm  # Windows MSYS2
```

### `fasm: command not found`

```bash
wget https://flatassembler.net/fasm-1.73.32.tgz
tar -xzf fasm-1.73.32.tgz
sudo cp fasm/fasm /usr/local/bin/
chmod +x /usr/local/bin/fasm
```

### `g++: command not found`

```bash
sudo apt install g++           # Debian / Ubuntu
sudo dnf install gcc-c++       # Fedora
sudo pacman -S gcc             # Arch (g++ included)
brew install gcc               # macOS
```

### `clang: command not found` (Objective-C)

```bash
sudo apt install clang         # Debian / Ubuntu
sudo dnf install clang         # Fedora
sudo pacman -S clang           # Arch
# macOS: xcode-select --install
```

### `zig: command not found`

```bash
wget https://ziglang.org/download/0.13.0/zig-linux-x86_64-0.13.0.tar.xz
tar -xf zig-linux-x86_64-0.13.0.tar.xz
sudo mv zig-linux-x86_64-0.13.0 /opt/zig
echo 'export PATH="$PATH:/opt/zig"' >> ~/.bashrc
source ~/.bashrc

# Or via package manager
sudo pacman -S zig
brew install zig
```

### Cross-compiler not found (without Zig)

```bash
sudo apt install gcc-arm-linux-gnueabihf
sudo dnf install gcc-arm-linux-gnu
sudo pacman -S arm-linux-gnueabihf-gcc

# Or switch to the Zig backend to avoid all of this:
fz -cc main.c -zig -target arm-linux-gnueabihf
```

### `undefined reference to _start`

Define `_start` explicitly:

```asm
global _start
_start:
    ; ...
```

Or use the C runtime:

```bash
fz -asm program.asm -mode c
```

### Binary crashes immediately (segfault on startup)

Likely `-mode raw` used with code referencing libc symbols. Switch to:

```bash
fz -asm program.asm -mode c
```

### Pre-link duplicate symbol error

Check for conflicting `global` declarations. Skip the check when using weak symbols intentionally:

```bash
fz -dir ./src -no-symbol-check
```

### Sanitizer error at runtime

Fix the reported memory/UB issue. To temporarily disable:

```bash
fz -cc main.c -sanitize=false
```

### Sanitizers silently disabled for WASM target

Expected behavior. Pass `-sanitize=false` to suppress the notice.

### `fz verify` reports MODIFIED files unexpectedly

Re-generate the manifest from the current known-good state:

```bash
fz verify --generate
```

### `fz audit` false positive on a secret pattern

Annotate the line with a suppression comment:

```c
const char *example = "not-a-real-key"; // fz-audit: ignore
```

### Build hangs / times out

```bash
fz -asm big_program.asm -timeout 300
```

### Cache returns stale results

```bash
fz -dir . -clean
fz -dir ./src
# Or one-off:
fz -dir ./src -no-cache
```

### Build profile not applying expected optimization level

Verify the persisted profile:

```bash
cat ~/.config/fz/.profile.config
```

Reset to default:

```bash
rm ~/.config/fz/.profile.config
fz -profile balanced -cc main.c
```

### `fz pm add` fails / git not found

```bash
sudo apt install git
sudo dnf install git
sudo pacman -S git
brew install git
```

### `fz pm install` hash mismatch

The downloaded package content does not match the BLAKE3 hash in the catalog manifest. Do not use the package. Report the issue at [github.com/forgezero-cli/catalog](https://github.com/forgezero-cli/catalog).

### `compile_commands.json` not picked up by editor

Ensure the file is in the **project root**. Regenerate after adding new source files:

```bash
fz -compile-commands
```

### `fz -update` fails with permission denied

```bash
sudo fz -update
```

### Watch mode does not detect changes on WSL2

Edit files from within the WSL2 terminal. This is a WSL2 kernel limitation with `inotify` when files are edited from Windows applications.

### Windows: `gcc` not found

Ensure MSYS2 `mingw64\bin` is in your Windows `PATH`:

```
C:\msys64\mingw64\bin
```

---

## 34. Roadmap

| Feature | Status |
|---------|--------|
| `exclude` patterns in config file | Done (v1.5.0) |
| `include` patterns in config file | Done (v1.5.0) |
| Multiple `source_dirs` | Done (v1.5.0) |
| Explicit `source_files` list | Done (v1.5.0) |
| `libs` field for library linking | Done (v1.5.0) |
| `flags.cc` for C compiler flags | Done (v1.5.0) |
| `.fzignore` file support | Done (v1.5.0) |
| Multi-level config merging | Done (v1.5.0) |
| `fz -init` project scaffolding | Done (v1.6.0) |
| `-format bin` flat binary output | Done (v1.6.0) |
| Parallel builds (`-j N`) | Done (v1.7.0) |
| Interactive shell (`fz -shell`) | Done (v1.7.0) |
| C++ support (`.cpp`, `.cc`, `.cxx`) | Done (v1.7.0) |
| Static library mode (`-type static`) | Done (v1.8.0) |
| Unique object file names (path-based) | Done (v1.8.0) |
| Cross-compilation (`-target <triple>`) | Done (v1.9.0) |
| LSP integration (`-compile-commands`) | Done (v1.9.0) |
| Smart self-update with rollback | Done (v1.9.0) |
| BLAKE3 hashing (7× faster cache) | Done (v2.0.0) |
| Package manager (`fz pm`) | Done (v2.0.0) |
| Official package catalog | Done (v2.0.0) |
| Shared library support (`-shared`) | Done (v2.0.0) |
| Zig toolchain backend (`-zig`) | Done (v3.0.0) |
| SBOM generation (CycloneDX + BLAKE3) | Done (v3.0.0) |
| SAST audit scanner (`fz audit`) | Done (v3.0.0) |
| Reproducible builds (`--reproducible`) | Done (v3.0.0) |
| Source tree verification (`fz verify`) | Done (v3.0.0) |
| Symlink boundary protection | Done (v3.0.0) |
| Build profiler (`fz bench`) | Done (v3.0.0) |
| Race-condition-free parallel pipeline | Done (v3.0.0) |
| FASM native ELF64 auto-injection | Done (v3.0.0) |
| WebAssembly (`wasm32-emscripten` / `wasm32-wasi`) | Done (v3.0.0) |
| VFS abstraction + `OpenVerified` TOCTOU hardening | Done (v3.1.0) |
| Aegis hardened `RunCommand` wrapper | Done (v3.1.0) |
| `SecureWriteFile` atomic write pipeline | Done (v3.1.0) |
| Constant-time toolchain checksum verify | Done (v3.1.0) |
| `fz doctor` self-audit command | Done (v3.1.0) |
| Native Windows `fs.Windows` + rename retry | Done (v3.1.0) |
| 90%+ coverage + `fs.Mock` fault injection | Done (v3.1.0) |
| Zero-allocation linker hot-path (`0 allocs/op`) | Done (v4.1.0) |
| Stack-buffered syscalls (`openHot`, `unlinkHot`) | Done (v4.1.0) |
| HADES ELF emitter refactor (`.symtab` ordering, fixed relocations) | Done (v4.1.0) |
| Direct linker invocation (`-ld` flag) | Done (v4.1.0) |
| Arena size override (`--arena-size=N`) | Done (v4.1.0) |
| Platform-specific syscall drivers (Linux/Windows/Darwin) | Done (v4.1.0) |
| Cross-architecture validation (`amd64` + `arm64`) | Done (v4.1.0) |
| 400-module benchmark data point (4.95× speedup) | Done (v4.1.0) |
| 100% `golangci-lint` compliance (strict mode) | Done (v4.1.0) |
| Atomic SBOM generation helpers | Done (v4.1.0) |
| Gloria `while` loops | Done (v4.4.0) |
| Gloria `peek` / `poke` memory access | Done (v4.4.0) |
| Gloria `in8` / `out8` I/O port primitives | Done (v4.4.0) |
| Gloria VGA framebuffer output (`0xB8000`) | Done (v4.4.0) |
| Gloria register constants (R0–R15) | Done (v4.4.0) |
| Gloria extended register support (R8–R15) | Done (v4.4.0) |
| Embedded musl runtime via `go:embed` (Linux x86_64) | Done (v4.5.0-think) |
| RISC-V + musl cross-compilation (`riscv64-linux-musl`) | Done (v4.5.1) |
| Gloria error propagation improvements | Done (v4.5.1) |
| Objective-C support (`.m`, Clang/Zig, auto-linking) | Done (v4.6.0) |
| Build profiles (`-profile` / `-p`) | Done (v4.7.0) |
| `fz contribute` — `CONTRIBUTING_USER.md` generation | Done (v4.7.0) |
| Version output refinement (`-v` / `--version`) | Done (v4.7.0) |
| Colored terminal output (green success / red error) | Planned |
| GDB integration and improved debug workflow | Planned |
| Man page (`man fz`) | Planned |
| Windows native support without WSL2 | In progress |
| macOS full support and tested runtime | In progress |

---

## 35. Virtual Filesystem Layer (Aegis)

> **Package:** `internal/fs` · **Consumers:** `internal/utils`, `internal/doctor`, `internal/verify`, `internal/sbom`, `internal/pkgman`

ForgeZero v3.1.0 introduces a deliberate separation between *what* filesystem operations the build tool requires and *how* the host operating system performs them. Goals: enable deterministic fault-injection tests without patching `os` globally, and centralize symlink and path-substitution defenses in one audited code path.

### 35.1 Design: The `FileSystem` Interface

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

**Binding model:** `fs.Default` is set to `Unix{}` or `Windows{}` at compile time via build tags. `utils.fileSystem()` returns the active implementation.

### 35.2 Unix Implementation and TOCTOU Mitigation

`OpenVerified` on Unix:

```go
pre, err := os.Lstat(path)          // metadata without following final symlink
if pre.Mode()&os.ModeSymlink != 0 {
    return nil, ErrSymlink            // reject symlinks outright
}
f, err := os.Open(path)
post, err := f.Stat()               // metadata of the opened fd
if !os.SameFile(pre, post) {
    f.Close()
    return nil, ErrPathChanged        // inode/device changed between check and use
}
return f, nil
```

**Why `Lstat`?** `Stat` follows symlinks. An attacker could place a symlink at `path`; `Lstat` inspects the link node itself and rejects it immediately.

**Why `SameFile` after open?** This closes the TOCTOU window:

| Phase | Attacker action | Without `SameFile` | With `SameFile` |
|-------|-----------------|--------------------|-----------------|
| T0 | `path` is a regular file | Check passes | `Lstat` records inode A |
| T1 | Attacker replaces `path` with symlink to `/etc/passwd` | — | — |
| T2 | Reader calls `Open` | May read sensitive file | `Open` follows new symlink |
| T3 | Compare identities | — | `os.SameFile(pre, post)` fails → `ErrPathChanged` |

### 35.3 Windows Implementation

Native Windows support is a **separate compilation unit**, not a runtime fork:

| File | Build constraint | Purpose |
|------|------------------|---------|
| `unix.go`, `default_unix.go`, `rename_unix.go` | `!windows` | POSIX backend |
| `windows.go`, `default_windows.go`, `rename_windows.go` | `windows` | Win32 API backend |
| `pathnorm.go` | all platforms | `CleanPath`, `HasDrivePrefix`, `IsUNC`, `NormalizeAbs` |

`OpenVerified` on Windows follows the same logical steps as Unix. Every entry point normalizes paths through `CleanPath` (drive letters, backslashes, UNC prefixes).

### 35.4 The `Mock` Implementation and Fault Injection

```go
m := fs.NewMock(fs.Default)
m.SetFailOp("Rename", fs.ErrDiskFull)
m.SetFail("OpenVerified", resolvedPath, fs.ErrPermission)
utils.SetFileSystem(m)
defer utils.SetFileSystem(nil)
```

| Variable | Simulated condition |
|----------|---------------------|
| `ErrDiskFull` | `ENOSPC` / quota exhaustion |
| `ErrPermission` | `EACCES` / `EPERM` |
| `ErrTimeout` | I/O timeout |
| `ErrInterrupted` | `EINTR` |
| `ErrSymlink` | Symlink policy rejection |
| `ErrPathChanged` | TOCTOU detection |

### 35.5 Consumer Integration

| Function | VFS operations |
|----------|----------------|
| `SecureWriteFile` | `MkdirAll`, `CreateTemp`, `Chmod`, write, close, `Rename`, `Chmod` |
| `ReadFileSecure` | `ResolveSecurePath` → `OpenVerified` → read |
| `CopyFile` | `OpenVerified` source, temp file, `Rename` |
| `StatResolved` / `ReadDirResolved` | `Stat`, `ReadDir` on resolved paths |

---

## 36. Aegis Security Core

> **Packages:** `internal/utils`, `internal/fs`, `internal/pkgman`, `internal/assembler`, `internal/linker`, `internal/zig`

### 36.1 Command Hardening: The `RunCommand` Pipeline

Every external process spawned by ForgeZero — `git`, `ar`, `zig`, `fasm`, `gcc`, `g++`, `clang`, `ld`, `nasm`, `objdump`, `nm`, `readelf` — must pass through `utils.RunCommand`.

**Stage 1 — Argument validation:** `ValidateCLIArg` rejects shell metacharacters, backticks, pipes, redirection symbols, and embedded NUL/newline bytes.

**Stage 2 — Absolute executable resolution:** `exec.LookPath` resolves the binary; no bare-name invocations.

**Stage 3 — Argument sanitization:** Each argument is validated, preventing injection via malicious config flags or crafted filenames.

**Stage 4 — Fixed environment:**

| Variable | Value | Rationale |
|----------|-------|-----------|
| `LC_ALL` | `C` | Stable locale sorting and diagnostics |
| `LANG` | `C` | Same |
| `TZ` | `UTC` | Reproducible timestamps |
| `SOURCE_DATE_EPOCH` | `1600000000` | Aligns with reproducible build expectations |

**Stage 5 — Execution root:** `cmd.Dir` is set to the project execution root so relative paths in tool invocations resolve inside the project tree.

### 36.2 Atomic Writes: `SecureWriteFile`

`atomicWrite` algorithm:

1. `CreateTemp` in the destination directory (same volume for atomic rename).
2. `Chmod(tmpName, 0600)`.
3. Write full payload.
4. `Close` (buffers flushed to kernel).
5. `renameResolved(tmpName, resolved)`.
6. `Chmod(resolved, 0600)`.

Files written through this path include: `.fz.yaml` updates from `fz pm`, `.fz.manifest`, `compile_commands.json`, SBOM outputs, `CONTRIBUTING_USER.md`, and `fz -init` templates.

### 36.3 Constant-Time Toolchain Checksum Verification

`.fz.yaml` may specify expected BLAKE3 digests per tool binary:

```yaml
tool_checksums:
  gcc: "abc123..."
  nasm: "def456..."
```

Comparison uses `crypto/subtle.ConstantTimeCompare` to remove timing side-channels. Mismatch produces `tool checksum mismatch for <name>` and fails the build before any compilation.

### 36.4 Execution Root and Path Confinement

`EnsureInsideRoot`, `HashDirWithRoot`, and doctor's `scanTree` use `ResolveSecurePath` to ensure scanned paths do not escape the root via symlinks or `..` segments after evaluation. This protection is an invariant, not a configurable option.

---

## 37. System Self-Audit (`fz doctor`)

> **Package:** `internal/doctor` · **Entry point:** `fz doctor [options]`

`fz doctor` is a pre-flight diagnostic. It does not compile code. It answers whether the current machine satisfies ForgeZero's minimum operational requirements.

### 37.1 Invocation

```bash
fz doctor
fz doctor -root /path/to/project
fz doctor -json
fz doctor -root ./myapp -json
```

### 37.2 Audit Pipeline (Four Stages)

**Stage A — Toolchain Reachability:** probes `zig` (required), `fasm` (required on non-Windows), `wasm-ld` (optional).

**Stage B — Recursive Permission Audit:** writes `.fz_doctor_probe` via `SecureWriteFile`, then walks the entire tree with `OpenVerifiedRead` on every regular file. Counts `DirsScanned` and `FilesSeen`.

**Stage C — Platform Integrity:**

| Field | Source |
|-------|--------|
| `platform.goos` | `runtime.GOOS` |
| `platform.goarch` | `runtime.GOARCH` |
| `platform.filesystem_impl` | `fs.ImplName()` (`unix` / `windows`) |
| `platform.execution_root` | `utils.GetExecutionRoot()` |
| `platform.num_cpu` | `runtime.NumCPU()` |

**Stage D — Health Aggregation:** `Healthy = false` if any required tool is missing, or if `Readable` or `Writable` is false.

### 37.3 Human-Readable Output

```
fz doctor: ok
platform: linux/amd64 fs=unix sep="/" root=/home/dev/myproject cpus=16
toolchain:
  zig (required): /usr/local/bin/zig
  fasm (required): /usr/bin/fasm
  wasm-ld: missing
permissions: root=/home/dev/myproject readable=true writable=true dirs=42 files=318
```

### 37.4 JSON Output

```json
{
  "status": "ok",
  "healthy": true,
  "toolchain": [
    { "name": "zig", "required": true, "found": true, "path": "/usr/local/bin/zig" },
    { "name": "fasm", "required": true, "found": true, "path": "/usr/bin/fasm" },
    { "name": "wasm-ld", "required": false, "found": false, "error": "not found in PATH" }
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
  "errors": []
}
```

**CI gate usage:**

```bash
fz doctor -json | jq -e '.healthy'
```

---

## 38. Cross-Platform Readiness

### 38.1 Compile-Time Backend Selection

```
GOOS=linux   → compiles: unix.go, default_unix.go, rename_unix.go
GOOS=windows → compiles: windows.go, default_windows.go, rename_windows.go
GOOS=darwin  → compiles: unix.go (stubbed syscall layer; builds succeed, runtime untested)
```

No `if runtime.GOOS == "windows"` exists inside `OpenVerified`. The correct struct is selected by the Go toolchain at compile time.

### 38.2 Platform-Specific Syscall Drivers (v4.1.0)

| Platform | Implementation | Syscalls |
|----------|---------------|---------|
| Linux | `golang.org/x/sys/unix` | Direct `SYS_OPENAT`, `SYS_UNLINKAT` |
| Windows | `golang.org/x/sys/windows` | `CreateFile`, `ReadFile`, `WriteFile` |
| Darwin | Stubbed syscall layer | Builds succeed; runtime untested |

`openHot` and `unlinkHot` use fixed-size stack buffers for UTF-8 path conversion on all platforms, achieving zero heap allocation in the hot path.

### 38.3 Atomic Rename on Windows

Windows may return sharing violations when AV or indexing software holds a handle. `rename_windows.go` implements bounded retry:

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

Backoff: 10 ms, 20 ms, 30 ms, … up to 80 ms between attempts.

### 38.4 Cross-Architecture Validation (v4.1.0)

Verified builds for `amd64` and `arm64` across Linux, Windows, and Darwin. No CGO dependencies; pure Go + static C linkage where required. Binary digests are stable across rebuilds with identical inputs on all validated platforms.

---

## 39. Testing Standards (Aegis)

> **Policy version:** v4.7.0 · **Command:** `go test ./internal/... -cover`

### 39.1 Coverage Targets

| Package | Coverage class | Notes |
|---------|----------------|-------|
| `internal/pkgman` | ≥ 90% | HTTP catalog mock, `runGit` injection |
| `internal/fs` | ≥ 90% | `Mock` all ops, `OpenVerified`, pathnorm |
| `internal/doctor` | ≥ 90% | Permission failures, JSON, `scanTree` open errors |
| `internal/config` | ≥ 90% | `LoadMerged`, `Merge`, validation |
| `internal/zig` | ≥ 90% | `RunCommand` mock, link/compile failures |
| `internal/man` | 100% | Man page generator |
| `internal/assembler` | high 80s–90s | Mocked `runCommand`, all target triple branches |
| `internal/linker` | high 70s–80s | Zero-allocation benchmarks, Windows impl, symbol parsers |
| `internal/gloria` | ≥ 90% | Walk and mapping errors, VGA buffer tests, builtin propagation |

### 39.2 Zero-Allocation Enforcement (v4.1.0)

Allocation-tracking benchmarks for Linux and Windows targets enforce zero-allocation guarantees in critical paths:

```
BenchmarkCopyFileHot       0 allocs/op   0 B/op
BenchmarkResolveSymbols    0 allocs/op   0 B/op
BenchmarkEmitRelocations   0 allocs/op   0 B/op
```

Failure on any `allocs/op > 0` in hot-path benchmarks blocks merge.

### 39.3 Fault Injection via `fs.Mock`

```go
m := fs.NewMock(fs.Default)
m.SetFailOp("CreateTemp", fs.ErrDiskFull)
utils.SetFileSystem(m)
t.Cleanup(func() { utils.SetFileSystem(nil) })
err := utils.SecureWriteFile("out/config.yaml", data)
// expect error, no partial final file
```

### 39.4 Race and Integration Commands

```bash
go test ./... -race
go test ./internal/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### 39.5 Static Analysis (v4.1.0)

100% compliance with `golangci-lint` under strict configuration:

```bash
golangci-lint run -E gofmt,govet,staticcheck,unused ./...
```

Zero lint warnings in main branch. All PRs must pass this check.

### 39.6 Gloria Test Infrastructure (v4.4.0)

Gloria runtime tests use two helpers to avoid hardware dependencies:

**`patchVGA()`** — replaces the compile-time `0xB8000` constant with a pointer to a heap-allocated fake VGA buffer. Called at the start of any test that exercises VGA output. The substitution is undone via `t.Cleanup`.

**`dumpVGA()`** — renders the fake VGA buffer to stdout using `syscall.Write` with zero heap allocations. Used in golden-file tests to verify expected screen output.

```go
func TestPrintBareMetalHello(t *testing.T) {
    buf := patchVGA(t)
    compileAndRun(t, `fn main() { print("hello\n") }`, kernelMode)
    got := dumpVGA(buf)
    if !strings.Contains(got, "hello") {
        t.Fatalf("VGA output missing expected string: %q", got)
    }
}
```

### 39.7 Contributor Requirements

Pull requests touching `internal/fs`, `internal/utils` (I/O or `RunCommand`), `internal/doctor`, or `internal/gloria` must:

1. Include table-driven tests for new branches.
2. Use `fs.Mock` or existing seams for failure paths.
3. Not regress package coverage below the 90% bar.
4. Pass `go test ./...` locally before review.
5. Pass `golangci-lint` in strict mode.
6. Include benchmark assertions for any new hot-path code — `0 allocs/op` is the bar.

---

## 40. HADES Engine: Codegen & ELF Emission

> **New in v4.1.0 Citadel** · **Gloria extended in v4.4.0**

The HADES engine is ForgeZero's integrated codegen and ELF emission layer. It is responsible for transforming parsed AST nodes into correct, deterministic ELF64 object files without relying on external assembler binaries for supported instruction sets. HADES is the exclusive compilation backend for Gloria (`.glo`) source files and is also used internally for hot-path object emission in the linker.

### 40.1 ELF Emitter Refactor

**Symbol table ordering.** The ELF specification requires local symbols to precede global symbols in `.symtab`. Earlier versions of ForgeZero emitted symbols in parse order, which could produce object files that technically linked but were non-compliant with strict linker expectations. v4.1.0 enforces local-before-global ordering unconditionally.

**Absolute relocation offsets.** `call` and `jmp` target resolution was recalculated to correctly compute offsets for absolute relocations. This was validated via a factorial execution test: a self-referential recursive function compiled and linked through the HADES pipeline produces exit code 120 (`factorial(5)`), confirming that `call` targets resolve to the correct runtime address.

**Instruction/label disambiguation.** The lexer/parser boundary now maintains strict differentiation between CPU instruction mnemonics and user-defined labels. In earlier versions, certain two-character mnemonics could be misidentified as label prefixes when appearing at column zero in specific syntactic contexts. v4.1.0 eliminates this ambiguity with explicit boundary checks, preventing silent object file corruption.

### 40.2 Parser Integrity Guards

The parser maintains invariants at each AST node boundary. Malformed input that previously produced incomplete or silently incorrect AST nodes now triggers an explicit `ErrMalformedAST` error before codegen begins. Silent fallback degradation — where a corrupted AST would produce a broken but structurally valid ELF — is no longer possible.

### 40.3 Codegen Zero-Allocation Path

The HADES codegen path (`resolveSymbols`, `emitRelocations`, `copyFileHot`) was refactored to eliminate all heap allocations from the hot path. All intermediate buffers are stack-allocated with fixed capacity. The codegen pipeline operates at hardware memory bandwidth limits (~1.18 GB/s on Intel i5-10310U) with zero GC interference.

> If you find an allocation in our hot paths — it's a bug.

### 40.4 Extended Instruction Emission (v4.4.0)

The following codegen primitives were added or extended for Gloria JIT support:

| Function | Purpose |
|----------|---------|
| `emitMovMemToReg64` | Emit `MOV reg, [mem]` for arbitrary 64-bit memory reads |
| `emitMovRegToMem64` | Emit `MOV [mem], reg` for arbitrary 64-bit memory writes |
| `emitPushReg` | Push register to stack; extended to R8–R15 |
| `emitPopReg` | Pop register from stack; extended to R8–R15 |
| `emitBareMetalPrint` | Write string directly to VGA framebuffer at `0xB8000` |
| `emitLowLevelPrint` | Dispatch to `sys_write` (userspace) or VGA path (kernel mode) |

All new emission functions maintain the zero-allocation invariant. `parseStringLiteral` resolves `\n` and `\t` escape sequences at compile time, so no runtime string scanning occurs.

### 40.5 Direct Linker Bypass (`-ld`)

The `-ld` flag exposes the HADES linker invocation directly, bypassing the `gcc`/`clang` wrapper layer used in `-mode auto` and `-mode c`. This is distinct from `-mode raw` (which selects `ld` over `gcc` at the linking stage); `-ld` removes the compiler driver from the picture entirely at the build orchestration level. Overhead reduction is ~3–5% on small projects; on large builds dominated by codegen, the difference is negligible.

Use `-ld` when you need precise control over the linker command vector and do not want ForgeZero's compiler validation checks in the critical path.

---

## 41. Contributing

Contributions are welcome: bug reports, feature requests, documentation improvements, and code patches.

1. **Open an issue** before starting significant work to align on the approach.
2. **Fork the repository** and create a descriptive feature branch (`feature/watch-debounce`, `fix/nasm-elf32`).
3. **Write tests** for new behavior:

   ```bash
   go test ./...
   go test ./internal/... -cover
   golangci-lint run -E gofmt,govet,staticcheck,unused ./...
   ```

   Security-sensitive changes must include fault-injection or mock tests per [Section 39](#39-testing-standards-aegis). Hot-path changes must include benchmark assertions (`0 allocs/op`).

4. **Submit a Pull Request** with a clear description of the change and the problem it solves.

Commit messages should be concise and use the imperative mood: *"Add JSON output mode"* not *"Added JSON output mode"*.

For machine-generated contributor guidance, run `fz contribute` in your project directory. The resulting `CONTRIBUTING_USER.md` reflects the actual state of your environment and toolchain.

Repository: [github.com/forgezero-cli/ForgeZero](https://github.com/forgezero-cli/ForgeZero)

---

## 42. License

ForgeZero is released under the **GPLv3 License**.

```

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
```

---

*If ForgeZero saves you time, consider giving the repository a star on GitHub — it helps the project grow.*
