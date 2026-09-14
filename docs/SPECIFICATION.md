# ForgeZero Technical Specification

## Status and scope

This document is the normative engineering description of ForgeZero (`fz`). It describes the implemented orchestration model, cache contracts, configuration schema, compiler dispatch, low-level hashing paths, task execution, Gloria, and Plan 9 assembly interoperability.

ForgeZero is a build orchestrator. It is not a language compiler, linker replacement, or universal shell abstraction. Language semantics, object generation, ABI rules, and final code generation remain the responsibility of the selected external toolchain or the explicitly supported Gloria emitter.

Claims in this document are classified as follows:

- **Contract**: behavior required by the public or package-level implementation.
- **Optimization**: an implementation strategy that may vary by platform or workload.
- **Limitation**: a boundary that must not be inferred away by callers.

## 1. Architecture

### 1.1 Thin coordinator boundary

ForgeZero accepts declarative build intent, normalizes it, discovers the relevant source and dependency graph, computes cache identities, schedules actions, invokes specialized tools, and validates or packages the result. It does not reinterpret the language being compiled.

The coordinator boundary is deliberately narrow:

- CLI code translates user intent into typed runtime state.
- Configuration code parses TOML, legacy YAML, variables, includes, defaults, and validation rules.
- Builder code owns discovery, scheduling, invalidation, dependency execution, and object/cache orchestration.
- Assembler code selects NASM, FASM, GAS, Go's Plan 9 assembler, GCC, Clang, or Zig according to source and target.
- Linker code selects compiler-driver linking, `ld`, archive creation, linker scripts, target flags, and symbol checks.
- Security and analysis packages own path policy, verification, SBOM, audit, sealing, and environment constraints.

No layer is permitted to silently alter the user's ABI, target semantics, optimization flags, linker intent, or output format.

### 1.2 Pipeline transformation

The build pipeline is modeled as:

$$
B = L\left(C\left(S\left(P\left(D\left(N(F,K)\right)\right)\right)\right)\right)
$$

where:

- $F$ is the command-line flag state;
- $K$ is project configuration;
- $N$ normalizes defaults, paths, variables, profiles, and policy values;
- $D$ discovers sources, dependencies, ignored paths, generated inputs, and task relationships;
- $P$ executes the project-owned FZP preprocessing layer;
- $S$ computes source, action, and cache state transitions;
- $C$ dispatches assembler/compiler actions and produces object files;
- $L$ links objects or creates archives;
- $B$ is the resulting artifact and associated report state.

Each transformation has an ownership boundary. A failure at one boundary must be reported as that boundary's error class rather than being disguised as a generic compiler failure.

### 1.3 Throughput model

The useful build throughput is approximated by:

$$
T_{build} \approx \frac{W_{compiler}}{D_{discovery}+D_{dispatch}+D_{repeat}+D_{invalidate}}
$$

The denominator is not literally zero. It is minimized through the following mechanisms:

- **Discovery**: source filtering, ignore rules, dependency closure, and generated include tracking avoid redundant filesystem work.
- **Dispatch**: target-aware backend selection keeps command construction explicit and avoids opaque multi-stage abstractions.
- **Repeated work**: RAM, disk, action, source, and configuration caches reuse prior results when identity contracts match.
- **Invalidation**: metadata-first checks, keyed source hashes, lexical chunk reuse, and action fingerprints prevent unnecessary compiler invocations.

For a single large translation unit dominated by optimization or code generation, the compiler remains the primary cost. ForgeZero's advantage grows with high module fan-out, frequent incremental edits, and dependency-aware workflows.

### 1.4 Runtime stages

A normal build proceeds as follows:

1. CLI flags and runtime context are created.
2. TOML, YAML, FZP, and local configuration candidates are resolved.
3. Configuration values are expanded, merged, defaulted, and validated.
4. Toolchain, target, isolation, cache, and concurrency state are attached to the context.
5. Custom tasks and build rules are scheduled by stage and dependency graph.
6. Sources and generated include trees are discovered.
7. Source metadata and cache identity are evaluated.
8. Changed sources are sent to assembler/compiler backends.
9. Objects are restored from or stored in object caches where valid.
10. Linker or archive actions produce the requested artifact.
11. Hooks, reports, SBOM, audit, verification, sealing, and watcher actions run according to policy.

