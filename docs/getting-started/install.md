# Installation

The repository includes a simple installation flow for local use.

## Typical installation path

- Run the repository install script if you want a packaged local setup.
- Or build the CLI directly with Go:

```bash
go build -o fz ./cmd/fz
```

## Notes

The build system expects a working toolchain for the languages you intend to compile. For assembly and C-style projects, that usually means having the relevant assembler and compiler frontends available on the PATH.
