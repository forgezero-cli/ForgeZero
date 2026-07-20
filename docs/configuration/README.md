# Configuration model

ForgeZero uses a configuration layer to translate project intent into build actions. The important design choice is that configuration is explicit and composable, not hidden in an imperative build script.

## How configuration is loaded

The project resolves configuration from a small set of known locations, then validates and expands it. The loader supports TOML as the primary format and YAML as a compatibility format.

## What configuration controls

Configuration can define:

- source directories and files,
- output names and build mode,
- flags for assembler, compiler, and linker,
- variables used inside the config and downstream scripts,
- hooks and build rules,
- dependency build steps,
- preprocessor behavior and include paths.

## Why the config is structured this way

The config system is intentionally broad because ForgeZero needs to support both simple projects and highly customized toolchain setups. The design preserves the ability to grow from a one-file experiment to an advanced embedded or systems build without switching systems.
