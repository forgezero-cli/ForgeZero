# Cross-Compilation

Added in v1.9.0, extended with Zig backend in v3.0.0, validated across targets in v4.1.0, and extended to `riscv64-linux-musl` in v4.5.1.

## Basic usage

```bash
fz -cc main.c -target arm-linux-gnueabihf
fz -cc main.c -target aarch64-linux-gnu
fz -cc main.c -target riscv64-linux-gnu
fz -dir ./src -target arm-linux-gnueabihf -out firmware
```

With Zig backend (no external cross toolchain packages needed):

```bash
fz -cc main.c -zig -target aarch64-linux-musl
fz -cc main.c -zig -target riscv64-linux-musl
```

---

## How it works

### Without `-zig`

ForgeZero constructs prefixed toolchain binary names by prepending the triple:

- Compiler: `<triple>-gcc`
- C++ compiler: `<triple>-g++`
- Linker: `<triple>-gcc` or `<triple>-ld` (depending on linking mode)
- Archiver: `<triple>-ar` when building static libraries

### With `-zig`

The triple is passed directly to `zig cc -target`.

Zig ships libc variants (musl/glibc) and WASI sysroot internally, making it recommended for static musl targets including `riscv64-linux-musl`.

---

## Installing cross-compilers (without Zig)

Debian / Ubuntu:

```bash
sudo apt install gcc-arm-linux-gnueabihf
sudo apt install gcc-aarch64-linux-gnu
sudo apt install gcc-riscv64-linux-gnu
```

Fedora:

```bash
sudo dnf install gcc-arm-linux-gnu
sudo dnf install gcc-aarch64-linux-gnu
```

Arch Linux:

```bash
sudo pacman -S arm-linux-gnueabihf-gcc
sudo pacman -S aarch64-linux-gnu-gcc
```

---

## Cross-compilation with a config file

```yaml
# .fz.yaml
source_dirs:
  - src
output: firmware.elf
target: arm-linux-gnueabihf
mode: raw
flags:
  cc:
    - -mcpu=cortex-m4
    - -mfpu=fpv4-sp-d16
    - -mfloat-abi=hard
    - -ffreestanding
  ld:
    - -T
    - linker.ld
```

