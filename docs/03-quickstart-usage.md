# ForgeZero Quick Start & Usage

## Quick Start

### Assemble / run

```bash
fz -asm hello.asm
./hello
```

### Compile C

```bash
fz -cc main.c
./main
```

### Compile C++

```bash
fz -cc main.cpp
./main
```

### Compile Objective-C

```bash
fz -cc main.m
./main
```

### Compile Gloria

```bash
fz -gloria main.glo
./main
```

### Build a directory

```bash
fz -dir ./src
./src
```

### Initialize a new project

```bash
fz -init
```

## Build profiles

```bash
fz -p performance -cc main.c
fz -profile power-saver -dir ./src
```

## Cross-compilation

```bash
fz -cc main.c -target arm-linux-gnueabihf
```

## Build with Zig backend

Zig avoids installing a prefixed cross toolchain.

```bash
fz -cc main.c -zig -target aarch64-linux-musl
```

## Direct linker invocation

`-ld` (v4.1.0) bypasses compiler validation layers and invokes the linker directly.

```bash
fz -asm boot.asm -ld -out boot.elf
```

## IDE integration

Generate `compile_commands.json`:

```bash
fz -compile-commands
fz -dir ./src -compile-commands
```

## Libraries

### Static library

```bash
fz -dir ./src -type static -lib mylib
```

### Shared library

```bash
fz -cc mylib.c -shared -o libmylib.so
```

## Package manager

```bash
fz pm add github.com/me/my-lib
fz pm install esp-idf
fz pm list
fz pm remove my-lib
```

## Security & build quality

### SBOM

```bash
fz sbom
```

### Audit

```bash
fz audit
fz audit -json | tee audit_report.json
```

### Reproducible build

```bash
fz -dir ./src --reproducible
```

### Verify source tree

```bash
fz verify
```

## Build profiler

```bash
fz bench
fz bench -n 5 -json
```

## Project contributor support

```bash
fz contribute
```

## WebAssembly

### WASI (via Zig)

```bash
fz -cc main.c -zig -target wasm32-wasi -out main.wasm
wasmtime main.wasm
```

### Browser (via Emscripten)

```bash
source /path/to/emsdk/emsdk_env.sh
fz -cc main.c -target wasm32-emscripten -out main.js
```

## Development workflow extras

### Watch mode

```bash
fz -dir ./kernel -watch
```

### Clean

```bash
fz -dir . -clean
```

### Update with rollback

```bash
sudo fz -update
# If something breaks:
sudo cp /usr/local/bin/fz.old /usr/local/bin/fz
```