### 1.5 Memory ownership

The builder owns scheduling state, cache indexes, source metadata, object paths, and task fingerprints. Toolchain processes own their internal compiler memory. Mapped source buffers are valid only for the lifetime of their mapping and are unmapped after hashing. A `BoomBoomHasher` is sequentially owned by one update stream; concurrent calls on one hasher are not a supported contract.

## 2. Performance-critical implementation

### 2.1 BoomBoom hashing

BoomBoom is a keyed BLAKE3-derived incremental source hash used for source-cache identity. It is not a replacement for the project's stable BLAKE3 manifest or SBOM contracts.

The context key is derived from a domain separator, compiler identity, compiler version/profile, target, and ordered flags:

$$
K_c = BLAKE3_{keyed}(K_0, domain \parallel compiler \parallel version \parallel target \parallel flags)
$$

Each logical source region is hashed with a domain-separated leaf header containing its kind and index. Internal Merkle nodes use a separate domain-separated pair header.

#### Lexical logical chunking

The scanner is lexical rather than semantic. It does not construct an AST and does not prove compiler equivalence. It tracks:

- line comments;
- block comments;
- quoted strings and escape sequences;
- brace depth;
- directive lines beginning at logical line start;
- top-level statements terminated by semicolons;
- residual code regions.

Chunks are represented by `[Start, End)` offsets, a kind tag, and a 32-byte digest. Leading and trailing lexical whitespace is removed from the region before hashing. A formatting change can therefore change a leaf even when compiler semantics are unchanged.

#### Incremental state

The hasher retains:

- prior chunk boundaries and leaf digests;
- a previous input snapshot;
- changed chunk indexes;
- a flat Merkle tree;
- reusable worker jobs and worker state.

If a candidate chunk retains the same range, kind, and bytes as the previous snapshot, its digest is reused. The initial build or a capacity growth may allocate. The warmed steady-state path is designed for zero allocations and is enforced by tests for the parallel update path.

#### Flat Merkle tree

Leaves occupy a heap-style array beginning at the next power-of-two base. The final leaf is duplicated to fill incomplete levels. A full tree construction is $O(n)$ leaf placement plus parent construction. A stable tree updates changed leaves and recomputes parent paths; each changed path is $O(\log n)$, although the current implementation retains conservative leaf comparison in portions of the update path to preserve allocation and correctness behavior across structural changes.

The tree is capacity-reused. When capacity is insufficient, storage grows geometrically; when the required tree fits, the existing backing array is resliced.

#### Parallel leaf path

More than four changed leaves use a persistent worker set. Work publication and completion use a mutex/condition-variable protocol. Workers sleep while no batch is available and wake on a condition signal; they do not consume CPU through an atomic polling loop. `Close` marks the pool closed, broadcasts the condition, and waits for all workers to exit.

Workers call `runtime.LockOSThread` during their lifetime. This constrains Go scheduler migration for the worker goroutine, but it is not a guarantee of Linux CPU affinity or physical-core pinning. The operating system remains responsible for final CPU placement.

#### AMD64 primitives

On amd64, the low-level utility layer provides Plan 9 assembly primitives:

- vector equality uses unaligned 128-bit loads, `PCMPEQB`, and `PMOVMSKB`, followed by scalar tail comparison;
- hash-pair copying uses four 128-bit unaligned moves;
- delimiter search compares 16-byte blocks against repeated delimiter masks and finds the first set bit with `BSFQ`.

The Go wrappers retain length checks, zero-length handling, tail handling, and non-amd64 fallbacks. The delimiter primitive is exposed as a safe primitive; it is not used to skip arbitrary lexer regions because strings and comments require stateful interpretation.

The BLAKE3 package retains responsibility for CPU feature dispatch. ForgeZero does not force AVX-512 through an unsupported private backend.

### 2.2 File I/O and mapping

BoomBoom uses a hybrid path:

- empty files use the empty hash path;
- files smaller than 16 KiB use direct `os.ReadFile`, avoiding mapping setup and teardown overhead;
- larger files use platform mapping where available;
- Linux mappings use `madvise(MADV_WILLNEED)`;
- failed mappings fall back to io_uring when enabled and then to ordinary file reads;
- unsupported platforms retain the portable read fallback.

