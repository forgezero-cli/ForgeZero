# ☘️TODO – fz (future)

## Core

- [ ] **Exclude patterns**  
  - `exclude` field in `.fz.yaml`  
  - `-exclude` CLI flag (repeatable)

- [ ] **C++ support**  
  - auto‑detect `.cpp`, `.cc`, `.cxx`  
  - compile with `g++` (same strict warnings)  
  - flag `-cpp <file>`

- [ ] **Custom user flags**  
  - `-asm-flag` (repeatable)  
  - `-cc-flag` (repeatable)  
  - `-ld-flag` (repeatable)

- [ ] **Build time report**  
  - `-time` flag → per‑file and total timings

## Diagnostics & user experience

- [ ] **Colored build output**  
  - success (green), error (red), warning (yellow), info (cyan)  
  - respect `NO_COLOR`

- [ ] **Verbose per‑file progress**  
  - `[1/5] assembling boot.asm` (even without `-verbose`)

- [ ] **Better error messages**  
  - suggest fixes: missing `#include`, undefined `main`/`_start`, missing `libc`

- [ ] **Adjustable watch debounce**  
  - `-watch-delay <ms>` (default 500)

## CI / testing / packaging

- [ ] **GitHub Actions CI**  
  - test on Linux, macOS, Windows (cross‑compile)

- [ ] **Code coverage**  
  - >80% line coverage, integrated in CI

- [ ] **Shell completions**  
  - bash, zsh, fish

- [ ] **Man page installation**  
  - `make install-man`

- [ ] **Homebrew tap**  
  - macOS formula

- [ ] **AUR package**  
  - `yay -S fz`

## Advanced / experimental

- [ ] **Cross‑platform object formats**  
  - `-target` flag → `elf64`, `macho64`, `win64`

- [ ] **Linker script support**  
  - `-script <file>` (pass `-T` to `ld`)

- [ ] **LTO**  
  - `-flto` flag

- [ ] **Incremental linking**  
  - keep objects even when source list order changes

- [ ] **Remote cache**  
  - store cache in S3 / GCS for CI sharing

- [ ] **Hot reload**  
  - restart binary automatically after successful rebuild

## Documentation & examples

- [ ] **More example projects**  
  - bare‑metal bootloader  
  - mixed C + asm demo  
  - static library

- [ ] **Benchmark suite**  
  - realistic projects to measure cache performance
