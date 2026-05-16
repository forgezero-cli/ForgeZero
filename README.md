
# 🌱 ForgeZero (fz)

A unique builder for your ASM projects, built using Node.js technology.
This builder automatically detects .asm files, making it very convenient to compile your project.

fz is a command-line tool that assembles and links assembly source files using standard system tools (NASM, FASM, GCC, LD).  
It supports three syntaxes: **NASM** (`.asm`), **GAS** (`.s`/`.S`), and **FASM** (`.fasm`).

# NO DEPENDENCY / NO PROBLEMS

> Supports NASM and GNU Assemblers!

## Features

- Automatic detection of .asm or .s files based on assembler
- Support for NASM and GAS assemblers also FASM 
- Automatic output naming from input file
- Debug for FASM files, option --debug 
- Cross-platform linking (Linux, Windows, macOS)
- Library linking with -l flags
- Include directories support
- Clean build option
- Verbose mode
- Support for the Go language. Some modules have already been successfully rewritten in this language!
- GDB support (coming soon)

## Requirements

Make sure the required tools are installed and available in your `PATH`:

| Source type | Required tool | Installation example (Ubuntu)     |
|-------------|---------------|------------------------------------|
| `.asm`      | `nasm`        | `sudo apt install nasm`            |
| `.s` / `.S` | `gcc`         | `sudo apt install gcc`             |
| `.fasm`     | `fasm`        | download from https://flatassembler.net |

For linking, `gcc` or `ld` is required. `gcc` is recommended for C runtime integration.

## Installation 
Clone the repository and build:

```bash
git clone https://github.com/alexvoste/ForgeZero.git
cd go/
go build -o fz ./cmd/fz 
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


## MIT License 

> If you like this project, please consider giving it a star in the repository as a token of your appreciation =) I’d be happy to keep working on it, and feel free to point out any issues in the repository comments (issues)! I’d also love to see your code contributions. 

I hope your assembly language program will run faster. 

**Author:** AlexVoste
