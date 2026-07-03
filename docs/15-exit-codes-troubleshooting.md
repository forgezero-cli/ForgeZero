# Exit Codes & Troubleshooting

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success; or `fz verify` / `fz audit` clean; or `fz doctor` healthy |
| `1` | Build error; duplicate global symbol; verify/audit/doctor failures |
| `2` | Argument/config/toolchain error |

---

## Troubleshooting

### `fz: command not found`

Ensure Go bin dir is on PATH:

```bash
texport PATH="$PATH:$(go env GOPATH)/bin"
```

### `nasm: command not found`

Install via your OS package manager.

### `fasm: command not found`

Download and install from flatassembler.net.

### `g++: command not found`

Install C++ compiler (`g++` or `gcc-c++`).

### `clang: command not found` (Objective-C)

Install `clang` (or ensure Xcode Command Line Tools on macOS).

### `zig: command not found`

Install Zig and add it to PATH.

### Cross-compiler not found (without Zig)

Install the prefixed cross toolchain, or switch to Zig backend.

### Duplicate global symbol error

Fix conflicting `global` declarations, or (temporarily) disable pre-link symbol check:

```bash
fz -dir ./src -no-symbol-check
```

### Sanitizer runtime failure

Fix the reported memory/UB issue, or temporarily disable:

```bash
fz -cc main.c -sanitize=false
```

### `fz verify` reports MODIFIED

Re-generate the manifest from the known-good state:

```bash
fz verify --generate
```

