# System Self-Audit (`fz doctor`) & Testing Standards (Aegis)

## `fz doctor`

Pre-flight diagnostic. It checks whether the current machine satisfies ForgeZero requirements.

### Invocation

```bash
fz doctor
fz doctor -root ./myapp
fz doctor -json
```

### Audit pipeline (four stages)

- Toolchain reachability (required tools like `zig`, `fasm` on non-Windows)
- Recursive permission audit (writes a probe, then reads regular files via verified open)
- Platform integrity (GOOS/GOARCH, filesystem implementation, execution root, CPU count)
- Health aggregation (fails if required tools missing or unreadable/unwritable)

### JSON output usage

```bash
fz doctor -json | jq -e '.healthy'
```

---

## Testing standards

Aegis security changes are tested via:

- `go test ./...` and `-race`
- coverage targets per internal package
- fault injection via `fs.Mock`
- strict zero-allocation enforcement in hot paths where applicable
- strict `golangci-lint` compliance

