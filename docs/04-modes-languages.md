# Build Modes & Supported Languages

## Supported Languages & Extensions

| Extension               | Language      | Backend                    | Notes |
|-------------------------|---------------|----------------------------|-------|
| `.asm`                  | Assembly      | NASM                       | x86/x86-64, Intel syntax, ELF64 |
| `.s`                    | Assembly      | GAS via `gcc -c`           | AT&T syntax |
| `.S`                    | Assembly      | GAS via `gcc -c`           | AT&T syntax + C preprocessor |
| `.fasm`                 | Assembly      | FASM                       | Separate install; auto `format ELF64` injection |
| `.c`                    | C             | GCC, Clang, or `zig cc`    | Strict flags + sanitizers by default |
| `.cpp` / `.cc` / `.cxx` | C++           | G++, Clang++, or `zig c++` | Same strict flags as C |
| `.m`                    | Objective-C   | Clang or `zig cc`          | Auto framework linking |
| `.glo`                  | Gloria        | Built-in (HADES)           | Compiles to raw x86-64 ELF |

All other extensions are silently ignored during directory and recursive scanning.

---

## Build Modes

### 1) Single File Mode

Compiles and links a single source file into a binary.

```bash
fz -asm program.asm
fz -cc main.c
fz -cc main.cpp
fz -cc main.m
fz -gloria main.glo
```

- Output name is derived from the source filename (`program.asm` → `program`).
- A single object file is created and removed after linking unless `-keep-obj` is set.
- Override output with `-out` and object name with `-out-obj`.

### 2) Directory Mode

Recursively scans a directory, compiles each supported source to a unique object file, then links everything.

```bash
fz -dir ./src
```

Object file naming uses the full relative path to avoid collisions across subdirectories.
Object files live in `.fz_objs/` and are removed after linking unless `-keep-obj` is passed.

### 3) Configuration File Mode

ForgeZero searches the working directory for a config file in this order:

1. `.fz.yaml`
2. `fz.yaml`
3. `.fz.yml`
4. `fz.yml`

Run without flags:

```bash
fz
```

Or specify explicitly:

```bash
fz -config ./configs/release.yaml
```

CLI flags always take precedence over config values.

---

## What to read next

- CLI reference and linking modes: [05 — CLI Build Reference, Profiles & Linking](./05-cli-build-reference.md)
- Configuration fields: [09 — Configuration File Reference](./09-config-file-reference.md)

