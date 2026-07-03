# C / C++ / Objective-C

## C Compilation

### Strict warning flags

Every `.c` file is compiled with:

```text
-Wall -Wextra -Werror -Wpedantic -Wshadow -Wconversion
```

Any warning stops the build.

### Sanitizers

By default:

```text
-fsanitize=address
-fsanitize=undefined
```

In strict mode (`-strict`):

```text
-fsanitize=address
-fsanitize=undefined
-fsanitize-address-use-after-return=always
-fsanitize-address-use-after-scope
```

Disable sanitizers:

```bash
fz -cc main.c -sanitize=false
```

Sanitizers are automatically disabled for `wasm32-*` targets.

---

## C++ Compilation

`.cpp` / `.cc` / `.cxx` are compiled with the same strict flags as C.

```text
-Wall -Wextra -Werror -Wpedantic -Wshadow -Wconversion
```

Use:

```bash
fz -cc main.cpp
fz -cc main.cxx
```

Mixed C/C++ directory builds are supported:

```bash
fz -dir ./src
```

---

## Objective-C Compilation

Objective-C (`.m`) is supported as a first-class extension.

### Pipeline

1) Detect `.m`
2) Select backend (Clang by default; Zig when `-zig` is active)
3) Compile with Objective-C mode
4) Auto-link Objective-C runtime and required frameworks

### Usage

```bash
fz -cc main.m
fz -cc main.m -verbose
fz -cc main.m -zig
fz -dir ./src
```

### Requirements

- `clang` must be available on `PATH` for full feature sets.
- On macOS: Xcode Command Line Tools provide required framework headers.
- On Linux: you can compile Obj-C code that targets GNU runtime; macOS-specific frameworks are not available.

