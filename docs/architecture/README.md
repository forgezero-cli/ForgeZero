# Architecture

The architecture of ForgeZero is intentionally simple: a thin CLI entrypoint collects intent, a config layer normalizes it, and a build engine turns that configuration into a concrete set of compiler, assembler, and linker actions.

## Core flow

1. The entrypoint in the CLI bootstraps flags, config, and runtime context.
2. The config subsystem loads TOML or YAML, expands variables, applies defaults, and validates the result.
3. The builder discovers sources, filters platform-specific files, resolves dependencies, and schedules work.
4. The assembler and linker backends produce objects and link the final binary.
5. Supporting layers such as hooks, scripts, SBOM generation, and auditing run around the core pipeline.

## Why this shape?

ForgeZero is designed as a coordinator rather than a monolithic compiler. That gives it three practical advantages:

- It can support many source kinds without reimplementing each language.
- It can stay fast by keeping the orchestration logic in one place and relying on upstream toolchains for the real compilation work.
- It preserves a direct mental model: the user still sees the compiler, the linker, and the resulting object files.

## Main subsystems

- CLI and runtime setup: command parsing and project initialization.
- Config layer: parsing, validation, merging, and variable expansion.
- Builder: discovering files, scheduling work, applying caching, and linking.
- Assembler and linker integration: dispatching to NASM, GAS, FASM, GCC, Clang, Zig, and LD.
- Security and analysis helpers: SBOM, hooks, audits, and isolation settings.
