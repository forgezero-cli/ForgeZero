# TOML configuration

TOML is the preferred configuration format for ForgeZero. It is human-readable, explicit, and easy to generate from tooling.

## Minimal example

```toml
output = "app"
mode = "auto"
profile = "balanced"

[flags]
cc = ["-O2"]
ld = ["-Wl,--gc-sections"]
```

## Common sections

- `source_dirs` and `source_files`: select which inputs should participate in the build.
- `exclude` and `include`: restrict which files are visible to the builder.
- `output`, `out_obj`, `mode`, `target`: control output naming and link behavior.
- `flags.asm`, `flags.cc`, `flags.ld`: pass backend-specific flags.
- `toolchain_opts`: control tool discovery and environment allowances.
- `preprocess`: configure the built-in FZP preprocessor.
- `hooks`: define pre-build actions and failure handling.
- `build_rules`: declare custom transformations and outputs.
- `dep_build`: define dependency build steps for larger projects.

## Variables and expansion

Values can contain variables such as `$PWD`, `$HOME`, or custom entries from the `variables` section. Expansion happens before the config is validated. This allows one configuration file to be reused across different environments without hard-coding machine-specific paths.
