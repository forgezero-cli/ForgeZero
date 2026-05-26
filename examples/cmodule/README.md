cmodule — experimental

Architecture
dlopen/dlsym bridge loads an external ELF shared object and resolves a direct entry symbol. The host passes a pointer to `fz_context_t` for plugin initialization. The loader exposes raw symbol lookup and a helper to invoke the plugin entry.

The ABI
```c
typedef struct {
    const char* plugin_path;
    const char* config_path;
    const char* source_path;
    const char* dir_path;
    const char* out_bin;
    const char* out_obj;
    const char* build_type;
    const char* target;
    const char* toolchain;
    const char* mode;
    const char* cc_flags;
    const char* ld_flags;
    const char* format;
    const char* isolation;
    const char** source_dirs;
    int source_dir_count;
} fz_context_t;
```

Build
```sh
fz -dir ./c_src/ -out libfz_example.so 
fz -verify libfz_example.so 

```

Safety
Memory safety and pointer validity are the module's responsibility.
