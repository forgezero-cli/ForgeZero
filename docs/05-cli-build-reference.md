# CLI Build Reference, Profiles & Linking

## CLI Reference (high-level)

Run `fz` with one of the supported modes (e.g. `-asm`, `-cc`, `-gloria`, `-dir`) or a valid config file.

### Output naming

- `-out <name>` sets the output binary name.
- `-out-obj <name>` sets the object file name (single-file mode).

### Build profiles (`-profile` / `-p`)

Profiles are named presets that configure:

- CPU core usage
- `-j` parallel job count
- compiler optimization level

Active profile is persisted between runs in `~/.config/fz/.profile.config`.

| Profile | Cores (`-j`) | Optimization | GOMAXPROCS | Use case |
|---------|---------------|--------------|------------|----------|
| `performance` | All cores | `-O3`  | All cores | Maximum speed (CI/release) |
| `balanced`    | Half cores | `-O2`  | Half cores | Default daily development |
| `power-saver`| 1 | `-Os` | 1 | Battery-constrained |

Short form:

```bash
fz -p performance -cc main.c
```

---

## Linking modes

### `-mode auto` (default)

ForgeZero tries linkers in order until one works:

1) `gcc`
2) `gcc -no-pie`
3) `ld`

If `-strict` is enabled, `clang` with full sanitizer flags is attempted first.

### `-mode c`

Force GCC/Clang linking (C runtime).

### `-mode raw`

Force LD-only linking. Intended for bare-metal targets.

> Warning: raw-linked binaries cannot reference libc symbols.

---

## Direct linker invocation: `-ld` (v4.1.0)

`-ld` bypasses compiler validation layers and invokes the linker directly, reducing overhead ~3–5% on small projects.

```bash
fz -asm boot.asm -ld -out boot.elf
```

---

## Security & reporting-related flags

- `--reproducible` — deterministic builds
- `--arena-size=N` — advanced plugin arena override
- `-sanitize=false` — disable sanitizers
- `-strict` — stricter sanitizers (prefers clang)
- `-json` — emit a JSON build report
- `-watch` — rebuild on file changes
- `-clean` — remove build artifacts and exit

---

## Sub-commands

### Package manager

- `fz pm add <repo>[@version]`
- `fz pm remove <package>`
- `fz pm list`
- `fz pm update`
- `fz pm catalog`
- `fz pm search <query>`
- `fz pm install <name>`

### Security & integrity

- `fz sbom`
- `fz audit`
- `fz verify`
- `fz doctor`

### Performance

- `fz bench`

### Contributor support

- `fz contribute`

