# Virtual Filesystem Layer (Aegis)

> Package: `internal/fs`

## Goals

Centralize security-sensitive filesystem operations to:

- enable deterministic tests via fault injection
- mitigate symlink races and TOCTOU issues
- isolate OS-specific behavior behind a stable interface

---

## Design: `FileSystem` interface

A production build uses a `FileSystem` implementation selected by build tags.

Key operations include:

- verified open (`OpenVerified`)
- secure reads/writes
- temp files + atomic rename
- stat/lstat, symlink handling
- fault injection via `fs.Mock`

---

## TOCTOU mitigation: `OpenVerified` (Unix)

Conceptual algorithm:

1. `Lstat(path)` to inspect metadata without following a final symlink
2. reject symlinks
3. `Open(path)`
4. compare metadata after open (`SameFile`) to ensure the target didn’t change

---

## Windows backend

Windows uses a separate implementation module.

Paths are normalized consistently (drive letters, slashes, UNC).

---

## Mock implementation

`fs.Mock` can inject errors per operation to simulate:

- disk full
- permission errors
- timeouts
- symlink policy failures
- path-changed (TOCTOU detection)

---

## Consumers

Aegis is used by security-sensitive operations across internal packages:

- `SecureWriteFile`
- SBOM and manifest output
- doctor probe writes
- integrity verification

