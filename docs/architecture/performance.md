# Performance model

ForgeZero is designed for build throughput at scale. Its performance story is not based on a hidden optimizer; it is based on reducing overhead that usually compounds in large projects.

## What makes it faster

The main improvements come from:

- avoiding unnecessary process spawns for every translation unit,
- keeping build state in memory where practical,
- using caching for unchanged inputs,
- parallelizing work across available CPU cores,
- avoiding repeated full-project reanalysis when only a small slice changes.

## Why this matters

Traditional build pipelines often spend a lot of time in orchestration rather than compilation. ForgeZero reduces that orchestration cost so the toolchain spends more of its time doing useful work.

## Benchmarks and interpretation

The project’s published benchmarks compare ForgeZero against a conventional make-based pipeline for assembly-oriented workloads. The results show large gains as the number of modules grows. That should be read as evidence that orchestration overhead matters more than many people expect.

The important lesson is not “ForgeZero is always faster in every possible case.” The real lesson is that build systems become much more efficient when they are purpose-built for the job and avoid repeated work that the compiler and linker already do not need.