The 16 KiB threshold is an empirical policy, not a correctness boundary. It avoids paying virtual-memory-area and mapping syscall overhead when the complete input is cheaper to read directly. `MADV_WILLNEED` is a prefetch hint, not a guarantee that all pages are resident before hashing.

### 2.3 Configuration cache

Configuration has two cache layers:

1. An in-process cache indexed by normalized absolute path, source size, and nanosecond modification time. Returned configurations are deep clones so caller mutations do not alter cached state.
2. A `.fzcfg` sidecar containing a magic value, source modification time, source size, and a serialized normalized configuration payload. The sidecar is accepted only when metadata matches. It is written through a temporary file followed by rename.

The sidecar is a safe serialized representation, not a raw memory cast. Go maps, slices, strings, and interface-like runtime descriptors contain process-local pointers and cannot be persisted as a portable `unsafe` image. Configurations containing include or variable state are not eligible for the disk sidecar path when their values could be invalidated by external dependency state.

### 2.4 Process execution

Command execution is centralized through the utility command builder. It validates executable and argument policy, resolves the executable, applies execution root and environment policy, attaches output streams, and honors context cancellation.

ForgeZero does not force `vfork` or `CLONE_VM|CLONE_VFORK` globally. Those flags are unsafe as a general policy in a multithreaded Go process because child setup can touch runtime-managed state and suspend the parent while shared address space remains active. Process creation remains delegated to Go's supported `os/exec` implementation and platform runtime.

### 2.5 Compiler and assembler dispatch

C-family compiler invocations include `-pipe` where the selected compiler driver accepts it. PCH generation uses the same policy. Linker-driver paths also include `-pipe`; direct NASM, FASM, GAS, and `go tool asm` invocations do not receive compiler-driver-only flags.

## 3. TOML configuration specification

TOML is the preferred configuration format. YAML remains supported for compatibility and emits a deprecation warning. Configuration is parsed, includes are resolved, variables are expanded, defaults are applied, and the resulting state is validated before entering the builder.

### 3.1 Scalar build fields

| Field                 | Type             | Meaning                                                                               |
| --------------------- | ---------------- | ------------------------------------------------------------------------------------- |
| `name`                | string           | Logical project name.                                                                 |
| `profile`             | string           | Execution profile. Supported values include `balanced`, `powered`, and `performance`. |
| `target`              | string           | Target triple or target identifier.                                                   |
| `sysroot`             | string           | Compiler/linker sysroot.                                                              |
| `source_dir`          | string           | Primary source directory.                                                             |
| `source_dirs`         | array of strings | Multiple source roots.                                                                |
| `source_file`         | string           | Single source file mode.                                                              |
| `source_files`        | array of strings | Explicit source set.                                                                  |
| `output`              | string           | Final output path.                                                                    |
| `out_obj`             | string           | Explicit object output where applicable.                                              |
| `mode`                | string           | `auto`, `c`, or `raw`.                                                                |
| `toolchain`           | string           | `auto`, `zig`, `fasm`, `nasm`, `gas`, `gcc`, `clang`, or `ld`.                        |
| `linker`              | string           | Linker selection or override.                                                         |
| `debug`               | boolean          | Request debug information.                                                            |
| `verbose`             | boolean          | Emit commands and operational diagnostics.                                            |
| `keep_obj`            | boolean          | Retain intermediate objects.                                                          |
| `no_cache`            | boolean          | Disable cache use and force cache mode off.                                           |
| `cache_mode`          | string           | `disk`, `ram`, or `off`.                                                              |
| `cache_ram_mb`        | integer          | RAM object-cache capacity in MiB.                                                     |
| `optimization_level`  | integer          | Project optimization policy value.                                                    |
| `cpu_target`          | string           | CPU-specific target selection.                                                        |
| `auto_build_deps`     | boolean          | Enable dependency project builds.                                                     |
| `parse_makefile`      | boolean          | Permit Makefile-derived configuration discovery.                                      |
| `config_only`         | boolean          | Treat configuration as the primary build description.                                 |
| `ignore_file`         | string           | Ignore-rule file, defaulting to `.fzignore`.                                          |
| `deterministic_strip` | boolean          | Request deterministic stripping behavior where supported.                             |

### 3.2 Compiler and concurrency fields

