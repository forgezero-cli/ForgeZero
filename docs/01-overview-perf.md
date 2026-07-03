# ForgeZero (fz) — Overview & Performance

<div align="center">
  <table style="border:none; background:transparent;">
    <tr>
      <td style="vertical-align:middle; padding-right:32px; border:none;">
        <img src="../pictures/fz.jpg" alt="ForgeZero Logo" width="180" />
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

### Scaling Efficiency

| Metric | `fz` | `make -j4` |
|--------|-------|------------|
| Time growth (20→400 modules) | **+1056%** | **+2333%** |
| Overhead per module | ~0.36 ms | ~1.23 ms |
| I/O operations | **0 intermediate files** | 2× modules (`.o` read/write) |
| Process forks | **1** | ~2× modules + 1 |

> **Conclusion:** ForgeZero maintains **~3–5× speedup** at scale, growing to nearly **5× at 400 modules**.
> Traditional pipelines suffer from super-linear overhead due to process spawning, I/O contention, and CPU cache thrashing.
> ForgeZero's single-process design preserves cache locality across the entire build.

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

