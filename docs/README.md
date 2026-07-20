# ForgeZero Documentation

ForgeZero is a build system that favors explicit control over hidden magic. It is designed for engineers who want deterministic builds, predictable toolchain wiring, and fast iteration without surrendering the ability to understand what the compiler and linker are doing.

This documentation is organized around the actual implementation structure of the project:

- Architecture: how the entrypoint, config loader, builder, assembler, linker, and runtime helpers fit together.
- CLI: how the command-line interface is used in practice.
- Configuration: how TOML files, variables, profiles, hooks, and rules are interpreted.
- Languages: how supported source languages and the custom Gloria language are handled.
- Workflow: how real projects are built, tested, watched, and packaged.
- Internals: how the preprocessor, build engine, security layers, and analysis tools work.

## Philosophy

ForgeZero does not try to become a black box. Its guiding principles are:

1. Explicitness over magic.
2. Deterministic builds over heuristics.
3. Fast iteration over unnecessary indirection.
4. Security-conscious file handling and toolchain control.

If you want a build system that stays close to the compiler and linker, ForgeZero is designed for you.
