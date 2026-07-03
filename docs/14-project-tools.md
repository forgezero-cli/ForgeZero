# Project Tools: init, contribute, IDE, self-update

## Project initialization (`fz -init`)

Scaffolds a new project without overwriting existing files.

```bash
mkdir myproject && cd myproject
fz -init
```

Creates:

- `.fz.yaml`
- `.fzignore`
- `README.md`

---

## Contributor guidance (`fz contribute`)

Generates `CONTRIBUTING_USER.md` in the current working directory.

```bash
fz contribute
```

The output includes environment diagnostics (via `fz doctor`) and project-specific build/test recommendations.

---

## LSP & IDE integration (`-compile-commands`)

```bash
fz -compile-commands
fz -dir ./src -compile-commands
```

Produces `compile_commands.json` for clangd and other tooling.

---

## Self-update (`fz -update`)

```bash
fz -update
```

Fetches and installs the latest `fz` binary and backs up the previous binary as `fz.old`.

Rollback:

```bash
sudo cp /usr/local/bin/fz.old /usr/local/bin/fz
```

