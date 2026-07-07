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
[1100f19](https://github.com/forgezero-cli/ForgeZero/commit/1100f19))

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

### Fixed
- **Comprehensive test coverage** for dependency parsing, graph building, DAGScheduler, and PCH integration hooks.  
  ([b3c0694](https://github.com/forgezero-cli/ForgeZero/commit/b3c0694), [98ab763](https://github.com/forgezero-cli/ForgeZero/commit/98ab763), [51840fd](https://github.com/forgezero-cli/ForgeZero/commit/51840fd))

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