# 🧠 TODO – fz (next steps after v1.5.0)

## 🔜 Short term (v1.6.0 – v1.7.0)

### Core features

- [x] **Exclude patterns**  
  - `exclude` field in `.fz.yaml`  
  - `-exclude` CLI flag (repeatable) – *CLI flag not yet, only config*

- [ ] **C++ support**  
  - auto‑detect `.cpp`, `.cc`, `.cxx`  
  - compile with `g++` (same strict warnings: `-Wall -Wextra -Werror -Wpedantic -Wshadow -Wconversion`)  
  - CLI flag `-cpp <file>`

- [ ] **Custom user flags**  
  - `-asm-flag` (repeatable)  
  - `-cc-flag` (repeatable)  
  - `-ld-flag` (repeatable)

- [ ] **Build time report**  
  - `-time` flag → per‑file and total timings (assembly/compilation + linking)

### Diagnostics & user experience

- [ ] **Colored build output**  
  - success (green), error (red), warning (yellow), info (cyan)  
  - respect `NO_COLOR` environment variable

- [ ] **Verbose per‑file progress**  
  - `[1/5] assembling boot.asm` even without `-verbose` (just a simple counter)

- [ ] **Better error messages**  
  - suggest fixes: missing `#include`, undefined `main`/`_start`, missing `libc`

- [ ] **Adjustable watch debounce**  
  - `-watch-delay <ms>` (default 500) – allow user to tune reaction time

## ⚙️ CI / testing / packaging

- [ ] **GitHub Actions CI**  
  - test on Linux (amd64, arm64), macOS, Windows (cross‑compile) for every push

- [ ] **Raise test coverage**  
  - goal: >80% overall (currently ~70%, linker 40% needs mock improvements)

- [ ] **Shell completions**  
  - generate and install for bash, zsh, fish

- [ ] **Man page installation**  
  - `make install-man` to copy `fz.1` into `/usr/share/man/man1/`

- [ ] **Homebrew tap**  
  - macOS formula – `brew install forgezero-cli/fz/fz`

- [ ] **AUR package**  
  - Arch Linux – `yay -S fz`

## 🚀 Advanced / experimental

- [ ] **Cross‑platform object formats**  
  - `-target` flag → `elf64` (default), `macho64` (macOS), `win64` (Windows)

- [ ] **Linker script support**  
  - `-script <file>` – passes `-T <file>` to `ld`

- [ ] **LTO (Link Time Optimisation)**  
  - `-flto` flag → enable link‑time optimisation for C/C++

- [ ] **Incremental linking**  
  - keep object files even when source list order changes (smart caching)

- [ ] **Remote cache**  
  - store cache in S3 / GCS for CI sharing (optional feature)

- [ ] **Hot reload**  
  - restart built binary automatically after successful rebuild (useful for servers)

## 📚 Documentation & examples

- [ ] **Example projects**  
  - bare‑metal bootloader (x86_64)  
  - mixed C + asm demo (with `-cc` and `-asm`)  
  - static library creation and linking

- [ ] **Benchmark suite**  
  - realistic large project to measure cache performance and build speed over time

- [ ] **Official website**  
  - minimal static page with quick start, config reference, and examples

## 💡 Ideas for v2.0

- [ ] **Plugin system** – custom assembler/compiler backends
- [ ] **Language server** – LSP for assembly (syntax checking, navigation)
- [ ] **Graphical dashboard** – TUI to monitor build progress and cache hits
