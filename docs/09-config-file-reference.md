# Configuration File Reference

ForgeZero can read YAML configuration files (project-level config) and merge them with defaults and CLI flags.

## Basic fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `source_dir` | string | — | Single source directory (backward compatible) |
| `source_dirs` | `[]string` | — | Multiple source directories scanned recursively |
| `source_files` | `[]string` | — | Exact list of files to build (overrides directories) |
| `output` | string | auto | Output binary name |
| `mode` | string | `auto` | Linking mode: `auto`, `c`, `raw` |
| `format` | string | `elf64` | Output format: `elf32`, `elf64`, `bin` |
| `target` | string | — | Cross-compilation triple |
| `backend` | string | `auto` | Compiler backend: `auto`, `gcc`, `clang`, or `zig` |
| `profile` | string | `balanced` | Build profile: `performance`, `balanced`, `power-saver` |
| `reproducible` | bool | `false` | Enable deterministic build mode |
| `type` | string | `executable` | Output type: `executable` or `static` |
| `lib` | string | — | Library name for `-type static` |
| `jobs` | int | profile-dependent | Parallel compilation jobs (`0` = auto) |
| `debug` | bool | `false` | Emit debug symbols (`-g`) |
| `verbose` | bool | `false` | Print invoked commands |
| `keep_obj` | bool | `false` | Preserve object files after linking |
| `no_cache` | bool | `false` | Disable build cache |
| `cache_mode` | string | `disk` | Cache storage: `disk`, `ram`, or `off` |
| `sanitize` | bool | `true` | Enable ASan + UBSan for C/C++ |
| `strict` | bool | `false` | Strict sanitizer mode (prefers clang/clang++) |
| `ignore_file` | string | `.fzignore` | Path to a `.gitignore`-style exclusion file |

---

## Multiple source directories

```yaml
source_dirs:
  - kernel
  - libc
  - drivers
output: forgeos.elf
mode: raw
```

Object names are prefixed with their parent directory to avoid collisions.

---

## Explicit source file lists

```yaml
source_files:
  - boot/start.asm
  - kernel/main.c
  - kernel/irq.c
output: kernel.elf
mode: raw
```

`source_files` takes precedence over `source_dirs` / `source_dir`.

---

## Include & exclude patterns

```yaml
exclude:
  - "test_*"
  - "*/legacy/"
  - "*.tmp"

include:
  - "*.asm"
  - "*.c"
```

Evaluation order: `exclude` → `.fzignore` → symlink boundary check → `include` → supported extensions.

---

## Library linking

```yaml
libs:
  - m         # -lm
  - pthread   # -lpthread
  - c         # -lc
```

---

## Custom compiler & linker flags

```yaml
flags:
  asm:
    - -DDEBUG_BUILD
    - -I./include
  cc:
    - -O3
    - -march=native
    - -DNDEBUG
    - -ffreestanding
  ld:
    - -T
    - linker.ld
    - -Map
    - output.map
```

---

## `.fzignore` file

Same semantics as `.gitignore`. Example:

```text
*.o
*.swp
temp/
test_*/
vendor/
legacy/old_abi.asm
```

---

## Full annotated example

```yaml
source_dirs:
  - kernel
  - libc
  - drivers

output: forgeos.elf
format: elf64
profile: balanced

mode: raw
debug: true
verbose: false
keep_obj: true
no_cache: false
cache_mode: disk
jobs: 0

sanitize: true
strict: false

exclude:
  - "test_*"
  - "*/legacy/"
  - "*.tmp"

include:
  - "*.asm"
  - "*.c"
  - "*.cpp"
  - "*.m"
  - "*.glo"
  - "*.s"

libs:
  - gcc
  - m

flags:
  asm:
    - -DDEBUG_BUILD
    - -I./include
  cc:
    - -O2
    - -march=native
    - -ffreestanding
  ld:
    - -T
    - linker.ld

ignore_file: .myfzignore
```

