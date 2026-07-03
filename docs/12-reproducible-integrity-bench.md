# Reproducible Builds, Verify, and Build Profiling

## Reproducible builds (`--reproducible`)

When `--reproducible` is active, ForgeZero removes common non-determinism sources:

- disables random build IDs
- sets `SOURCE_DATE_EPOCH` based on last Git commit timestamp
- normalizes absolute paths in debug info
- sorts objects before linking

```bash
fz -dir ./src --reproducible
```

### Verifying reproducibility

```bash
# Machine A
fz -dir ./src --reproducible -out release_a
sha256sum release_a

# Machine B
fz -dir ./src --reproducible -out release_b
sha256sum release_b
```

Hashes must match.

---

## Source Tree Integrity (`fz verify`)

Generate manifest:

```bash
fz verify --generate
fz verify --generate -manifest ./release.manifest
```

Verify:

```bash
fz verify
fz verify --strict
fz verify -manifest ./release.manifest
```

Categories:

- MODIFIED
- MISSING
- UNTRACKED (only with `--strict`)

---

## Build profiler (`fz bench`)

```bash
fz bench
fz bench -dir ./src
fz bench -n 5
fz bench -json
```

---

## Related flags

- `-json` (build report JSON)
- `-debug` (debug symbols)
- `-verbose` (print invoked external commands)

