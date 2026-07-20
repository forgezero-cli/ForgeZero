# Configuration flexibility

ForgeZero does not assume that every project fits the same build shape. A build system that is only good at the simplest case quickly becomes painful in real engineering environments.

## The flexibility model

The configuration system supports:

- multiple source roots,
- explicit file lists,
- glob-like include and exclude patterns,
- per-language toolchain flag injection,
- custom hooks and scripts,
- dependency build steps,
- conditional preprocessor behavior.

## Why this is important

Real projects often need more than “compile these files and link them.” They need environment-specific settings, platform-specific source selection, bootstrapping steps, and reproducible build behavior. ForgeZero keeps these capabilities in the config layer so they remain visible and maintainable.