```toml
[compiler]
path = "/usr/bin/gcc"

[concurrency]
workers = 16
pin = false
pin_to = ["0", "1"]
```

`compiler.path` overrides the selected compiler executable. `concurrency.workers` controls scheduler width. `concurrency.pin` and `pin_to` express project policy for concurrency placement; they do not alone guarantee kernel CPU affinity.

`instruction_sets` is an ordered array of instruction-set policy values. It participates in build context identity and validation.

### 3.3 Source selection and flags

- `exclude`: source and directory exclusion patterns.
- `include`: additional source or include selection patterns.
- `scripts`: project-level scripts.
- `libs`: libraries passed to the linker.
- `[flags].asm`: assembler flags.
- `[flags].cc`: C/C++ compiler flags.
- `[flags].ld`: linker flags.

Flags are preserved as ordered lists. Changing an effective flag changes the source/build context identity where that context participates in hashing.

### 3.4 Toolchain options

```toml
[toolchain_opts]
search_priority = ["local", "system"]
env_allow = ["PATH", "CC"]
tool_paths = { gcc = "/usr/bin/gcc" }
```

`search_priority` controls executable discovery. `env_allow` limits environment values admitted into isolated toolchain execution. `tool_paths` maps tool names to explicit locations.

### 3.5 FZP preprocessing

```toml
[preprocess]
enabled = true
inputs = ["config.h.in"]
outputs = ["generated/config.h"]
defines = { FEATURE_X = "1" }
```

FZP is a project-owned directive processor. It supports define/undef operations, conditional branches, `defined(...)`, quoted project includes, include-cycle detection, and allowed-path policy. System angle-bracket includes remain compiler responsibilities.

### 3.6 Variables

```toml
[variables]
BUILD_KIND = "release"
```

Variables may be referenced in supported scalar, array, map, task, rule, hook, and preprocessing fields. Expansion occurs before validation. Environment-derived values must be treated as part of reproducibility policy because they can alter paths, commands, or tool selection.

### 3.7 Hooks

```toml
[hooks]
pre_build = [{ cmd = "./prepare.sh", critical = true }]
on_failure = "./collect-failure.sh"
```

Pre-build hooks run before the build body. A critical hook failure aborts the build. `on_failure` is invoked when the outer build fails and is intentionally executed with a background context.

### 3.8 Build rules

```toml
[[build_rules]]
name = "generate"
action = "python3 generator.py $in $out"
inputs = ["schema.json"]
outputs = ["generated.c"]
depfile = "generated.d"
```

Build rules form an output ownership graph. Duplicate outputs are invalid. Rules are skipped when outputs exist and are newer than their dependencies, subject to dependency-file expansion. `$in`, `$out`, and `$depfile` are expanded into shell-safe command text.

### 3.9 Custom task engine

```toml
[[tasks]]
name = "codegen"
stage = "pre-build"
command = "python3 gen.py"
inputs = ["schema.json"]
outputs = ["generated.c"]
```

A task is eligible to run when any output is absent or its fingerprint marker does not match. The fingerprint includes input file identities and the command string. Input content changes therefore invalidate the task even if modification times are coarse or preserved.

Tasks are grouped into `pre-build`, `build`, and `post-build` stages. Tasks within a stage are represented as a DAG: an input matching another task's output establishes a dependency; unrelated tasks can run concurrently up to the configured worker count. Shell dispatch is selected once per process: POSIX systems use `sh -c`, Windows uses `cmd.exe /c`.

Task paths normalize separators for the host operating system. Shell quoting remains the shell's responsibility because commands may contain redirections, pipelines, quoting, and environment expansion.

### 3.10 ISO configuration

```toml
[iso]
enabled = true
source_dir = "iso"
output = "image.iso"
volume_id = "FORGEZERO"
boot_image = "boot.bin"
boot_catalog = "boot.cat"
boot_load_size = "4"
no_emul_boot = true
boot_info_table = true
joliet = true
rock_ridge = true
hybrid = false
custom_args = []
```

ISO fields control source tree, output image, volume metadata, boot image/catalog, El Torito behavior, filesystem extensions, hybrid mode, and extra tool arguments.

### 3.11 Dependency builds

`dep_build` contains `enabled`, `skip_tests`, `build_targets`, `outputs`, `include`, `environment`, `pre_build`, `post_build`, `exclude_files`, `only_files`, `steps`, and `step_sets`.

