#ifndef FZ_MODULE_H
#define FZ_MODULE_H

typedef void (*fz_entry_t)(void*);

typedef struct {
    const char* name;
    int version;
    fz_entry_t entry;
} fz_module_info;

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

#endif
