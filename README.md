# ⚡️ ForgeZero (fz) — Overview & Performance

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

ForgeZero is a high-performance, zero-overhead build tool for assembly, C, C++, Objective-C, and Gloria developers.

It wraps NASM, GAS, FASM, GCC, Clang, Zig, and LD into a single unified command-line interface:

- No Makefiles
- No build scripts
- No configuration required to get started

> Inspired by the simplicity of **Suckless** and the efficiency of **TinyCC**

**Author:** [alexvoste](https://github.com/alexvoste)

*Non nobis, Domine, non nobis, sed nomini tuo da Gloriam*

---

## 📚 Table of Contents

- [01 — Overview & Performance](docs/01-overview-perf.md)
- [02 — Installation](docs/02-installation.md)
- [03 — Quick Start & Usage](docs/03-quickstart-usage.md)
- [04 — Build Modes, Languages & Extensions](docs/04-modes-languages.md)
- [05 — CLI Build Reference, Profiles & Linking](docs/05-cli-build-reference.md)
- [06 — C / C++ / Objective-C](docs/06-c-cpp-objc.md)
- [07 — Gloria Language](docs/07-gloria-language.md)
- [08 — Cross-Compilation](docs/08-cross-compilation-targets.md)
- [09 — Configuration File Reference](docs/09-config-file-reference.md)
- [10 — Assembler Backends & Zig Backend](docs/10-backends-asm-linker-zig.md)
- [11 — Supply Chain Security (SBOM + SAST audit)](docs/11-supply-chain-security.md)
- [12 — Reproducible builds, verify, bench](docs/12-reproducible-integrity-bench.md)
- [13 — WebAssembly (WASM)](docs/13-wasm-web-targets.md)
- [14 — Project tools (init, contribute, LSP, update)](docs/14-project-tools.md)
- [15 — Exit codes & Troubleshooting](docs/15-exit-codes-troubleshooting.md)
- [16 — Roadmap](docs/16-roadmap.md)
- [90 — Virtual Filesystem Layer (Aegis)](docs/90-internals-aegis.md)
- [91 — Aegis Security Core](docs/91-internals-security-core.md)
- [92 — Doctor, Platform Readiness, Testing Standards](docs/92-internals-doctor-platform-testing.md)
- [93 — HADES Engine, Contributing, License](docs/93-internals-hades-contributing-license.md)
- [99 — Contents (Index)](docs/99-contents.md)

---

## Performance: Full Scaling Benchmark

Benchmarks measured against standard `nasm -f elf64 && ld` and `make -j4` pipelines.
Test environment: **Intel Core i5-10310U** (4C/8T, 1.7 GHz base), Arch Linux, Samsung 980 NVMe.
Results are mean ± stddev over ≥10 runs via `hyperfine`.

| Modules | ForgeZero (`fz`) | Traditional (`make -j4`) | Speedup |
|---------|-------------------|--------------------------|---------|
| 20      | 19.3 ± 1.2 ms     | 45.4 ± 2.3 ms            | **2.35×** |
| 50      | 31.1 ± 1.3 ms     | 85.0 ± 2.1 ms            | **2.73×** |
| 100     | 57.0 ± 5.3 ms     | 185.5 ± 7.7 ms           | **3.25×** |
| 150     | 73.1 ± 4.3 ms     | 229.3 ± 3.6 ms           | **3.14×** |
| 200     | 82.2 ± 4.2 ms     | 291.1 ± 11.2 ms          | **3.54×** |
| 400     | 223.1 ± 9.8 ms    | 1105.0 ± 24.1 ms         | **4.95×** |

### Extreme Scaling stress-test (5,000 C files)

When we scale up to huge, multi-module codebases, the architectural gap widens exponentially. Testing on an **AMD Ryzen 7 PRO 4750U** (16 threads) and old **AMD FX-8370E** (8 cores) with 5,000 files shows a complete collapse of traditional tools:

* **Ninja (`ninja -j 16`):** ~28.9 seconds (choking on 5,000+ process forks and context switches).
* **ForgeZero (`fz`):** **~597 milliseconds** 🚀 *(47.3× faster, utilizing all 16 threads at 98% capacity with a mere 47MB RAM footprint).*

---

## Scaling Efficiency

| Metric | `fz` | `make -j4` |
|--------|-------|------------|
| Time growth (20→400 modules) | **+1056%** | **+2333%** |
| Overhead per module | ~0.36 ms | ~1.23 ms |
| I/O operations | **0 intermediate files** | 2× modules (`.o` read/write) |
| Process forks | **1** | ~2× modules + 1 |

> **Conclusion:** ForgeZero maintains **~3–5× speedup** at scale, growing to nearly **5× at 400 modules** and up to **47× at 5,000 modules**.
> Traditional pipelines suffer from super-linear overhead due to process spawning, I/O contention, and CPU cache thrashing.
> ForgeZero's single-process design preserves cache locality across the entire build.

### Why the difference?

| Factor | Traditional (`make + nasm + ld`) | ForgeZero (`fz`) |
|--------|---------------------------------|-------------------|
| **Processes** | 400+ forks at scale | **Minimally required forks** (highly concurrent scheduler) |
| **I/O** | Writes N intermediate `.o` files to disk | **Zero intermediate files** (in-memory pipeline) |
| **CPU Cache** | Cold start for every fork | **Hot cache** (code & data stay in L1/L2) |
| **Parallelism** | OS-level (`-j4`), high scheduling overhead | **Goroutines**, zero-cost concurrency |
| **Memory** | GC/Allocator overhead per process | **Zero-allocation hot path** (`0 allocs/op`, `0 B/op`) |
| **Allocations** | Unbounded heap churn | **Stack-buffered syscalls** in hot paths |

---

## Scaling Projection

| Modules | `fz` (est.) | `make -j4` (est.) | Speedup |
|---------|--------------|-------------------|---------|
| 20      | 19 ms        | 45 ms             | **2.35×** |
| 100     | 57 ms        | 185 ms            | **~3.2×** |
| 400     | 223 ms       | 1105 ms           | **~4.9×** |
| 1000    | ~530 ms      | ~3000+ ms         | **~5.5×** |

*Note: Projections beyond 400 modules assume continued sub-linear growth. Real-world results vary based on I/O and CPU contention.*

---

## How to reproduce

```bash
# Clone and build ForgeZero
git clone https://github.com/forgezero-cli/ForgeZero
cd ForgeZero
chmod +x build.sh
./build.sh
# Or
bash build.sh 

# Run the benchmark script (generates N test modules and runs hyperfine)
./bench.sh  # Edit NUM_MODULES in script for different module counts

# Export benchmark results to Markdown
hyperfine --warmup 3 --prepare 'make clean && rm -rf .fz_objs fz_out' \
  './fz -dir . -out fz_out' 'make -j4' \
  --export-markdown results.md
```