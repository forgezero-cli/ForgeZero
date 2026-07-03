# ForgeZero Installation

> This guide is a cleaned, structured version of the original `docs/README.md`.

## Requirements: Assembler and compiler tools

| Source type              | Required tool          | Notes |
|--------------------------|------------------------|-------|
| `.asm`                   | `nasm`                 | x86/x86-64 Intel syntax |
| `.s` / `.S`              | `gcc` (drives `as`)    | AT&T syntax; `.S` files are C-preprocessed first |
| `.fasm`                  | `fasm`                 | Must be downloaded separately from flatassembler.net |
| `.c`                     | `gcc` or `clang`       | Strict flags + sanitizers by default |
| `.cpp` / `.cc` / `.cxx`  | `g++` or `clang++`     | Same strict flags as C; `clang++` preferred in strict mode |
| `.m`                     | `clang` or `zig cc`    | Objective-C; automatic framework linking (v4.6.0) |
| `.glo`                   | built-in (HADES)       | Gloria; compiles to raw x86-64 ELF, no external tools required |

## Linker tools

| Linker  | Required for |
|---------|--------------|
| `gcc`   | Default linking, C runtime support |
| `ld`    | Raw linking (`-mode raw`), linker scripts; direct invocation with `-ld` flag (v4.1.0) |
| `clang` | Strict sanitizer mode (`-strict`); Objective-C compilation |
| `ar`    | Static library mode (`-type static`) |

## Cross-compilation tools (optional)

When using `-target <triple>`, `fz` looks for prefixed toolchain binaries on your `PATH`:

| Target triple            | Expected compiler prefix     |
|--------------------------|------------------------------|
| `arm-linux-gnueabihf`    | `arm-linux-gnueabihf-gcc`    |
| `aarch64-linux-gnu`      | `aarch64-linux-gnu-gcc`      |
| `riscv64-linux-gnu`      | `riscv64-linux-gnu-gcc`      |
| `riscv64-linux-musl`     | `riscv64-linux-musl-gcc` or `-zig` (v4.5.1) |
| `x86_64-linux-gnu`       | `x86_64-linux-gnu-gcc`       |

Install cross-compilers via your package manager (e.g. `sudo apt install gcc-arm-linux-gnueabihf`).

When using `-zig`, no prefixed toolchain is required — Zig resolves the target internally.

## Optional tools (used internally)

| Tool        | Purpose |
|-------------|---------|
| `nm`        | Pre-link duplicate symbol check (primary) |
| `objdump`   | Fallback for symbol check |
| `readelf`   | Second fallback for symbol check |
| `git`       | Required for `fz pm add` |
| `zig`       | Required for `-zig` backend (v3.0.0+) |
| `emcc`      | Required for `wasm32-emscripten` target (v3.0.0+) |
| `hyperfine` | Benchmarking (`fz bench` and `bench.sh`) — optional but recommended |

## Go version

Go **1.21** or later is required to build `fz` from source.

---

## Linux — Debian / Ubuntu

```bash
sudo apt update
sudo apt install -y nasm gcc binutils git
sudo apt install -y clang   # required for Objective-C; optional for -strict

# Optional cross-compilers
sudo apt install -y gcc-arm-linux-gnueabihf
sudo apt install -y gcc-aarch64-linux-gnu
sudo apt install -y gcc-riscv64-linux-gnu

# Optional Zig
wget https://ziglang.org/download/0.13.0/zig-linux-x86_64-0.13.0.tar.xz

tar -xf zig-linux-x86_64-0.13.0.tar.xz
sudo mv zig-linux-x86_64-0.13.0 /opt/zig
echo 'export PATH="$PATH:/opt/zig"' >> ~/.bashrc
source ~/.bashrc
zig version

# Optional FASM
wget https://flatassembler.net/fasm-1.73.32.tgz
tar -xzf fasm-1.73.32.tgz
sudo cp fasm/fasm /usr/local/bin/
chmod +x /usr/local/bin/fasm

# Install ForgeZero
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest

# Ensure Go bin dir is on PATH
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc

fz -v
```

---

## Linux — Fedora / RHEL / CentOS

```bash
# Fedora
sudo dnf install -y nasm gcc binutils clang git

# RHEL / CentOS (enable EPEL for nasm)
sudo dnf install -y epel-release
sudo dnf install -y nasm gcc binutils clang git

# Optional Zig
wget https://ziglang.org/download/0.13.0/zig-linux-x86_64-0.13.0.tar.xz

tar -xf zig-linux-x86_64-0.13.0.tar.xz
sudo mv zig-linux-x86_64-0.13.0 /opt/zig
echo 'export PATH="$PATH:/opt/zig"' >> ~/.bashrc

# Install ForgeZero
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
```

---

## Linux — Arch Linux / Manjaro

```bash
sudo pacman -S --noconfirm nasm gcc binutils clang git zig

# FASM is available in the AUR
yay -S fasm

go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
```

---

## Linux — openSUSE

```bash
sudo zypper install -y nasm gcc binutils clang git

go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
```

---

## macOS

macOS support is in progress.

```bash
# Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Dependencies
brew install nasm gcc go git zig

# Install ForgeZero
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest

# Add Go bin dir to PATH
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc

fz -v
```

> Note: the Darwin syscall layer is currently stubbed — builds succeed but runtime behavior is untested.

---

## Windows

As of **v3.1.0 Aegis**, ForgeZero includes a native Windows filesystem backend and Windows-specific atomic rename retry logic.

As of **v4.1.0 Citadel**, platform-specific syscall drivers are implemented via `golang.org/x/sys/windows`.

### Option A — WSL2 (recommended)

Follow the Debian/Ubuntu instructions.

### Option B — Native Windows (experimental, MSYS2)

1) Install MSYS2

```bash
pacman -Syu
pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-binutils mingw-w64-x86_64-clang git
```

2) Install NASM for Windows and add to PATH.

3) Install Go for Windows.

4) Install ForgeZero:

```powershell
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
```

5) Run:

```powershell
fz doctor
```

---

## Build from source (all platforms)

```bash
git clone https://github.com/forgezero-cli/ForgeZero.git
cd ForgeZero

go build -o fz ./cmd/fz/main.go    # Linux/macOS
go build -o fz.exe ./cmd/fz/main.go  # Windows

go test ./...
go test ./internal/... -cover
```

---

## Go Install

```bash
go install github.com/forgezero-cli/ForgeZero/cmd/fz@latest
fz -v
fz --version
```

