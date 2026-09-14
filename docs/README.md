# ForgeZero Documentation

The normative reference is [ForgeZero Technical Specification](SPECIFICATION.md). It is the authoritative description of the implemented orchestration model, performance-critical paths, configuration schema, Custom Task Engine, Gloria, and Plan 9 assembly interoperability.

ForgeZero is a deterministic build orchestration layer. It favors explicit control over hidden magic and keeps compiler, assembler, linker, ABI, target, and cache boundaries observable.

The documentation is organized around the implementation structure of the project:

- Architecture: how the entrypoint, config loader, builder, assembler, linker, and runtime helpers fit together.
- CLI: how the command-line interface is used in practice.
- Configuration: how TOML files, variables, profiles, hooks, and rules are interpreted.
- Languages: how supported source languages and the custom Gloria language are handled.
- Workflow: how real projects are built, tested, watched, and packaged.
- Internals: how the preprocessor, build engine, security layers, and analysis tools work.

## Normative reading order

1. [Technical Specification](SPECIFICATION.md)
2. [Getting started](getting-started/quickstart.md)
3. [CLI reference](cli/reference.md)
4. [TOML configuration](configuration/toml.md)
5. [Internals](internals/README.md)

## Philosophy

ForgeZero does not try to become a black box. Its guiding principles are:

1. Explicitness over magic.
2. Deterministic builds over heuristics.
3. Fast iteration over unnecessary indirection.
4. Security-conscious file handling and toolchain control.

If you want a build system that stays close to the compiler and linker, ForgeZero is designed for you.
