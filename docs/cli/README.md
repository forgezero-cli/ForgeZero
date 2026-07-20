# CLI usage

The CLI is intentionally thin. It is responsible for parsing user intent, selecting configuration, and handing the actual work to the build engine.

## Typical commands

- `fz` or `fz -dir .` builds the project in the current directory.
- `fz -asm path/to/file.asm` assembles a single file.
- `fz -cc path/to/file.c` compiles a single C-like translation unit.
- `fz -dir src -out app` builds from a specific directory and writes a named binary.
- `fz -watch` rebuilds automatically when sources change.
- `fz -clean` removes generated build artifacts.
- `fz -json` produces machine-readable output for scripting and CI.

## Important design point

The CLI is not the actual compiler. It is the control plane. The real compilation work lives in the builder and backend integrations. That separation is deliberate: it keeps the user experience consistent even when the underlying toolchain changes.