A `BuildStep` contains conditional fields (`if`, `elif`, `else`), failure-flow fields (`try`, `catch`, `finally`), grouping/staging fields (`group`, `stage`, `parallel`, `step_set`), substitutions (`with`), command forms (`command`, `run`), input/output lists, and `persistent` policy.

## 4. Gloria language specification

Gloria is an embedded experimental low-level language. Its design target is small controlled machine-code generation where a full C frontend, object format, libc startup, and general linker would be excessive.

### 4.1 Design constraints

Gloria currently prioritizes:

- direct emitter control;
- compact lexer and token state;
- explicit stack and register policy;
- raw binary generation;
- small code-generation experiments;
- ABI and instruction-encoding investigation.

It does not provide a production type system, memory safety, universal ABI portability, complete standard library, or general-purpose optimization pipeline.

### 4.2 Source model

The implemented syntax includes function declarations, local `let` bindings, `return`, register-oriented declarations, `if`, `while`, arithmetic/comparison forms, calls, and output/input builtins depending on emitter mode.

### 4.3 Lowering pipeline

The compiler performs:

1. lexical tokenization;
2. function grouping;
3. compact function representation;
4. direct instruction emission;
5. function-table construction;
6. placeholder creation for calls and branches;
7. relative relocation patching;
8. entry dispatch placement;
9. raw binary emission.

For a near call, the displacement is:

$$
disp_{32} = offset(target) - (offset(call) + 5)
$$

The result is normally a raw image rather than an ELF object. Consumers must therefore know the intended load address and execution environment.

### 4.4 Bare-metal output

The x86-64 emitter can target memory-mapped VGA text memory at `0xb8000` in the corresponding bare-metal output path. A character byte is paired with an attribute byte and written as a word while the output pointer advances by two bytes. This is hardware-specific behavior, not a portable userspace terminal contract.

## 5. Plan 9 assembly interoperability

ForgeZero detects Go Plan 9 assembly by source markers such as `TEXT ·name(SB)` and `#include "textflag.h"`. Detected `.s` and `.S` inputs use the Go host toolchain's `go tool asm` path.

The invocation supplies:

- `$GOROOT/pkg/include` for Go's standard assembler headers;
- `$GOROOT/src/runtime` for runtime-local assembly includes;
- the source path;
- the object output path.

This path is distinct from GAS and C-preprocessor assembly. A file that lacks Plan 9 markers remains on the ordinary `.s` or `.S` dispatch path. The target architecture is inherited from the active Go toolchain and target configuration; Plan 9 source is not automatically portable across architectures merely because the file extension is `.s`.

`textflag.h` and Plan 9 directives are interpreted by Go's assembler, not by NASM, FASM, or GNU `as`. Register names, symbol notation, ABI contracts, frame sizes, and stack maps must follow Go assembler conventions.

## 6. Security, reproducibility, and limitations

ForgeZero separates performance caching from trust verification. A cache hit means that the configured identity matched; it is not a cryptographic statement about the entire filesystem or toolchain unless the verification subsystem is also invoked.

Path validation, symlink policy, FZP include boundaries, isolated environments, SBOM generation, audit checks, manifest verification, and sealing are separate policy surfaces.

The following limitations are normative:

- metadata-first caches trade some adversarial-strength guarantees for speed;
- lexical chunking is not semantic hashing;
- first-use capacity growth may allocate;
- filesystem latency and page faults dominate small-hash microbenchmarks;
- `madvise` is advisory;
- Go scheduler thread locking is not equivalent to CPU affinity;
- compiler and linker availability remains an external prerequisite;
- raw binaries require a known execution environment;
- external toolchains retain responsibility for language and ABI correctness;
- disk configuration cache serialization is not an unsafe pointer cast and must remain portable across process lifetimes.

## 7. Operational verification

The minimum validation sequence is:

```text
go build ./cmd/fz
go test ./...
go test -race ./...
go test ./internal/utils -run '^$' -bench . -benchmem
```

Cross-compilation is compile-only unless a compatible runner exists. Runtime tests must execute on the target operating system and architecture. Performance claims must record source snapshot, CPU model, governor, storage medium, toolchain versions, cache state, worker count, target, and optimization flags.
