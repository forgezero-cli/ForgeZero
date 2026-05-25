cmodule — experimental

Architecture
dlopen/dlsym bridge loads an external ELF shared object and resolves a single exported module descriptor. The host passes an opaque pointer to the module entry for zero-copy interaction. The loader exposes raw symbol lookup and a helper to invoke the module descriptor entry.

The ABI
```c
typedef struct {
    void* host;
    int64_t id;
} fz_context_t;
```

Build
```sh
gcc -fPIC -shared -o libfz_example.so c_src/fz_example.c -I../../internal/cplugin
CGO_ENABLED=1 go run main.go
```

Safety
Memory safety and pointer validity are the module's responsibility.
