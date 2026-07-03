# HADES Engine, Contributing & License

## HADES engine (codegen & ELF emission)

HADES is ForgeZero’s integrated codegen and ELF emission layer.

Key points:

- deterministic ELF64 object emission
- correct `.symtab` ordering (local-before-global)
- deterministic absolute relocation offsets
- strict parser integrity checks (no silent AST corruption)
- zero-allocation hot-path refactors for codegen

---

## Contributing

General rules:

- open an issue first for significant work
- write tests for new behavior
- security-sensitive changes must include failure-path testing
- hot-path changes require benchmark assertions (0 allocs/op)
- run:

```bash
go test ./...
go test ./internal/... -cover
golangci-lint run -E gofmt,govet,staticcheck,unused ./...
```

---

## License

ForgeZero is released under the GPLv3 License.

