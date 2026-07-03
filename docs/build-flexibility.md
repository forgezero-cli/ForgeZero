# Build Flexibility: ISO, Scripts, Hooks, Rollback

This document covers the flexibility layer added on top of the core build so a
project can be described entirely through `.fz.yaml` and a handful of flags,
without external Makefiles/CMake/Ninja.

## Custom scripts and hooks (cross-platform)

Post-build scripts (`scripts:`) and build hooks (`hooks:`) now resolve the shell
per platform instead of hard-coding `sh -c`:

- Unix: `$SHELL` (fallback `/bin/sh`) with `-c`
- Windows: `%COMSPEC%` (fallback `cmd.exe`) with `/C`

Empty commands are skipped and every failure is wrapped with the offending
command for a precise error.

```yaml
scripts:
  - strip --strip-all app
  - sha256sum app > app.sha256

hooks:
  pre_build:
    - cmd: ./generate_headers.sh
      critical: true
  on_failure: ./notify_failure.sh
```

## ISO image generation

### Flag

```
-iso[=<dir>]     enable the ISO stage; optional source directory override
-iso-out <file>  output path for the generated ISO
-iso-hybrid      make the ISO hybrid (BIOS + USB bootable)
```

`-iso` is a boolean-style flag: use `-iso` to enable using config values, or
`-iso=./isoroot` to also set the source directory in one token.

### Config section

Everything the flag exposes (and more) is configurable under `iso:`:

```yaml
iso:
  enabled: true
  source_dir: isoroot
  output: live.iso
  volume_id: FORGEZERO
  boot_image: boot/grub/eltorito.img
  boot_catalog: boot/boot.cat
  boot_load_size: "4"
  no_emul_boot: true
  boot_info_table: true
  joliet: true
  rock_ridge: true
  hybrid: true
  custom_args:
    - -graft-points
```

### Resolution order

Source directory: `-iso=<dir>` → `iso.source_dir` → `source_dir` → `-dir` → `.`
Output path: `-iso-out` → `iso.output` → `<output>.iso` → `output.iso`
Hybrid: `-iso-hybrid` OR `iso.hybrid`

The stage runs only after a successful build (and after `scripts:`), so scripts
can populate the ISO root first. It requires one of `xorriso`, `genisoimage`, or
`mkisofs`; the discovered tool path is cached in memory for the process lifetime.

## Rollback

```
-rollback            restore the previous binary saved by -update
-rollback-to <ver>   download and install a specific release version
```

`-update` keeps the replaced binary as `<exe>.old`. `-rollback` atomically swaps
the running binary with that backup (and rotates the backup so the operation is
reversible), fully offline. `-rollback-to <ver>` reuses the same safe,
cross-platform download/install path as `-update` (temp file in the executable
directory, backup, atomic rename, mode preservation).
