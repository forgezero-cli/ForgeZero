## Rollback package

### Description

Updates can introduce regressions. This package lets a user return to a known
good binary, either the one they had before the last update or an explicit
release version, without external tools and on every supported platform.

It is a thin, well-tested layer over `internal/updater`, so downloads and
installs reuse the same safe machinery as `-update` (temp file in the executable
directory, backup, atomic rename, mode preservation, size limits, host
validation). It does not shell out to `curl` and does not assume `/usr/bin`.

### API

```go
func Run() error            // restore the previous binary (<exe>.old)
func To(version string) error // download and install a specific release
```

### -rollback

Restores `<exe>.old`, the backup that `-update` leaves behind. The swap is
atomic and reversible: the current binary is rotated into the backup slot, so a
second `-rollback` returns to where you started. Works fully offline.

### -rollback-to <version>

Downloads the requested release for the current OS/arch from the ForgeZero
GitHub releases and installs it over the running binary using the updater's
install path. The leading `v` is optional.

```bash
fz -rollback
fz -rollback-to 5.1.0
```

### Notes

- Installs in place next to the running executable; no hardcoded system path.
- Cross-platform: Linux, macOS, Windows (asset naming from `runtime.GOOS/GOARCH`).
- `To("")` returns an error; `Run()` errors clearly when no backup exists.
- The two package-level function hooks (`restore`, `install`) are overridable in
  tests, so the logic is verified without network or root.

```
 * Copyright (c) 2026 ForgeZero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
```
