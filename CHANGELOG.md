# CHANGELOG

## [5.3.1] – 2026-07-07

## [Unreleased]

### Added
- **DAG‑based parallel task scheduler** (`scheduler`) – enables efficient, dependency‑aware parallel execution of build tasks.  
  ([f6dbe36](https://github.com/forgezero-cli/ForgeZero/commit/f6dbe36), [98ab763](https://github.com/forgezero-cli/ForgeZero/commit/98ab763), [159f241](https://github.com/forgezero-cli/ForgeZero/commit/159f241))
- **Precompiled header (PCH) support** – generation, caching, and integration into the assembly process for faster compilation.  
  ([8011a5a](https://github.com/forgezero-cli/ForgeZero/commit/8011a5a), [0c210c9](https://github.com/forgezero-cli/ForgeZero/commit/0c210c9), [51840fd](https://github.com/forgezero-cli/ForgeZero/commit/51840fd))
- **Persistent hash cache** for source files using BLAKE3, with refresh logic to avoid redundant work.  
  ([df563f5](https://github.com/forgezero-cli/ForgeZero/commit/df563f5), [3f57932](https://github.com/forgezero-cli/ForgeZero/commit/3f57932))
- **Dependency graph construction** for source files, enabling precise rebuild decisions.  
  ([6ec7cbd](https://github.com/forgezero-cli/ForgeZero/commit/6ec7cbd), [3d78363](https://github.com/forgezero-cli/ForgeZero/commit/3d78363))
- **Host system detection** and **resource detection** with job limiting, for optimal build configuration on any machine.  
  ([20a6b83](https://github.com/forgezero-cli/ForgeZero/commit/20a6b83), [b6c6a9d](https://github.com/forgezero-cli/ForgeZero/commit/b6c6a9d))
- **Utility modules** for parsing dependencies and building graphs, with accompanying unit tests.  
  ([3d78363](https://github.com/forgezero-cli/ForgeZero/commit/3d78363), [b3c0694](https://github.com/forgezero-cli/ForgeZero/commit/b3c0694))
- **NASM selection** – added `--use-nasm` flag to force using NASM instead of the internal assembler for `.asm` files, improving compatibility with existing NASM‑based projects and providing more flexibility. 
[1100f19](https://github.com/forgezero-cli/ForgeZero/commit/1100f19))
- **Plan9 cache support** – added cache implementation for Plan9 operating system.  
  ([8a94769](https://github.com/forgezero-cli/ForgeZero/commit/8a94769))
#### [NEW] 2026-07-08
- **Multi‑level action cache (L1/L2/L3)** – stores build rule outputs in a zero‑copy memory‑mapped cache with BLAKE3 keys, dramatically reducing rebuild times for expensive actions like `./configure`.  
  ([29ae2c8](https://github.com/forgezero-cli/ForgeZero/commit/29ae2c8))
- **Build rule execution system** – supports custom `build_rules` in config with `action`, `inputs`, `outputs`, `depfile`; DAG‑based ordering; variable expansion (`$in`, `$out`, `$depfile`); and `restat` behaviour.  
  ([ef46b6f](https://github.com/forgezero-cli/ForgeZero/commit/ef46b6f), [96c5109](https://github.com/forgezero-cli/ForgeZero/commit/96c5109))
- **Internal logging package** – zero‑allocation buffered logger with `Debug`, `Info`, `Error` methods for high‑performance build logs.  
  ([f8d9ae](https://github.com/forgezero-cli/ForgeZero/commit/f8d9ae))
- **x86‑64 assembly reference** – comprehensive NASM syntax guide covering registers, instructions, syscalls, SIMD, and calling conventions.  
  ([85b5f58](https://github.com/forgezero-cli/ForgeZero/commit/85b5f58))
- **FZASM specification files** – detailed documentation for the internal assembler architecture, opcodes, and encoder design.  
  ([e87c7f8](https://github.com/forgezero-cli/ForgeZero/commit/e87c7f8))
- **NUMA‑aware atomic counters** – sharded counters by NUMA node for cache‑hit/miss tracking with reduced contention.  
  ([cf2655a](https://github.com/forgezero-cli/ForgeZero/commit/cf2655a))
- **Lock‑free queue implementation** – Michael‑Scott style queue with sequence‑based ring buffer for the scheduler.  
  ([7fe6a1d](https://github.com/forgezero-cli/ForgeZero/commit/7fe6a1d))
- **Zero‑alloc encoder foundation** – opcode constants and encoder framework for future instruction expansion.  
  ([6747f47](https://github.com/forgezero-cli/ForgeZero/commit/6747f47), [ed6a528](https://github.com/forgezero-cli/ForgeZero/commit/ed6a528))
- **Full Windows support for `seal`** – implemented platform‑specific machine ID retrieval via `GetVolumeInformationW`, file sealing with `FILE_ATTRIBUTE_READONLY`, and verification using WinAPI, replacing the stub with a fully functional Windows backend.  
  ([b3fb575](https://github.com/forgezero-cli/ForgeZero/commit/b3fb575))
- **Platform‑agnostic mmap wrappers** – introduced `mmapFile` and `munmapFile` helpers with build tags for Linux (`mmap_linux.go`) and other platforms (`mmap_other.go`), allowing the builder and action cache to use memory‑mapped I/O without direct `syscall` dependencies, improving cross‑platform compatibility.  
  ([503d7b4](https://github.com/forgezero-cli/ForgeZero/commit/503d7b4), [a4a37ea](https://github.com/forgezero-cli/ForgeZero/commit/a4a37ea))


### Changed
- **Overhauled core build engine** – now uses DAG scheduling and the new hash cache for a more reliable and faster build.  
  ([5dd68e4](https://github.com/forgezero-cli/ForgeZero/commit/5dd68e4))
- **Linker logic** – response‑file creation extracted to a dedicated module; inline logic removed for better separation of concerns.  
  ([e6a4211](https://github.com/forgezero-cli/ForgeZero/commit/e6a4211), [06a0af3](https://github.com/forgezero-cli/ForgeZero/commit/06a0af3))
- **PCH integration** – assembly step now fully delegates to the new PCH module.  
  ([0c210c9](https://github.com/forgezero-cli/ForgeZero/commit/0c210c9))
- **Optimized internal x86 assembler** – removed string conversions, eliminated slice allocations in memory operand parsing, and reduced `append` calls, resulting in faster assembly generation without any API changes.  
([976f09c](https://github.com/forgezero-cli/ForgeZero/commit/976f09c))
- **Optimized linker symbol parsing** – replaced string‑based parsing with byte‑level operations to reduce allocations and improve performance for large object files.  
  ([f8ed04b](https://github.com/forgezero-cli/ForgeZero/commit/f8ed04b))
- **Code style cleanup** – removed extra blank lines in `cache.go` and `symbols.go`.  
  ([5c1cea5](https://github.com/forgezero-cli/ForgeZero/commit/5c1cea5), [f293ec0](https://github.com/forgezero-cli/ForgeZero/commit/f293ec0))
### [NEW] 2026-07-8
- **Scheduler overhaul** – replaced lock‑based queues with lock‑free ring queues, added worker‑local priority queues (8 levels), work‑stealing, and persistent worker goroutines; benchmarks show **0 allocs/op** in hot path.  
  ([bbdc2a2](https://github.com/forgezero-cli/ForgeZero/commit/bbdc2a2), [606913e](https://github.com/forgezero-cli/ForgeZero/commit/606913e), [8875114](https://github.com/forgezero-cli/ForgeZero/commit/8875114), [102a1d0](https://github.com/forgezero-cli/ForgeZero/commit/102a1d0), [e8905fb](https://github.com/forgezero-cli/ForgeZero/commit/e8905fb))
- **Builder cache and action cache** – replaced direct `syscall.Mmap`/`Munmap` calls with the new cross‑platform wrappers, enabling zero‑copy caching on Linux while gracefully falling back on other OSes.  
  ([f9f04ec](https://github.com/forgezero-cli/ForgeZero/commit/f9f04ec), [1211ba8](https://github.com/forgezero-cli/ForgeZero/commit/1211ba8))
- **Seal package** – unified `MachineID` signature across all platforms, added full Windows implementation, and maintained stubs for other non‑Linux OSes.  
  ([5006a7c](https://github.com/forgezero-cli/ForgeZero/commit/5006a7c), [4981d9e](https://github.com/forgezero-cli/ForgeZero/commit/4981d9e), [b3fb575](https://github.com/forgezero-cli/ForgeZero/commit/b3fb575))
- **RAM cache rework** – switched from mutex‑protected map to `sync.Map`, added `syscall.Mmap` for zero‑copy object storage with safe `Munmap` on eviction.  
  ([1813d6e](https://github.com/forgezero-cli/ForgeZero/commit/1813d6e), [a293d58](https://github.com/forgezero-cli/ForgeZero/commit/a293d58))
- **Assembler optimisations** – register parser rewritten to return primitives instead of allocating structs; `fmt.Errorf` replaced with pooled buffer builder; PCH mutex replaced with `sync.Map`.  
  ([ad1a260](https://github.com/forgezero-cli/ForgeZero/commit/ad1a260), [2539dbc](https://github.com/forgezero-cli/ForgeZero/commit/2539dbc), [43dfc3e](https://github.com/forgezero-cli/ForgeZero/commit/43dfc3e))
- **Utils/VFS refactor** – replaced `RWMutex` with `atomic.Value` in `vfs.go`; `RunCommand` now uses pipe + single reader goroutine instead of mutex‑protected buffer writer.  
  ([ded5a6c](https://github.com/forgezero-cli/ForgeZero/commit/ded5a6c), [8c9ec50](https://github.com/forgezero-cli/ForgeZero/commit/8c9ec50))
- **Seal package** – replaced `allowed` map with `sync.Map`, removed `journalMu`, and used atomic circular buffer writes.  
  ([fab854c](https://github.com/forgezero-cli/ForgeZero/commit/fab854c), [aa35066](https://github.com/forgezero-cli/ForgeZero/commit/aa35066))
- **Linker target info** – replaced `RWMutex` with `atomic.Value` for lock‑free target feature detection.  
  ([b6e7442](https://github.com/forgezero-cli/ForgeZero/commit/b6e7442))

### Fixed
- **Comprehensive test coverage** for dependency parsing, graph building, DAGScheduler, and PCH integration hooks.  
  ([b3c0694](https://github.com/forgezero-cli/ForgeZero/commit/b3c0694), [98ab763](https://github.com/forgezero-cli/ForgeZero/commit/98ab763), [51840fd](https://github.com/forgezero-cli/ForgeZero/commit/51840fd))
- **Shell command validation** – allowed `-c` (Unix) and `/C` (Windows) arguments in `RunCommand` without strict validation, enabling complex shell payloads.  
  ([0ffe97e](https://github.com/forgezero-cli/ForgeZero/commit/0ffe97e))
- **Build rule depfile/restat tests** – added comprehensive test coverage for dependency file parsing and incremental rebuild decisions.  
  ([96c5109](https://github.com/forgezero-cli/ForgeZero/commit/96c5109))
- **Seal build tags** – added proper `//go:build linux` to `seal.go` and `//go:build !linux` to `seal_stub.go`, resolving duplicate declaration errors in CI.  
  ([73014f9](https://github.com/forgezero-cli/ForgeZero/commit/73014f9))
- **Cross‑platform compilation for Windows** – fixed undefined `syscall.Mmap`, `syscall.Munmap`, and `syscall.PROT_READ` errors by using platform‑specific wrappers, allowing the project to build successfully on Windows.  
  ([503d7b4](https://github.com/forgezero-cli/ForgeZero/commit/503d7b4), [a4a37ea](https://github.com/forgezero-cli/ForgeZero/commit/a4a37ea), [f9f04ec](https://github.com/forgezero-cli/ForgeZero/commit/f9f04ec), [1211ba8](https://github.com/forgezero-cli/ForgeZero/commit/1211ba8))

### Build
- Bumped core version from **v5.3.0** to **v5.3.1**.  
  ([67d023a](https://github.com/forgezero-cli/ForgeZero/commit/67d023a))

---

### Full Commit List (in chronological order)

```text
0c210c9 assembler: integrate precompiled header (PCH) logic into compileC
5dd68e4 builder: overhaul core build engine with DAG scheduling and hash caching
06a0af3 linker: remove inline response-file logic, delegate to separate module
8011a5a assembler: implement precompiled header (PCH) generation and caching
6ec7cbd builder: add dependency graph construction for source files
3f57932 builder: implement persistent hash cache for source files
b6c6a9d builder: add system resource detection and job limiting
51840fd builder: add unit tests for PCH integration hooks
20a6b83 builder: add host system detection for optimal build configuration
159f241 scheduler: add code generation utilities for task scheduling
67d023a build: bump core version from v5.3.0 to v5.3.1
f6dbe36 scheduler: implement DAG‑based parallel task scheduler
98ab763 scheduler: add comprehensive tests for DAGScheduler
e6a4211 linker: extract response‑file creation logic to dedicated module
3d78363 utils: add dependency parsing and graph building utilities
b3c0694 utils: add unit tests for dependency parsing and graph building
df563f5 builder: implement source file hashing and refresh logic using BLAKE3
1100f19 assembler: add --use-asm flag to select NASM over internal assembler and add units tests for NASM selection logic 
976f09c assembler: optimize x86 backend without API changes
f8ed04b linker: optimize symbol parsing with byte-level operations
5c1cea5 style: remove extra blank lines in cache.go
f293ec0 style: remove extra blank lines in symbols.go
8a94769 builder: add Plan9 cache implementation
7fe6a1d feat(scheduler): add lock-free queue implementation
f69f1ee test(scheduler): add stress test for 1000 tasks
cf2655a feat(utils): add NUMA-aware atomic counters
73456ad feat(config): add BuildRule struct and integrate into config validation and expansion
0ffe97e fix(utils): allow shell -c and /C arguments in RunCommand validation
e87c7f8 docs: add specification files for FZASM and scheduler
85b5f58 docs: add x86-64 assembly reference (NASM syntax) for developers
29ae2c8 feat(builder): add multi-level action cache (L1/L2/L3) for build rules
ef46b6f feat(builder): implement build rule execution with DAG and depfile support
96c5109 test(builder): add tests for build rules graph and depfile/restat behavior
f8d9ae feat(logger): add internal logging package with buffered output