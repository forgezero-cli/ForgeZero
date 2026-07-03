# WebAssembly (WASM)

## Target triples

| Target triple | Runtime | Use case |
|---|---|---|
| `wasm32-emscripten` | Emscripten / Browser | Full libc emulation + JS glue |
| `wasm32-wasi` | Wasmtime, WasmEdge, WAMR | Server-side / cloud-native |

---

## WASI via Zig (recommended)

No extra SDK required.

```bash
fz -cc main.c -zig -target wasm32-wasi -out main.wasm
wasmtime main.wasm
```

### WASM config example

```yaml
source_dirs:
  - src
output: mymodule.wasm
backend: zig
target: wasm32-wasi
sanitize: false
```

> Note: ASan/UBSan are automatically disabled for `wasm32-*` targets.

---

## Emscripten

```bash
source /path/to/emsdk/emsdk_env.sh
fz -cc main.c -target wasm32-emscripten -out main.js
# Produces main.wasm + main.js
```

