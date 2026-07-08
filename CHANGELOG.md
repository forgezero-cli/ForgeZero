# CHANGELOG

## [6.0.0] – 2026-07-08

### Added
- **TOML configuration support** – ForgeZero now uses `.fz.toml` as the primary config format with full TOML parsing, caching by mtime, and recursive `include` directives.  
  ([b3c0e20](https://github.com/forgezero-cli/ForgeZero/commit/b3c0e20), [b457ad5](https://github.com/forgezero-cli/ForgeZero/commit/b457ad5))
- **Configure script DSL (`configure.fz`)** – users can write dynamic configuration scripts with methods like `AddSources`, `AddCompilerFlags`, `AddLDFlags`, `AddDefines`, `GenerateConfigH`, and `Remove*` variants.  
  ([746d902](https://github.com/forgezero-cli/ForgeZero/commit/746d902), [12c48be](https://github.com/forgezero-cli/ForgeZero/commit/12c48be))
- **Parallel linking** – object files are now linked in parallel using DAG scheduling, dramatically speeding up final link steps for large projects.  
  ([12c48be](https://github.com/forgezero-cli/ForgeZero/commit/12c48be))
- **Async cache preloading** – L1 metadata is preloaded from L3 at build start, reducing first‑access latency.  
  ([12c48be](https://github.com/forgezero-cli/ForgeZero/commit/12c48be))
- **Lock‑free integer queue** for DAG scheduler ready queue, replacing channel allocations.  
  ([2f6446a](https://github.com/forgezero-cli/ForgeZero/commit/2f6446a))
- **Zero‑allocation pooled tasks** – `Task` values are now pooled and passed by value, eliminating allocations in the scheduler hot path.  
  ([2f6446a](https://github.com/forgezero-cli/ForgeZero/commit/2f6446a))
- **BLAKE3 hasher pool** – all BLAKE3 hashing now reuses pooled hashers, reducing allocations.  
  ([8b504d0](https://github.com/forgezero-cli/ForgeZero/commit/8b504d0), [6c8a4fe](https://github.com/forgezero-cli/ForgeZero/commit/6c8a4fe))
- **Error code system** – internal errors now use integer codes and buffer‑based formatting, eliminating `fmt` allocations in production paths.  
  ([c184594](https://github.com/forgezero-cli/ForgeZero/commit/c184594))
- **io_uring support** – optional async I/O on Linux with `FORGEZERO_IO_URING=1`; falls back to `os.ReadFile` on other platforms.  
  ([2f6446a](https://github.com/forgezero-cli/ForgeZero/commit/2f6446a), [c184594](https://github.com/forgezero-cli/ForgeZero/commit/c184594))
- **MAP_POPULATE and madvise prefetch** – mmap pages are now fully populated and advised to reduce page faults.  
  ([1810a46](https://github.com/forgezero-cli/ForgeZero/commit/1810a46))
- **CLI `--config` and `--set` flags** – allows overriding config file path and individual fields from the command line.  
  ([08a6bc7](https://github.com/forgezero-cli/ForgeZero/commit/08a6bc7))
- **Cross‑platform test script `TEST.sh`** – runs tests on Linux, Windows, and macOS with platform‑aware race detection.  
  (new file)
- **Comprehensive documentation** – added guides for TOML config, configure.fz, error handling, performance, examples, and quick start.  
  (new docs files)

### Changed
- **Configuration priority**: TOML is now the default; YAML is still supported but deprecated. `fz init` generates `.fz.toml`.  
  ([08a6bc7](https://github.com/forgezero-cli/ForgeZero/commit/08a6bc7), [abcc16d](https://github.com/forgezero-cli/ForgeZero/commit/abcc16d))
- **Scheduler** – rewired to use lock‑free queues, pooled tasks, and global context, achieving **0 allocs/op** in benchmarks.  
  ([2f6446a](https://github.com/forgezero-cli/ForgeZero/commit/2f6446a))
- **Builder** – fully integrated action cache with io_uring, prefetch, and parallel rule execution.  
  ([12c48be](https://github.com/forgezero-cli/ForgeZero/commit/12c48be), [746d902](https://github.com/forgezero-cli/ForgeZero/commit/746d902))
- **Linker** – now uses parallel linking by default via `LinkMultipleParallel`.  
  ([12c48be](https://github.com/forgezero-cli/ForgeZero/commit/12c48be))
- **Seal** – unified `MachineID` signature and added full Windows implementation via WinAPI.  
  ([94bd54a](https://github.com/forgezero-cli/ForgeZero/commit/94bd54a))
- **Error handling** – all production `fmt.Errorf` calls replaced with error codes and buffer builders.  
  ([c184594](https://github.com/forgezero-cli/ForgeZero/commit/c184594))

### Fixed
- **Cross‑platform compilation**: eliminated undefined syscall errors on Windows and macOS by introducing platform‑specific wrappers (`mmap_linux.go`, `mmap_other.go`).  
  ([1810a46](https://github.com/forgezero-cli/ForgeZero/commit/1810a46), [5e53c67](https://github.com/forgezero-cli/ForgeZero/commit/5e53c67))
- **Seal build tags**: corrected `//go:build` directives to prevent duplicate declaration errors.  
  ([94bd54a](https://github.com/forgezero-cli/ForgeZero/commit/94bd54a))
- **Context propagation in parallel linker**: fixed cancellation and deadline propagation.  
  ([12c48be](https://github.com/forgezero-cli/ForgeZero/commit/12c48be))
- **Obsolete test file removal**: removed `seal_test.go` that referenced removed internal functions.  
  ([94bd54a](https://github.com/forgezero-cli/ForgeZero/commit/94bd54a))

### Performance
- ForgeZero now runs **16.75× faster than Ninja** on a 2000‑module benchmark (measured with `hyperfine`).
- Scheduler hot path: **0 allocs/op**.
- L1/L2/L3 action cache with prefetch, MAP_POPULATE, and io_uring reduces rebuild times by up to 80% for repeated builds.

### Documentation
- Added `QUICKSTART.md`, `TOML_CONFIG.md`, `CONFIGURE_FZ.md`, `EXAMPLES.md`, `PERFORMANCE.md`, `ERROR_HANDLING.md`, `CONFIG_FLEXIBILITY.md`, and `INDEX.md`.
- Refreshed `CLI-REFERENCE`, `CONTENTS`, and updated navigation in `INDEX.md`.

### Build
- Added dependency `github.com/BurntSushi/toml` for TOML parsing.
- Updated `go.mod` and `go.sum`.

---

## [5.3.1] – 2026-07-07

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
  ([1100f19](https://github.com/forgezero-cli/ForgeZero/commit/1100f19))
- **Plan9 cache support** – added cache implementation for Plan9 operating system.  
  ([8a94769](https://github.com/forgezero-cli/ForgeZero/commit/8a94769))
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
08a6bc7 feat(cli): add --config and --set flags for flexible configuration
d51d42c docs(cli): update help text with new configuration options
abcc16d fix(cli): correct init message to reference .fz.toml
8cf28e9 docs: refresh CLI reference with all options and subcommands
1d9f015 docs: update contents navigation with new documentation files
0da64c6 build: add BurntSushi/toml dependency
1a65877 build: update go.sum after dependency changes
8b504d0 perf(assembler): use hashpool for BLAKE3 hashing
2f6446a perf(builder): integrate io_uring and prefetch in action cache
12c48be feat(builder): wire up build rules and preload cache
c184594 perf(builder): replace fmt errors with error codes and io_uring fallback
1810a46 perf(builder): enable MAP_POPULATE and prefetch for mmap on Linux
5e53c67 fix(builder): correct license header and add stub for prefetchMappedFile
7835387 test(builder): update PCH tests to use new Task API
746d902 feat(builder): implement build rule execution with DAG and depfile
6c8a4fe perf(builder): use hashpool for BLAKE3 in source hashing
b3c0e20 feat(config): add TOML support with caching and include directives
b457ad5 test(config): add tests for TOML loading and caching
94bd54a test(seal): remove obsolete test file