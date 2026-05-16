# 🌱 ForgeZero (fz) v1.0-release
*ForgeZero* is a high-performance assembler builder now fully powered by *Go*. 
It’s designed to be a "Swiss Army knife" for assembly developers, providing a seamless experience for compiling and linking without the headache of complex Makefiles.

fz is a command-line tool that assembles and links assembly source files using standard system tools (*NASM*, *FASM*, *GCC*, *LD*). It supports three syntaxes: *NASM* (.asm), *GAS* (.s/.S), and *FASM* (.fasm).

# NO DEPENDENCY / NO PROBLEMS
> Lightweight, fast, and written in Go. No more Node.js overhead!

## ✨ Features

- *Smart Detection:* Automatically identifies `.asm`, `.s`, or `.fasm` files.
- *NASM, GAS & FASM Support:* Works with the most popular assemblers out of the box.
- *Smart Naming:* Automatic output binary naming based on the source file.
- *Debug Ready:* Dedicated `--debug` flag (passes `-g` to the assembler).
- *Cross-Platform Linking:* Optimized for Linux, and ready for Windows/macOS.
- *Flexible Linking:* Support for `-l` flags and include directories.
- *Clean & Verbose:* Clean build options and verbose mode for debugging your build process.
- *Linking Modes:* Choose between `auto`, `raw` (ld), or `c` (gcc) linking.
- *GDB Support:* (Coming soon).

## 🛠 Requirements

Make sure the required tools are installed and available in your PATH:

| Source type | Required tool | Installation example (Ubuntu/Debian) |
|-------------|---------------|--------------------------------------|
| .asm        | nasm          | `sudo apt install nasm`              |
| .s / .S     | gcc (as)      | `sudo apt install gcc`               |
| .fasm       | fasm          | Download from [flatassembler.net](https://flatassembler.net) |

*For linking, `gcc` or `ld` is required. `gcc` is recommended for C runtime integration.*

## 🚀 Installation

Clone the repository and build:
```bash
git clone https://github.com/alexvoste/ForgeZero.git
cd ForgeZero
go build -o fz ./cmd/fz/main.go
```

MOVE IT TO YOUR PATH

```bash
sudo mv fz /usr/local/bin/
```


### Via Go Install (The Easy Way)
```bash
go install github.com/alexvoste/ForgeZero/cmd/fz@latest
```

## Usage 

```bash 
fz[options] -asm <source-file>
```

## Basic usage 

```bash
./fz -asm program.asm 
```


## Common options
```
Option              |       Description
-asm <file>	        | Path to assembly source file (required)
-out <name>	        | Output binary name (default: derived from source base name)
-out-obj <name>	    | Output object file name (default: <basename>.o)
-debug	            | Generate debug information (passes -g to assembler)
-verbose	          | Print every external command before running
-mode <auto|raw|c>	| Linking mode: auto (gcc → gcc -no-pie → ld), c (gcc only), raw (ld only)
-timeout <sec>	    | Timeout in seconds for each command (default 60)
-version	          | Show version and exit
```



## Examples

Assemble and link a NASM file with debug info:

```bash
./fz -asm hello.asm -debug -verbose
```

Force raw linking with ld (no libc):

```bash
./fz -asm boot.asm -mode raw -out kernel.bin
```

Use a custom object name and output binary:

```bash
./fz -asm code.s -out-obj tmp.o -out myprog
```

Supported source formats

    NASM (.asm): uses nasm -felf64

    GAS (.s, .S): uses gcc -c

    FASM (.fasm): uses fasm (flat assembler)

Exit codes

    0 – success

    1 – assembly or linking error

    2 – invalid arguments or missing source file



For complete information, use the -h flag to display the help. 


## 📝 MIT License 

> If you like this project, please consider giving it a *star* ⭐️ in the repository! I'm actively working on it, so feel free to open *Issues* or submit *Pull Requests*.

I hope your assembly language programs run faster than ever.

*Author:* [AlexVoste](https://github.com/alexvoste)

