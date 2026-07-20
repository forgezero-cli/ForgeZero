# Security and integrity

Security is treated as a first-class concern in ForgeZero rather than as an afterthought. The project includes mechanisms for auditing, SBOM generation, path validation, and stricter build isolation.

## Main security ideas

- The FZP preprocessor restricts includes to allowed paths.
- The builder can enforce symbol checks and perform integrity-oriented checks around outputs.
- The configuration system allows isolation and deterministic stripping options.
- Toolchain and environment behavior can be constrained through configuration.

## Why this matters

A build system that handles source code and external toolchains must be careful about path traversal, unexpected file writes, and ambiguity in the build graph. ForgeZero addresses these concerns in a practical way instead of pretending they will not arise in real projects.
