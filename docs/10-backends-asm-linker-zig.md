# Assembler Backends & Zig Toolchain Backend

## Assembler backends

### NASM (.asm)

Command:

```bash
nasm -felf64 <file> -o <output.o>
```

Example:

```bash
fz -asm hello.asm
```

---

### GAS (.s / .S)

Command:

```bash
gcc -c <file> -o <output.o>
```

Example:

```bash
fz -asm hello.s
```

---

### FASM (.fasm)

Command:

```bash
fasm <file> <output.o>
```

ForgeZero can inject `format ELF64 executable` automatically when no `format` directive exists.

---

## Zig Toolchain Backend (`-zig`)

Zig bundles libc and sysroots for supported targets, allowing cross-compilation without installing a prefixed toolchain.

### Activate Zig backend

```bash
fz -cc main.c -zig
fz -dir ./src -zig
fz -cc main.c -zig -target aarch64-linux-musl
```

### Compiler selection logic

- Default: `gcc/clang`, `g++/clang++`, Objective-C via `clang`
- With `-zig`: `zig cc` / `zig c++` (and Obj-C via `zig cc`)

### Installing Zig

Examples (package managers may vary):

- Debian/Ubuntu: download and extract Zig, then add to `PATH`
- Arch/macOS: use `pacman` / `brew`

---

## FASM improvements (when using ForgeZero)

- Automatic ELF64 `format` injection for `.fasm` if omitted
- When `-debug` is active, extra `-dDEBUG=1` is passed to FASM

