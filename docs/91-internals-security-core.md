# Aegis Security Core

This part documents how ForgeZero hardens subprocess execution and secure file output.

---

## Hardened command execution: `RunCommand`

Every external tool invocation must go through `utils.RunCommand`.

Main stages:

- validate CLI args (reject injection primitives)
- resolve executables via `exec.LookPath`
- sanitize/validate arguments again
- use fixed environment variables (`LC_ALL=C`, `TZ=UTC`, etc.)
- set execution root as the working directory

---

## Atomic writes: `SecureWriteFile`

Atomic write pipeline (conceptual):

1) create temp file in target directory (same volume)
2) set permissions (0600)
3) write full payload
4) close and flush
5) atomic rename into place
6) set final permissions

Files written through this path include manifests, config updates, SBOM outputs, doctor probes, and templates.

---

## Constant-time tool checksum checks

Optional per-tool BLAKE3 expectations can be stored in config.

Comparison uses constant-time equality (`crypto/subtle.ConstantTimeCompare`).

