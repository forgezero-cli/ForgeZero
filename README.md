# ForgeZero (fz)

Zero-overhead build tool for assembly, C, C++, Objective-C, and Gloria.

Version: 5.3.0
Language: Go
License: GPLv3
Platforms: Linux, Windows, macOS
Author: alexvoste (https://github.com/alexvoste)

ForgeZero wraps NASM, GAS, FASM, GCC, Clang, Zig, and LD behind a single
command-line interface. No Makefiles, no build scripts, no configuration
required to get started.

Design influences: Suckless (simplicity), TinyCC (efficiency).


## Documentation

    docs/01-overview-perf.md              Overview & performance
    docs/02-installation.md               Installation
    docs/03-quickstart-usage.md           Quick start & usage
    docs/04-modes-languages.md            Build modes, languages & extensions
    docs/05-cli-build-reference.md        CLI build reference, profiles & linking
    docs/06-c-cpp-objc.md                 C / C++ / Objective-C
    docs/07-gloria-language.md            Gloria language
    docs/08-cross-compilation-targets.md  Cross-compilation
    docs/09-config-file-reference.md      Configuration file reference
    docs/10-backends-asm-linker-zig.md    Assembler backends & Zig backend
    docs/11-supply-chain-security.md      Supply chain security (SBOM + SAST)
    docs/12-reproducible-integrity-bench.md  Reproducible builds, verify, bench
    docs/13-wasm-web-targets.md           WebAssembly (WASM)
    docs/14-project-tools.md              Project tools (init, contribute, LSP, update)
    docs/15-exit-codes-troubleshooting.md Exit codes & troubleshooting
    docs/16-roadmap.md                    Roadmap
    docs/90-internals-aegis.md            Internals: virtual filesystem layer (Aegis)
    docs/91-internals-security-core.md    Internals: Aegis security core
    docs/92-internals-doctor-platform-testing.md  Internals: doctor, platform readiness, testing
    docs/93-internals-hades-contributing-license.md  Internals: HADES engine, contributing, license
    docs/99-contents.md                   Full index


## Performance

Measured against `nasm -f elf64 && ld` and `make -j4`.
Environment: Intel Core i5-10310U (4C/8T, 1.7 GHz base), Arch Linux, NVMe SSD.
Mean +/- stddev over >= 10 runs, via hyperfine.

    modules   fz            make -j4         speedup
    20        19.3±1.2 ms   45.4±2.3 ms      2.35x
    50        31.1±1.3 ms   85.0±2.1 ms      2.73x
    100       57.0±5.3 ms   185.5±7.7 ms     3.25x
    150       73.1±4.3 ms   229.3±3.6 ms     3.14x
    200       82.2±4.2 ms   291.1±11.2 ms    3.54x
    400       223.1±9.8 ms  1105.0±24.1 ms   4.95x

Stress test, 5000 C files, AMD Ryzen 7 PRO 4750U (16 threads):

    ninja -j16   ~28.9 s
    fz           ~597 ms   (47.3x, 98% utilization across 16 threads, ~47 MB RSS)

Scaling characteristics, 20 to 400 modules:

    time growth        fz +1056%      make -j4 +2333%
    overhead/module     ~0.36 ms       ~1.23 ms
    intermediate files  0              2x modules (.o read/write)
    process forks       1              ~2x modules + 1

ForgeZero holds a 3-5x speedup at scale, rising to ~5x at 400 modules and
~47x at 5000 modules. Traditional pipelines degrade super-linearly from
process spawning, I/O contention, and cache thrashing; ForgeZero's
single-process, in-memory pipeline preserves cache locality across the
build.

Projection (est., sub-linear growth assumed beyond 400 modules):

    modules   fz         make -j4      speedup
    1000      ~530 ms    ~3000+ ms     ~5.5x


## Building from source

    git clone https://github.com/forgezero-cli/ForgeZero
    cd ForgeZero
    ./build.sh

Run benchmarks (edit NUM_MODULES in bench.sh to change scale):

    ./bench.sh
    hyperfine --warmup 3 --prepare 'make clean && rm -rf .fz_objs fz_out' \
        './fz -dir . -out fz_out' 'make -j4' \
        --export-markdown results.md