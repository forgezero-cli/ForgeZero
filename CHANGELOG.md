# CHANGELOG

# UNRELEASED

## 2026-07-22

### Added

- **Custom work-stealing thread pool (`fo`)** – lock-free ring buffer per worker with work-stealing for load balancing, exponential backoff when idle, and global pool singleton. Replaces DAG scheduler in builder and linker for simpler parallelism.  
  ([08cb407](https://github.com/forgezero-cli/ForgeZero/commit/08cb407))
- **Diagnostic error system** – line-level error reporting with colored terminal output, source context rendering, and fix suggestions. Errors now include file path, line number, parameter name, and actionable hints.  
  ([184746d](https://github.com/forgezero-cli/ForgeZero/commit/184746d), [f9b7697](https://github.com/forgezero-cli/ForgeZero/commit/f9b7697))
- **Config validation for hardware settings** – new config fields: `compiler.path`, `cpu_target`, `instruction_sets`, `concurrency.workers`, `concurrency.pin`, `concurrency.pin_to`. Includes architecture-aware validation with `isSupportedTarget()`.  
  ([2ec9e53](https://github.com/forgezero-cli/ForgeZero/commit/2ec9e53))

### Changed

- **Builder parallel execution** – replaced DAG scheduler with `fo` worker pool for simpler, more reliable parallel compilation.  
  ([bec8e9a](https://github.com/forgezero-cli/ForgeZero/commit/bec8e9a))
- **Linker parallel execution** – replaced DAG scheduler with `fo` worker pool for parallel linking.  
  ([9037a00](https://github.com/forgezero-cli/ForgeZero/commit/9037a00))
- **Config error formatting** – enhanced `Error` struct with `Path`, `Line`, `Parameter`, and `Suggestion` fields for detailed, user-friendly error messages.  
  ([f9b7697](https://github.com/forgezero-cli/ForgeZero/commit/f9b7697))
- **CLI config loader** – integrated diagnostic error logging for config loading failures with detailed location info.  
  ([a8dd4e0](https://github.com/forgezero-cli/ForgeZero/commit/a8dd4e0))

### Fixed

- **Config validation** – added comprehensive validation for compiler, CPU target, instruction sets, and concurrency settings with proper error codes and suggestions.  
  ([2ec9e53](https://github.com/forgezero-cli/ForgeZero/commit/2ec9e53), [79a375b](https://github.com/forgezero-cli/ForgeZero/commit/79a375b))

### Performance

- **Worker pool** – zero-allocation task submission with lock-free ring buffers and work-stealing for optimal load balancing.

### Testing

- **Config validation** – added tests for compiler and hardware validation across different architectures.  
  ([79a375b](https://github.com/forgezero-cli/ForgeZero/commit/79a375b))
- **Error rendering** – added tests for line-level error diagnostic rendering.  
  ([184746d](https://github.com/forgezero-cli/ForgeZero/commit/184746d))
- **Pool** – comprehensive tests for the `fo` work-stealing thread pool.  
  ([08cb407](https://github.com/forgezero-cli/ForgeZero/commit/08cb407))

## [6.0.0] - 2026-07-18

### Added

- **AVX2 optimization for ExecRawXor** – SIMD-accelerated XOR operations using AVX2 when available, with optimized 64-bit chunk fallback for significant performance gains.  
  ([2ddf2a4](https://github.com/forgezero-cli/ForgeZero/commit/2ddf2a4))
- **Zero-allocation Makefile parser** – manual byte-level parsing with `unsafe` string conversion eliminating all allocations during Makefile variable extraction.  
  ([2ddf2a4](https://github.com/forgezero-cli/ForgeZero/commit/2ddf2a4))
- **Encoder capacity optimization** – `Reserve` method and grow logic with 4096-byte default buffer to reduce allocations in hot path.  
  ([1199697](https://github.com/forgezero-cli/ForgeZero/commit/1199697))

### Changed

- **Assembler byte operations** – replaced `append` calls with capacity-checked `AppendByte`/`AppendBytes` methods for zero-allocation emission.  
  ([98f993f](https://github.com/forgezero-cli/ForgeZero/commit/98f993f), [291e081](https://github.com/forgezero-cli/ForgeZero/commit/291e081), [aa4bcc5](https://github.com/forgezero-cli/ForgeZero/commit/aa4bcc5))
- **GOROOT detection** – replaced `runtime.GOROOT()` with `exec.Command("go", "env", "GOROOT")` for better cross-compilation support.  
  ([722c33f](https://github.com/forgezero-cli/ForgeZero/commit/722c33f))
- **Sync.Pool usage** – fixed `topoOrder` pool to use pointer to slice, preventing allocation issues.  
  ([e89fb9f](https://github.com/forgezero-cli/ForgeZero/commit/e89fb9f))

### Fixed

- **Error handling** – added comprehensive error handling for JSON encoding, file writes, cache operations, and mmap operations across all packages.  
  ([2257e55](https://github.com/forgezero-cli/ForgeZero/commit/2257e55), [14b2d8d](https://github.com/forgezero-cli/ForgeZero/commit/14b2d8d), [5face59](https://github.com/forgezero-cli/ForgeZero/commit/5face59), [942dda1](https://github.com/forgezero-cli/ForgeZero/commit/942dda1), [88ac473](https://github.com/forgezero-cli/ForgeZero/commit/88ac473), [304abdc](https://github.com/forgezero-cli/ForgeZero/commit/304abdc), [a615529](https://github.com/forgezero-cli/ForgeZero/commit/a615529))
- **Makefile parser** – infinite loop fixed with proper EOF handling in include scanning.  
  ([2ddf2a4](https://github.com/forgezero-cli/ForgeZero/commit/2ddf2a4))
- **FindExecutable test** – inverted logic corrected for proper absolute path assertion.  
  ([2ddf2a4](https://github.com/forgezero-cli/ForgeZero/commit/2ddf2a4))
- **Config test** – proper error handling for Load function return value.  
  ([372e4e5](https://github.com/forgezero-cli/ForgeZero/commit/372e4e5))

### Testing

- **stdio** – added output tests for write operations.  
  ([2ddf2a4](https://github.com/forgezero-cli/ForgeZero/commit/2ddf2a4))
- **assembler** – added stress tests and section helper functions.  
  ([2ddf2a4](https://github.com/forgezero-cli/ForgeZero/commit/2ddf2a4))
- **logger** – added logger tests.  
  ([2ddf2a4](https://github.com/forgezero-cli/ForgeZero/commit/2ddf2a4))
- **XOR** – added benchmark tests for AVX2 optimization.  
  ([2ddf2a4](https://github.com/forgezero-cli/ForgeZero/commit/2ddf2a4))

### Documentation

- **YOUTUBE.md** – added new video link showing Redis build process.  
  ([a4d2186](https://github.com/forgezero-cli/ForgeZero/commit/a4d2186))
- **cmodule** – marked as unsupported with notice.  
  ([e792c7d](https://github.com/forgezero-cli/ForgeZero/commit/e792c7d))

### Build

- Added `*.prof` to `.gitignore` for profiling files.  
  ([3f5cd01](https://github.com/forgezero-cli/ForgeZero/commit/3f5cd01))
- Replaced deprecated `ioutil.TempFile` with `os.CreateTemp`.  
  ([bf06eb8](https://github.com/forgezero-cli/ForgeZero/commit/bf06eb8))

## [6.0.0] - 2026-07-17

### Added

- **RAM cache capacity limits** – `cache_ram_mb` config option and `SetRAMCacheCapacityMB` function to control RAM cache size with atomic tracking and eviction.  
  ([efb1576](https://github.com/forgezero-cli/ForgeZero/commit/efb1576), [8c332d3](https://github.com/forgezero-cli/ForgeZero/commit/8c332d3))
- **Dependency build steps** – support for custom `steps` and `step_sets` in `[dep_build]` with conditional execution (`if`, `elif`, `else`), error handling (`try`, `catch`, `finally`), grouping (`group`, `stage`, `parallel`), and persistent outputs.  
  ([ba554a7](https://github.com/forgezero-cli/ForgeZero/commit/ba554a7), [ddbd2c6](https://github.com/forgezero-cli/ForgeZero/commit/dddb2c6))
- **Scheduler improvements** – condition variable for pending tasks to reduce busy-waiting, comprehensive stress tests, and proper error propagation that stops dependent tasks on failure.  
  ([96c87ea](https://github.com/forgezero-cli/ForgeZero/commit/96c87ea), [16d1d35](https://github.com/forgezero-cli/ForgeZero/commit/16d1d35), [d8530f9](https://github.com/forgezero-cli/ForgeZero/commit/d8530f9), [af4bb07](https://github.com/forgezero-cli/ForgeZero/commit/af4bb07))
- **Config validation** – comprehensive validation for source config, mode, profile, toolchain, isolation, cache mode, and build rules with proper error codes.  
  ([ddbd2c6](https://github.com/forgezero-cli/ForgeZero/commit/dddb2c6))
- **Config expansion optimization** – early exit when no variables require expansion.  
  ([ddbd2c6](https://github.com/forgezero-cli/ForgeZero/commit/dddb2c6))
- **Variable expansion optimization** – early return for strings without `$` to avoid unnecessary work.  
  ([0bc36aa](https://github.com/forgezero-cli/ForgeZero/commit/0bc36aa))

### Changed

- **Assembler** – flag initialization moved to `sync.Once` and `target` parameter added to eliminate data races; all platform detection functions now accept explicit `target` parameter.  
  ([5bc08ad](https://github.com/forgezero-cli/ForgeZero/commit/5bc08ad))
- **L1 cache** – replaced lock-free operations with explicit `sync.RWMutex` protection to eliminate data races.  
  ([a81ff24](https://github.com/forgezero-cli/ForgeZero/commit/a81ff24))
- **Scheduler** – improved node validation order to prevent leaks and reduce busy-waiting with condition variables.  
  ([af4bb07](https://github.com/forgezero-cli/ForgeZero/commit/af4bb07), [96c87ea](https://github.com/forgezero-cli/ForgeZero/commit/96c87ea))
- **Config cache** – replaced `sync.Map` with `RWMutex`-protected map for better performance.  
  ([7050146](https://github.com/forgezero-cli/ForgeZero/commit/7050146))
- **TOML parsing** – optimized using unsafe string conversion to reduce allocations.  
  ([73d0763](https://github.com/forgezero-cli/ForgeZero/commit/73d0763))
- **Custom steps execution** – added proper integration with dependency builder.  
  ([85b526f](https://github.com/forgezero-cli/ForgeZero/commit/85b526f))

### Fixed

- **Data races** – fixed all data races in assembler (global `Target` writes) and builder L1 cache (concurrent `l1Store` writes).  
  ([5bc08ad](https://github.com/forgezero-cli/ForgeZero/commit/5bc08ad), [a81ff24](https://github.com/forgezero-cli/ForgeZero/commit/a81ff24))
- **Scheduler node leak** – nodes are now validated before allocation to prevent leaks on invalid dependencies.  
  ([af4bb07](https://github.com/forgezero-cli/ForgeZero/commit/af4bb07))
- **Scheduler error handling** – dependent tasks now correctly skip execution when a dependency fails.  
  ([d8530f9](https://github.com/forgezero-cli/ForgeZero/commit/d8530f9))
- **Config include processing** – fixed includes to allow relative paths and proper merging of config values.  
  ([ddbd2c6](https://github.com/forgezero-cli/ForgeZero/commit/dddb2c6))
- **RAM cache size tracking** – proper atomic tracking of used bytes with rollback on errors.  
  ([efb1576](https://github.com/forgezero-cli/ForgeZero/commit/efb1576))

### Performance

- **Variable expansion** – early return for strings without `$` reduces overhead.  
  ([0bc36aa](https://github.com/forgezero-cli/ForgeZero/commit/0bc36aa))
- **Config parsing** – optimized TOML parsing with unsafe string conversion.  
  ([73d0763](https://github.com/forgezero-cli/ForgeZero/commit/73d0763))
- **Scheduler** – reduced busy-waiting with condition variables, improved node validation.  
  ([96c87ea](https://github.com/forgezero-cli/ForgeZero/commit/96c87ea), [af4bb07](https://github.com/forgezero-cli/ForgeZero/commit/af4bb07))

### Testing

- **Builder** – added comprehensive tests for autodeps, dependency build steps (unit, integration, parallel, TOML), and graph resolution.  
  ([4758fc6](https://github.com/forgezero-cli/ForgeZero/commit/4758fc6), [6f033d8](https://github.com/forgezero-cli/ForgeZero/commit/6f033d8), [87591cd](https://github.com/forgezero-cli/ForgeZero/commit/87591cd), [191ef17](https://github.com/forgezero-cli/ForgeZero/commit/191ef17), [e05874d](https://github.com/forgezero-cli/ForgeZero/commit/e05874d), [c329d22](https://github.com/forgezero-cli/ForgeZero/commit/c329d22), [10319cf](https://github.com/forgezero-cli/ForgeZero/commit/10319cf))
- **Scheduler** – added DAG stress test with 128 nodes and tests for invalid dependencies and error propagation.  
  ([16d1d35](https://github.com/forgezero-cli/ForgeZero/commit/16d1d35), [d8530f9](https://github.com/forgezero-cli/ForgeZero/commit/d8530f9))
- **Linker** – added object linking tests.  
  ([81d9890](https://github.com/forgezero-cli/ForgeZero/commit/81d9890))
- **Config** – added tests for CacheRAMMB override and merge.  
  ([c643ff4](https://github.com/forgezero-cli/ForgeZero/commit/c643ff4))

## [6.0.0] - 2026-07-16

### Added

- **`--config-only` CLI flag** – restricts build configuration to explicit `fz.toml`/`configure.fz` settings, skipping auto-discovery and Makefile parsing for better control in complex build environments.  
  ([5d2fed1](https://github.com/forgezero-cli/ForgeZero/commit/5d2fed1), [5d5519c](https://github.com/forgezero-cli/ForgeZero/commit/5d5519c), [0569c1d](https://github.com/forgezero-cli/ForgeZero/commit/0569c1d))
- **`configure.fz` support** – build rules can now be defined via `configure.fz` files with `OUTPUT`, `ACTION`, and `MAKECMD` variables, providing a lightweight alternative to full TOML configuration.  
  ([0569c1d](https://github.com/forgezero-cli/ForgeZero/commit/0569c1d))
- **Makefile parsing** – experimental Makefile parsing for source discovery via `--parse-makefile` flag, enabling incremental migration from Makefile-based projects.  
  ([b0bf0bd](https://github.com/forgezero-cli/ForgeZero/commit/b0bf0bd))
- **`FZ_COMPILE_WORKER_MEM_MB` environment variable** – allows overriding the compile worker memory limit (default 1024MB) for fine-tuned resource control.  
  ([0215d95](https://github.com/forgezero-cli/ForgeZero/commit/0215d95))

### Changed

- **Config propagation** – `ParseMakefile` setting from config now properly propagates to CLI flags during config loading.  
  ([f668ac6](https://github.com/forgezero-cli/ForgeZero/commit/f668ac6))
- **Symlink handling** – source hash calculation now skips symlinks that point to directories, preventing hash errors.  
  ([967e5e3](https://github.com/forgezero-cli/ForgeZero/commit/967e5e3))
- **State display** – removed duplicate `Debug` output line in `cmdShow` for cleaner output.  
  ([6f1ea0d](https://github.com/forgezero-cli/ForgeZero/commit/6f1ea0d))

### Fixed

- **Config-only validation** – config-only mode now properly errors when no `SourceFiles` or `BuildRules` are provided, preventing ambiguous builds.  
  ([f8fd775](https://github.com/forgezero-cli/ForgeZero/commit/f8fd775))

### Build

- Removed binary files (`dump.rdb`, `fz`, `redis-server`) from repository.  
  ([394d90c](https://github.com/forgezero-cli/ForgeZero/commit/394d90c), [795dd80](https://github.com/forgezero-cli/ForgeZero/commit/795dd80), [b0bf0bd](https://github.com/forgezero-cli/ForgeZero/commit/b0bf0bd))

## [6.0.0] - 2026-07-15

### Added

- **Auto-dependency management** – `AutoBuildDeps` (default: true) automatically builds dependencies from `deps/` directory during build. Dependencies are built as static archives with ordering controlled by `configure.fz`.  
  ([7087eec](https://github.com/forgezero-cli/ForgeZero/commit/7087eec), [1edd34c](https://github.com/forgezero-cli/ForgeZero/commit/1edd34c))
- **`DepBuildConfig`** – per-dependency build settings: `enabled`, `skip_tests`, `outputs`, `include`, `environment`, `pre_build`, `post_build`, `exclude_files`, `only_files`.  
  ([92b880a](https://github.com/forgezero-cli/ForgeZero/commit/92b880a))
- **`AutoBuildConfig`** – global auto-build settings: `enabled`, `log_level`, `continue_on_error`, `build_order`, `default_skip_tests`, `default_environment`.  
  ([92b880a](https://github.com/forgezero-cli/ForgeZero/commit/92b880a))
- **`.fzignore` support** – files and directories can be excluded using `.fzignore` patterns, with priority loading from project root.  
  ([62ca4ca](https://github.com/forgezero-cli/ForgeZero/commit/62ca4ca))
- **Config-based source discovery** – source files and directories can be specified entirely in config (`source_file`, `source_dir`, `source_files`, `source_dirs`), eliminating the need for CLI flags.  
  ([69bf7ff](https://github.com/forgezero-cli/ForgeZero/commit/69bf7ff), [4719780](https://github.com/forgezero-cli/ForgeZero/commit/4719780))
- **Intelligent include filtering** – automatically skips source files that are included by other source files, preventing duplicate compilation.  
  ([7087eec](https://github.com/forgezero-cli/ForgeZero/commit/7087eec))
- **Dependency include discovery** – automatically detects and adds include paths from `deps/` structure, including `deps/*/include` and `deps/*/src` directories.  
  ([7087eec](https://github.com/forgezero-cli/ForgeZero/commit/7087eec))
- **Local config merging** – `fz.toml` and `.fz.toml` files in source directories are automatically merged with global config.  
  ([7087eec](https://github.com/forgezero-cli/ForgeZero/commit/7087eec))
- **Structured error types** – added `ConfigError` with detailed error codes for better error reporting (`ErrorInvalidConfig`, `ErrorParseTOML`, `ErrorCyclicInclude`, etc.).  
  ([92b880a](https://github.com/forgezero-cli/ForgeZero/commit/92b880a), [1edd34c](https://github.com/forgezero-cli/ForgeZero/commit/1edd34c))
- **`ScanDependenciesRoot`** – dependency scanning with explicit root directory for proper relative path resolution.  
  ([b7928c0](https://github.com/forgezero-cli/ForgeZero/commit/b7928c0), [8dc55e1](https://github.com/forgezero-cli/ForgeZero/commit/8dc55e1))
- **`NewDepBuilder`** – dedicated builder for automated dependency construction with logging and error handling.  
  ([1edd34c](https://github.com/forgezero-cli/ForgeZero/commit/1edd34c))

### Changed

- **Build source discovery** – now prioritizes config-defined sources when CLI flags are omitted, with fallback to CLI flags.  
  ([62ca4ca](https://github.com/forgezero-cli/ForgeZero/commit/62ca4ca), [69bf7ff](https://github.com/forgezero-cli/ForgeZero/commit/69bf7ff))
- **Ignore file loading** – `.fzignore` is now loaded from project root first, with verbose logging for debugging.  
  ([62ca4ca](https://github.com/forgezero-cli/ForgeZero/commit/62ca4ca))
- **Config validation** – `ValidateSourceFlags` now accepts config parameter for source discovery from config.  
  ([69bf7ff](https://github.com/forgezero-cli/ForgeZero/commit/69bf7ff), [4719780](https://github.com/forgezero-cli/ForgeZero/commit/4719780))
- **Dependency graph** – `buildDependencyGraph` now accepts `rootDir` for proper dependency resolution.  
  ([8dc55e1](https://github.com/forgezero-cli/ForgeZero/commit/8dc55e1))
- **Error handling** – all config errors now use structured `ConfigError` types with codes instead of raw strings.  
  ([92b880a](https://github.com/forgezero-cli/ForgeZero/commit/92b880a))

### Fixed

- **Obj directory creation** – `.fz_objs` directory is now created before dependency builds to prevent errors.  
  ([7087eec](https://github.com/forgezero-cli/ForgeZero/commit/7087eec))
- **Config include cycles** – cyclic includes are now properly detected and reported with `ErrorCyclicInclude`.  
  ([92b880a](https://github.com/forgezero-cli/ForgeZero/commit/92b880a))
- **Multi-format config** – config files without extension now try TOML then YAML fallback.  
  ([92b880a](https://github.com/forgezero-cli/ForgeZero/commit/92b880a))

## [6.0.0] - 2026-07-09

### Added

- **FZP (ForgeZero Preprocessor)** – built‑in preprocessor that handles `#define`, `#undef`, `#ifdef`, `#ifndef`, `#if`, `#else`, `#elif`, `#endif`, `#include`, `#error`, `#pragma once`. Automatically scans `*.h.in` templates and generates headers during a normal build.  
  ([4f4ed42](https://github.com/forgezero-cli/ForgeZero/commit/4f4ed42), [bf82e89](https://github.com/forgezero-cli/ForgeZero/commit/bf82e89))
- **`fzpkg` – secure package management** – packages are verified against a trusted‑key store. New sub‑commands: `fz pm verify`, `fz pm sign`, `fz pm keys`, `fz pm trust`.  
  ([987fe3c](https://github.com/forgezero-cli/ForgeZero/commit/987fe3c), [b6b6124](https://github.com/forgezero-cli/ForgeZero/commit/b6b6124))
- **CLI `--set` flag** – override any config field from the command line (repeatable).  
  ([11bf94f](https://github.com/forgezero-cli/ForgeZero/commit/11bf94f))
- **CLI `--config-fzp` flag** – explicitly load an FZP preprocessor configuration file.  
  ([63174e3](https://github.com/forgezero-cli/ForgeZero/commit/63174e3))
- **CLI `--verify-signatures` flag** – enable package signature verification during build.  
  ([63174e3](https://github.com/forgezero-cli/ForgeZero/commit/63174e3))
- **Automatic include dirs for generated headers** – `*.h.in` outputs are placed in `.fz_objs/include/` and added to the compiler’s `-I` paths.  
  ([8c3615e](https://github.com/forgezero-cli/ForgeZero/commit/8c3615e), [c3151be](https://github.com/forgezero-cli/ForgeZero/commit/c3151be))
- **Expanded variable expansion** – `$VAR` now resolves environment variables in config files.  
  ([a42c5f9](https://github.com/forgezero-cli/ForgeZero/commit/a42c5f9))
- **Security documentation** – added `SECURITY`, `FZP`, `FZPKG` guides.  
  ([1af0791](https://github.com/forgezero-cli/ForgeZero/commit/1af0791), [de72289](https://github.com/forgezero-cli/ForgeZero/commit/de72289), [2672f61](https://github.com/forgezero-cli/ForgeZero/commit/2672f61))
- **FZP macro expansion** – `#define` now expands macro values when `defines` are provided in `[preprocess]`; allows writing `#define FZ_OUTPUT OUTPUT` and getting `#define FZ_OUTPUT "myapp"` from `defines = { OUTPUT = "\"myapp\"" }`.  
  ([493287c](https://github.com/forgezero-cli/ForgeZero/commit/493287c))
- **Builder preprocessing fix** – preprocessor now correctly receives `defines` from the config, enabling macro substitution in generated headers.  
  ([a16557b](https://github.com/forgezero-cli/ForgeZero/commit/a16557b))
- **Integration test project** – added `FZP_TEST/` with a complete C project that verifies FZP preprocessing, including conditional blocks and macro substitution.  
  ([9728a0a](https://github.com/forgezero-cli/ForgeZero/commit/9728a0a))

### Changed

- **YAML configuration is now deprecated** – a warning is emitted when a YAML config is loaded; TOML is the recommended format.  
  ([6d817d1](https://github.com/forgezero-cli/ForgeZero/commit/6d817d1), [df815b7](https://github.com/forgezero-cli/ForgeZero/commit/df815b7))
- **Package manager (fz pm)** – replaced internal implementation with `fzpkg`, adding cryptographic trust and verification.  
  ([b6b6124](https://github.com/forgezero-cli/ForgeZero/commit/b6b6124))
- **Path sanitisation** – `pkgman` now rejects directory traversal and empty paths.  
  ([6bf8ec0](https://github.com/forgezero-cli/ForgeZero/commit/6bf8ec0))

### Fixed

- **Integration test for FZP** – test now uses system `fz` binary and skips if not found, avoiding build failures.  
  ([34b60c8](https://github.com/forgezero-cli/ForgeZero/commit/34b60c8), [9154a3c](https://github.com/forgezero-cli/ForgeZero/commit/9154a3c))
- **Include resolution** – relative includes are resolved correctly, and include cycles are detected.  
  ([4f4ed42](https://github.com/forgezero-cli/ForgeZero/commit/4f4ed42))

### Documentation

- Added `FZP`, `FZPKG`, `SECURITY` guides, updated `INDEX` and `TOML_CONFIG` to reflect deprecation of YAML.

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
63174e3 feat(cli): integrate FZP config loading and signature verification
11bf94f feat(cli): add --config-fzp, --set, and --verify-signatures flags
b6b6124 feat(pm): add verify, sign, keys, trust subcommands via fzpkg
621af0d docs: add FZPKG and SECURITY entries to INDEX
6d817d1 docs: mark YAML as deprecated, promote TOML as primary
c3151be feat(assembler): add AdditionalIncludeDirs for generated headers
fded515 test(assembler): add TestCompileCAddsAdditionalIncludeDirs
8c3615e feat(builder): run preprocess step and set include dirs automatically
ed6a6b1 test(builder): add TestRunPreprocessGeneratesHeaderFromTemplate
df815b7 feat(config): add PreprocessConfig, deprecate YAML, support --set overrides
2e5314a test(config): add tests for TOML enum, env vars, --set, generate config.h
5c540dc feat(errors): add CodePreprocessFailed and CodeIncludeFailed
6bf8ec0 fix(pkgman): sanitize package paths and prevent traversal
d784483 test(pkgman): add TestUpdateConfigRejectsTraversalPath
a42c5f9 feat(vars): expand environment variables in config expansion
1af0791 docs: add FZP preprocessor documentation
de72289 docs: add FZPKG package manager security docs
2672f61 docs: add security model overview
bf82e89 feat(config): add FZP config loading functions
34b60c8 test(config): add FZP integration tests
9154a3c test(config): add FZP unit tests
4f4ed42 feat(fzp): add ForgeZero Preprocessor core (lexer, parser, processor)
987fe3c feat(fzpkg): add package verification with trusted keys (verify, sign, trust)
97e36f1 chore: update .gitignore for build artifacts
3f5e785 style: add newline at end of file
62ca4ca feat: enhance build source discovery with config support
69bf7ff feat: integrate config with source flag validation
4719780 feat: support config-based source discovery in validation
7087eec feat: comprehensive dependency management and auto-build system
8dc55e1 feat: add rootDir support to dependency scanning
92b880a feat: add configuration error types and dependency build settings
5e88481 style: add copyright header to test file
b7928c0 feat: add root directory support for dependency scanning
1edd34c feat: add auto-dependency builder and error types
5d2fed1 feat: add ConfigOnly flag to CLI
5d5519c feat: propagate ConfigOnly flag to config
0569c1d feat: add configure.fz support and config-only mode
5d51d85 feat: add ConfigOnly field to Config struct
f668ac6 feat: propagate ParseMakefile from config to flags
f8fd775 test: add config-only mode test
0215d95 feat: add FZ_COMPILE_WORKER_MEM_MB env var
967e5e3 fix: improve symlink handling in source_hash
6f1ea0d chore: remove duplicate Debug output in cmdShow
b0bf0bd feat: add Makefile parsing for source discovery
8c1a1f0 test: add Makefile parsing tests
795dd80 chore: remove fz binary
d65d2e7 chore: add build tags and fix imports in parse_mk.go
5bc08ad assembler: fix data race by moving flag initialization to sync.Once and using target parameter
a81ff24 builder: add mutex protection for L1 cache to eliminate data race
85b526f builder: add custom steps execution and fix formatting
8c332d3 builder: add RAM cache capacity configuration via config
efb1576 builder: implement RAM cache size limits with atomic tracking
ddbd2c6 config: add build steps, step sets, RAM cache config and validation improvements
c643ff4 config: add tests for CacheRAMMB override and merge
7050146 config: replace sync.Map with RWMutex map for config cache
73d0763 config/toml: optimize TOML parsing with unsafe string conversion
af4bb07 scheduler: fix node dependency validation order to prevent leaks
d8530f9 scheduler: add tests for invalid dependency and error propagation
96c87ea scheduler: add condition variable for pending tasks to reduce busy-waiting
16d1d35 scheduler: add DAG stress test with 128 nodes
0bc36aa variables: add early return optimization for strings without dollar sign
4758fc6 builder: add tests for autodeps package
6f033d8 builder: add additional builder tests
ba554a7 builder: implement dependency build steps with parallel execution
87591cd builder: add integration tests for dependency build steps
191ef17 builder: add parallel step execution tests
e05874d builder: add unit tests for dependency build steps
c329d22 builder: add TOML config tests for dependency build steps
10319cf builder: add graph tests for dependency resolution
81d9890 linker: add object linking tests
3f5cd01 chore: add *.prof to gitignore
2257e55 fix: handle JSON encode errors in dispatch
14b2d8d fix: handle WriteSuccessReport error
5face59 fix: add error handling for write operations
942dda1 fix: handle EncodeBuildReport error in WriteErrorReport
88ac473 fix: handle JSON encode errors in audit and verify
a4d2186 docs: add new YouTube video link
e792c7d docs: mark cmodule as unsupported
722c33f fix: use exec.Command for GOROOT detection
98f993f perf: optimize byte slice operations with capacity checks
291e081 perf: use AppendByte/AppendBytes instead of append
1199697 perf: optimize Encoder with capacity checks and larger buffer
aa4bcc5 perf: use AppendBytes for byte sequences
bf06eb8 chore: replace deprecated ioutil.TempFile
304abdc fix: add error handling for cache operations
75347f9 chore: fix formatting and remove blank lines
60f7a7f chore: add newline at end of file
a615529 fix: add error handling for cache operations
e89fb9f perf: fix sync.Pool usage for topoOrder
6181552 chore: fix indentation in graph_test.go
372e4e5 fix: handle error return in config test
2ddf2a4 perf: rewrite Makefile parser with zero allocation parsing
9b669f8 test(scheduler): avoid unsafe.Pointer conversion in test helper
e734275 chore(gloria): remove unused asm trampoline with missing Go declaration
479f1b5 fix(assembler): match WriteByte signature to io.ByteWriter
fd508de fix(scheduler): avoid copying atomic values in dag nodes and priority queues
08cb407 drivers: add fo work-stealing thread pool
9037a00 linker: replace DAG scheduler with worker pool
bec8e9a builder: replace DAG scheduler with worker pool
a8dd4e0 cli: integrate config diagnostic error reporting
79a375b config: add tests for compiler and hardware validation
2ec9e53 config: add compiler, cpu_target, instruction_sets, and concurrency settings
f9b7697 config: enhance Error struct with location and fix suggestions
184746d errors: implement diagnostic system with line-level error reporting
```
