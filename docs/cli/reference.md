# CLI reference guide

The following flags are the most important ones to understand when working with ForgeZero.

## Build selection

- `-dir <path>`: build a directory tree.
- `-asm <file>`: build a single assembly source.
- `-cc <file>`: build a single C/C++/Objective-C source.
- `-source <file>` or equivalent single-file entrypoints: target a specific input file.
- `-out <name>`: set the output binary name.

## Build behavior

- `-mode auto|c|raw`: choose how the link step is performed.
- `-debug`: emit debug information.
- `-verbose`: print the executed commands.
- `-keep-obj`: retain intermediate object files.
- `-no-cache`: disable incremental caching.
- `-strict`: enable stricter diagnostics and sanitization-related behavior.
- `-watch`: rebuild automatically on changes.
- `-json`: emit JSON output for tools.
- `-clean`: remove generated build products.

## Configuration and environment

- `-config <file>`: load a specific config file.
- `-set <key=value>`: override a config value at runtime.
- `-profile <name>`: select a named profile.

The full CLI surface is larger, but most work is centered around the flags above. The command line is designed to make the happy path obvious while still allowing advanced users to override details when necessary.
