# Gloria Language

## Overview

Gloria is ForgeZero's integrated systems programming language.

- Compiles directly to raw x86-64 ELF binaries via the HADES engine
- Designed for bare-metal targets, OS kernels, firmware, and other environments
- Typical output size: **69–125 bytes**
- No runtime dependencies

Key features:

- Bare-metal VGA framebuffer output via `print()` in kernel mode
- x86-64 I/O port access via `in8()` / `out8()`
- Arbitrary memory read/write via `peek()` / `poke()`

---

## Syntax reference

```go
fn main() {
    let a = 10
    let b = 20
    let c = a + b
    print("result computed")
}
```

Supported constructs include:

- `let` variable declarations
- arithmetic: `+`, `-=`, `*=`, etc.
- function definition: `fn name(a, b) { ... }`
- function call: `let r = add(1, 2)`
- conditional: `if x { ... }`
- loop: `while x { ... }`
- print: `print("hello")`

Memory & I/O:

- `peek(address)` / `poke(address, value)`
- `in8(port)` / `out8(port, value)`

---

## Control flow

### `if`

Executes the body when the evaluated variable is non-zero.

```go
let x = 1
if x {
    print("x is set")
}
```

### `while` (JIT features)

`while` iterates while the condition variable is non-zero.
Supports `=`, `+=`, `-=` assignment and builtin calls inside the loop body.

```go
let i = 5
while i {
    print("counting")
    i -= 1
}
```

---

## Memory access & I/O ports

### `peek(address)`

Reads a 16-bit value from a memory address.
`address` can be an integer literal or a variable.

```go
let val = peek(0xB8000)
let addr = 0x1000
let x = peek(addr)
```

### `poke(address, value)`

Writes a 16-bit value to a memory address.

```go
poke(0xB8000, 0x0741)   // write 'A' with white-on-black to VGA
let pos = 2
poke(pos, 0x0742)
```

### `in8(port)` / `out8(port, value)`

I/O port primitives on x86-64:

```go
let scancode = in8(0x60)
out8(0x3F8, 65)   // write 'A' on COM1
```

---

## VGA framebuffer output

When compiled in bare-metal (kernel) mode, `print()` writes directly to `0xB8000`.

Behavior notes:

- Register **R15** is reserved as the VGA cursor offset and preserved across calls
- String escape sequences `\n` and `\t` are resolved at compile time

```go
fn main() {
    print("Booting...\n")
    print("OK\t[done]")
}
```

In userspace mode (default), `print()` uses Linux `sys_write` (fd 1).

---

## Register constants

Gloria programs have named constants for all 16 x86-64 general-purpose registers.

| Constant | Register |
|----------|----------|
| `regRAX` | RAX |
| `regRCX` | RCX |
| `regRDX` | RDX |
| `regRBX` | RBX |
| `regRSP` | RSP |
| `regRBP` | RBP |
| `regRSI` | RSI |
| `regRDI` | RDI |
| `regR8`  | R8 |
| `regR9`  | R9 |
| `regR10` | R10 |
| `regR11` | R11 |
| `regR12` | R12 |
| `regR13` | R13 |
| `regR14` | R14 |
| `regR15` | R15 |

`R15` is reserved by the Gloria runtime in kernel mode.

---

## Compilation pipeline

```bash
fz -gloria main.glo
./main

fz -gloria main.glo -verbose
fz -gloria main.glo -out kernel.elf
```

Pipeline stages:

1. Lexer (CPU mnemonic disambiguation vs labels)
2. Parser (AST validation; `ErrMalformedAST` on malformed input)
3. Codegen (HADES, zero-allocation stack buffers)
4. ELF emission (deterministic `.symtab` ordering and relocation offsets)

The entire pipeline runs in a single process with zero intermediate files on disk.

