# Error handling and exit codes

ForgeZero aims to surface useful errors instead of silently guessing. When a build fails, the system tries to report the root cause in the context of the build step that failed.

## What to expect

- Syntax or toolchain errors are surfaced with the relevant file and command.
- Config errors explain which value is invalid and why.
- Preprocessor failures report the failing directive.
- Linker and symbol-check failures report the problem clearly rather than hiding it deep in a toolchain log.

## Philosophy

The project prefers transparent failure. A build system should be debuggable. If a configuration is wrong, the user should not have to infer that from a vague error message or a cryptic tool exit.
