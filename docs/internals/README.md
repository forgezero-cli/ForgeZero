# Internals

The internal packages are where ForgeZero’s behavior is implemented. This section is primarily for contributors and maintainers who need to understand how the project works below the CLI surface.

## Important packages

- `internal/config`: parsing, validation, defaults, and config merging.
- `internal/builder`: source discovery, dependency tracking, caching, and build orchestration.
- `internal/assembler`: backend integration for assembly compilation.
- `internal/linker`: object linking and binary output.
- `internal/fzp`: the lightweight preprocessor used by the project.
- `internal/gloria`: the experimental Gloria language implementation.
- `internal/fs`, `internal/seal`, `internal/sbom`, and related packages: security and integrity support.

## Design principle

The implementation is modular, but the modules are intentionally close to the build problem domain. This makes the code easier to reason about than a generic plugin framework or a heavily abstracted build engine.
